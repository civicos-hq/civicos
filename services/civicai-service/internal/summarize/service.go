package summarize

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/civicos/civicai-service/internal/gemini"
	"google.golang.org/genai"
)

const (
	// Ceiling on how many comments we send to Gemini in one call. Well
	// below the model's context window — this is a cost + latency guard,
	// not a correctness one. Set to newest-first so an old, cold thread
	// with 500 comments doesn't drown out the recent turn.
	maxComments = 200
	// Threads with fewer than this many comments don't benefit much from a
	// summary and their prompt cost per useful insight is bad. The handler
	// still runs but the FE hides the button in that case.
	minComments = 2
)

type SummarizeInput struct {
	Resource string `json:"resource" binding:"required,oneof=petition issue consultation"`
	ID       string `json:"id" binding:"required"`
}

type Sentiment struct {
	Positive float64 `json:"positive"`
	Neutral  float64 `json:"neutral"`
	Negative float64 `json:"negative"`
}

type SummaryOutput struct {
	Resource           string    `json:"resource"`
	ResourceID         string    `json:"resourceId"`
	Title              string    `json:"title"`
	TLDR               string    `json:"tldr"`
	Themes             []string  `json:"themes"`
	Sentiment          Sentiment `json:"sentiment"`
	TopAsks            []string  `json:"topAsks"`
	RecommendedActions []string  `json:"recommendedActions"`
	CommentsAnalyzed   int       `json:"commentsAnalyzed"`
	OfficialResponders []string  `json:"officialResponders"`
	Model              string    `json:"model"`
	GeneratedAt        time.Time `json:"generatedAt"`
	Cached             bool      `json:"cached"`
}

type Service struct {
	ai     *gemini.Client
	source *SourceClient
	cache  *Cache
}

func NewService(ai *gemini.Client, source *SourceClient, cache *Cache) *Service {
	return &Service{ai: ai, source: source, cache: cache}
}

const systemInstruction = `You are CivicAI, the intelligence layer of the CivicOS civic engagement platform.
An organization admin or elected representative is reviewing a citizen thread and needs a decision-support summary — NOT a chatbot answer.

Ground rules:
- Base every claim on the thread you were given. Never invent facts, names, statistics, or promises.
- Be neutral. Do not take sides between citizens and the organization.
- Prefer specific, actionable language over vague summary-speak.

Produce:
- tldr: 1 to 2 sentences (max 60 words) that a busy admin can read in ten seconds.
- themes: 3 to 6 short noun phrases capturing what the thread is about (e.g. "delayed response times", "safety near school zones").
- sentiment: three numbers between 0 and 1 that sum to roughly 1.0 — the mix of positive, neutral, and negative comments overall.
- topAsks: 3 to 5 concrete things the community is asking for, in the community's own voice where possible.
- recommendedActions: 2 to 4 actions the organization could take, ordered by impact. Each is one sentence, imperative mood ("Publish a repair timeline.").`

func responseSchema() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"tldr":   {Type: genai.TypeString},
			"themes": {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
			"sentiment": {
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"positive": {Type: genai.TypeNumber},
					"neutral":  {Type: genai.TypeNumber},
					"negative": {Type: genai.TypeNumber},
				},
				Required: []string{"positive", "neutral", "negative"},
			},
			"topAsks":            {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
			"recommendedActions": {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
		},
		Required: []string{"tldr", "themes", "sentiment", "topAsks", "recommendedActions"},
	}
}

// Summarize fetches the thread, calls Gemini, caches the result. Returns
// the cached copy if fresh. The caller must supply the requester's JWT so
// the source client can inherit their read permissions.
func (s *Service) Summarize(ctx context.Context, in SummarizeInput, bearerToken string) (*SummaryOutput, error) {
	if cached := s.cache.Get(ctx, in.Resource, in.ID); cached != nil {
		cached.Cached = true
		return cached, nil
	}

	res, err := s.source.Fetch(ctx, in.Resource, in.ID, bearerToken)
	if err != nil {
		return nil, err
	}

	comments := visibleComments(res.Comments)
	prompt := buildPrompt(res, comments)

	type geminiOut struct {
		TLDR               string    `json:"tldr"`
		Themes             []string  `json:"themes"`
		Sentiment          Sentiment `json:"sentiment"`
		TopAsks            []string  `json:"topAsks"`
		RecommendedActions []string  `json:"recommendedActions"`
	}
	var raw geminiOut
	if err := s.ai.GenerateJSON(ctx, systemInstruction, prompt, responseSchema(), &raw); err != nil {
		return nil, err
	}

	out := &SummaryOutput{
		Resource:           in.Resource,
		ResourceID:         in.ID,
		Title:              res.Title,
		TLDR:               strings.TrimSpace(raw.TLDR),
		Themes:             raw.Themes,
		Sentiment:          normalizeSentiment(raw.Sentiment),
		TopAsks:            raw.TopAsks,
		RecommendedActions: raw.RecommendedActions,
		CommentsAnalyzed:   len(comments),
		OfficialResponders: officialResponders(comments),
		Model:              s.ai.Model(),
		GeneratedAt:        time.Now().UTC(),
	}
	s.cache.Set(ctx, in.Resource, in.ID, out)
	return out, nil
}

// MinComments is exported so the handler can 200-with-an-explanation on
// under-threshold threads instead of paying for a low-signal Gemini call.
func MinComments() int { return minComments }

// visibleComments strips moderator-hidden rows (community-service already
// replaces their content with a placeholder, but there's no point spending
// tokens on "[Removed by moderator]"). Newest-first so we keep the most
// recent maxComments if the thread is oversized.
func visibleComments(all []Comment) []Comment {
	out := make([]Comment, 0, len(all))
	for _, c := range all {
		if c.IsHidden || strings.TrimSpace(c.Content) == "" {
			continue
		}
		out = append(out, c)
	}
	if len(out) > maxComments {
		out = out[len(out)-maxComments:]
	}
	return out
}

func officialResponders(comments []Comment) []string {
	seen := make(map[string]struct{})
	var names []string
	for _, c := range comments {
		if !c.IsOfficialResponse {
			continue
		}
		if _, dup := seen[c.AuthorName]; dup {
			continue
		}
		seen[c.AuthorName] = struct{}{}
		names = append(names, c.AuthorName)
	}
	return names
}

// buildPrompt assembles the resource header + the comment stream in a
// stable, parseable format. Gemini reads this best when structure is
// obvious and formatting is consistent.
func buildPrompt(res *Resource, comments []Comment) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Resource: %s\n", strings.ToUpper(res.Kind))
	fmt.Fprintf(&sb, "Title: %s\n", res.Title)
	if res.Status != "" {
		fmt.Fprintf(&sb, "Status: %s\n", res.Status)
	}
	fmt.Fprintf(&sb, "\nOriginal post:\n%s\n", res.Description)
	fmt.Fprintf(&sb, "\nDiscussion (%d comments, oldest first):\n", len(comments))
	for i, c := range comments {
		role := c.AuthorRole
		if c.IsOfficialResponse {
			role += " · OFFICIAL"
		}
		fmt.Fprintf(&sb, "[%d] %s (%s): %s\n", i+1, c.AuthorName, role, strings.TrimSpace(c.Content))
	}
	return sb.String()
}

func normalizeSentiment(s Sentiment) Sentiment {
	// Clamp negatives, then rescale to sum ~1.0. Gemini usually returns
	// clean values but the guardrail is cheap.
	if s.Positive < 0 {
		s.Positive = 0
	}
	if s.Neutral < 0 {
		s.Neutral = 0
	}
	if s.Negative < 0 {
		s.Negative = 0
	}
	total := s.Positive + s.Neutral + s.Negative
	if total == 0 {
		return Sentiment{Neutral: 1}
	}
	return Sentiment{
		Positive: s.Positive / total,
		Neutral:  s.Neutral / total,
		Negative: s.Negative / total,
	}
}
