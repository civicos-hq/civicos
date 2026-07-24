package classify

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/civicos/civicai-service/internal/gemini"
	"google.golang.org/genai"
)

// Category values MUST stay in sync with community-service domain.IssueCategory.
// If a new category ships, add it here and to the FE enums — Gemini is
// constrained to this exact set via the response schema.
var Categories = []string{
	"INFRASTRUCTURE",
	"HEALTH",
	"EDUCATION",
	"SECURITY",
	"ENVIRONMENT",
	"UTILITIES",
	"TRANSPORT",
	"OTHER",
}

// Severity is a CivicAI-only concept for now — community-service doesn't
// store severity on the issue itself yet. Surfaced in the UI as a hint so
// the reporter can escalate accordingly.
var Severities = []string{"LOW", "MEDIUM", "HIGH", "CRITICAL"}

type ClassifyInput struct {
	Title       string `json:"title" binding:"required,min=3"`
	Description string `json:"description" binding:"required,min=10"`
	// Optional context. Currently unused in the prompt but kept in the
	// request shape so we can add community-scoped priors later
	// (e.g. per-community common categories) without an FE change.
	CommunityID string `json:"communityId"`
}

type ClassifyOutput struct {
	Category      string   `json:"category"`
	Severity      string   `json:"severity"`
	SuggestedTags []string `json:"suggestedTags"`
	// Reasoning is a one-line explanation the FE can show on hover — helps
	// the reporter decide whether to accept the suggestion.
	Reasoning   string    `json:"reasoning"`
	Confidence  float64   `json:"confidence"`
	Model       string    `json:"model"`
	GeneratedAt time.Time `json:"generatedAt"`
}

type Service struct {
	ai *gemini.Client
}

func NewService(ai *gemini.Client) *Service { return &Service{ai: ai} }

const systemInstruction = `You are CivicAI, an AI assistant for the CivicOS civic engagement platform.
Your job is to classify citizen-reported community issues into one of a fixed set of categories.

Rules:
- Pick exactly one category from the allowed enum.
- Pick exactly one severity from the allowed enum. Use HIGH/CRITICAL only when the report indicates immediate risk to life, health, or infrastructure.
- Suggest 2 to 5 short, lowercase, hyphenated tags relevant to the issue (e.g. "burst-pipe", "no-electricity").
- Reasoning is a single sentence, max 25 words, explaining why you chose the category.
- Confidence is 0.0 to 1.0.
- Never invent facts about the issue. Only classify what the reporter wrote.`

// responseSchema pins the shape and enum values so Gemini can never return
// a category or severity we don't understand.
func responseSchema() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"category": {Type: genai.TypeString, Enum: Categories},
			"severity": {Type: genai.TypeString, Enum: Severities},
			"suggestedTags": {
				Type:  genai.TypeArray,
				Items: &genai.Schema{Type: genai.TypeString},
			},
			"reasoning":  {Type: genai.TypeString},
			"confidence": {Type: genai.TypeNumber},
		},
		Required: []string{"category", "severity", "suggestedTags", "reasoning", "confidence"},
	}
}

func (s *Service) Classify(ctx context.Context, in ClassifyInput) (*ClassifyOutput, error) {
	prompt := fmt.Sprintf(
		"Title: %s\n\nDescription: %s",
		strings.TrimSpace(in.Title),
		strings.TrimSpace(in.Description),
	)

	var out ClassifyOutput
	if err := s.ai.GenerateJSON(ctx, systemInstruction, prompt, responseSchema(), &out); err != nil {
		return nil, err
	}

	// Defensive normalization — Gemini honours the enum via schema, but a
	// future SDK version or model swap could regress. Snap-to-nearest
	// beats a downstream 400 at the community-service.
	out.Category = snap(out.Category, Categories, "OTHER")
	out.Severity = snap(out.Severity, Severities, "MEDIUM")
	if out.Confidence < 0 {
		out.Confidence = 0
	}
	if out.Confidence > 1 {
		out.Confidence = 1
	}
	out.Model = s.ai.Model()
	out.GeneratedAt = time.Now().UTC()
	return &out, nil
}

func snap(v string, allowed []string, fallback string) string {
	up := strings.ToUpper(strings.TrimSpace(v))
	for _, a := range allowed {
		if a == up {
			return a
		}
	}
	return fallback
}
