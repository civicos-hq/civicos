// Package draft generates announcement drafts from a short brief. Unlike
// summarize, this endpoint doesn't need to fetch anything from other
// services — the caller supplies the raw material. That keeps it fast and
// makes the "Draft with AI" affordance feel like a simple pass-through.
package draft

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/civicos/civicai-service/internal/gemini"
	"google.golang.org/genai"
)

// Tone controls voice + register. The four values match the choices the
// FE exposes as a segmented control — new tones should ship as an
// intentional product decision, not a free-text field, because Gemini
// output quality is sensitive to a small, tested vocabulary.
var Tones = []string{"formal", "friendly", "urgent", "empathetic"}

// Audience helps the model calibrate assumed context. "members" implies
// the reader already knows the org and receives regular updates; "all"
// implies a broader public who may need more background.
var Audiences = []string{"all", "members"}

type DraftInput struct {
	// Brief is the org admin's rough note about what to announce. 20 chars
	// minimum forces at least a sentence — Gemini's output from three-word
	// briefs is generic and unusable.
	Brief    string `json:"brief" binding:"required,min=20,max=1500"`
	Tone     string `json:"tone" binding:"required,oneof=formal friendly urgent empathetic"`
	Audience string `json:"audience" binding:"required,oneof=all members"`
	// Optional org context — folded into the prompt when present so the
	// draft can address the reader as e.g. "residents of Aguda" instead
	// of the generic "community".
	OrgName string `json:"orgName"`
	OrgKind string `json:"orgKind"`
}

type DraftOutput struct {
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	KeyPoints   []string  `json:"keyPoints"`
	Tone        string    `json:"tone"`
	Audience    string    `json:"audience"`
	Model       string    `json:"model"`
	GeneratedAt time.Time `json:"generatedAt"`
}

type Service struct {
	ai *gemini.Client
}

func NewService(ai *gemini.Client) *Service { return &Service{ai: ai} }

const systemInstruction = `You are CivicAI, an assistant helping an organization admin draft a public announcement for the CivicOS civic engagement platform.

Ground rules:
- Ground every statement in the brief you were given. Do not invent facts, dates, names, quotes, phone numbers, statistics, or promises the admin didn't make.
- Match the requested tone precisely. "urgent" means direct and time-sensitive, not alarmist. "empathetic" acknowledges impact on people before logistics.
- Write for accessibility: short sentences, plain English, no bureaucratic jargon. A grade-8 reading level is a good default.
- The draft is a starting point that a human will edit. Bias toward specificity over hedging.

Produce:
- title: one short, clear headline (max 80 characters). No emoji, no ALL CAPS.
- body: the announcement body, 2 to 5 short paragraphs, plain text (no markdown headings). Include what happened, what the organization is doing, what the reader should do or expect next.
- keyPoints: 2 to 4 bullet-ready one-liners that a distracted reader could scan and still get the gist.`

func responseSchema() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"title":     {Type: genai.TypeString},
			"body":      {Type: genai.TypeString},
			"keyPoints": {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
		},
		Required: []string{"title", "body", "keyPoints"},
	}
}

func (s *Service) Draft(ctx context.Context, in DraftInput) (*DraftOutput, error) {
	prompt := buildPrompt(in)

	type raw struct {
		Title     string   `json:"title"`
		Body      string   `json:"body"`
		KeyPoints []string `json:"keyPoints"`
	}
	var r raw
	if err := s.ai.GenerateJSON(ctx, systemInstruction, prompt, responseSchema(), &r); err != nil {
		return nil, err
	}

	return &DraftOutput{
		Title:       strings.TrimSpace(r.Title),
		Body:        strings.TrimSpace(r.Body),
		KeyPoints:   r.KeyPoints,
		Tone:        in.Tone,
		Audience:    in.Audience,
		Model:       s.ai.Model(),
		GeneratedAt: time.Now().UTC(),
	}, nil
}

func buildPrompt(in DraftInput) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Tone: %s\n", in.Tone)
	fmt.Fprintf(&sb, "Audience: %s\n", audienceLabel(in.Audience))
	if strings.TrimSpace(in.OrgName) != "" {
		kind := ""
		if in.OrgKind != "" {
			kind = fmt.Sprintf(" (%s)", strings.ToLower(in.OrgKind))
		}
		fmt.Fprintf(&sb, "Publishing organization: %s%s\n", in.OrgName, kind)
	}
	fmt.Fprintf(&sb, "\nAdmin brief:\n%s\n", strings.TrimSpace(in.Brief))
	return sb.String()
}

func audienceLabel(a string) string {
	switch a {
	case "members":
		return "members and followers of this organization (assumed familiar with our work)"
	default:
		return "the general public in the community (may not know this organization)"
	}
}
