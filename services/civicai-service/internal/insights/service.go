package insights

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/civicos/civicai-service/internal/gemini"
	"github.com/redis/go-redis/v9"
	"google.golang.org/genai"
)

const (
	// Corpus caps. A single Gemini call runs happily against tens of
	// thousands of tokens, but at some point marginal cost outweighs
	// marginal insight. These numbers keep the average community under
	// ~8k tokens of context.
	maxIssues            = 25
	maxPetitions         = 15
	maxCommentsPerThread = 12
	// Cache TTL is longer than summarize's — community insights change
	// slowly (dominant themes don't flip in an hour) and the fan-out
	// makes fresh calls expensive.
	cacheTTL = 1 * time.Hour
)

type Service struct {
	ai     *gemini.Client
	source *SourceClient
	rdb    *redis.Client
}

func NewService(ai *gemini.Client, source *SourceClient, rdb *redis.Client) *Service {
	return &Service{ai: ai, source: source, rdb: rdb}
}

type InsightsInput struct {
	CommunityID string `form:"communityId" binding:"required"`
}

type SentimentMix struct {
	Positive float64 `json:"positive"`
	Neutral  float64 `json:"neutral"`
	Negative float64 `json:"negative"`
}

type Activity struct {
	IssueCount    int `json:"issueCount"`
	PetitionCount int `json:"petitionCount"`
	CommentCount  int `json:"commentCount"`
}

type InsightsOutput struct {
	CommunityID        string       `json:"communityId"`
	TLDR               string       `json:"tldr"`
	Themes             []string     `json:"themes"`
	SentimentMix       SentimentMix `json:"sentimentMix"`
	TopAsks            []string     `json:"topAsks"`
	RecommendedActions []string     `json:"recommendedActions"`
	Activity           Activity     `json:"activity"`
	Model              string       `json:"model"`
	GeneratedAt        time.Time    `json:"generatedAt"`
	Cached             bool         `json:"cached"`
}

const systemInstruction = `You are CivicAI, the intelligence layer of the CivicOS civic engagement platform.
An organization admin or elected representative wants a decision-support digest of what's happening across a whole community — not a single thread. You will receive a bundle of recent issue reports, petitions, and comments from citizens.

Ground rules:
- Base every claim on the corpus you were given. Never invent facts, names, dates, statistics, or promises.
- Be neutral. Do not take sides between citizens and the organization.
- Aggregate — pick the recurring stories across items, not a play-by-play of each one.
- Prefer specific, actionable language over vague summary-speak.

Produce:
- tldr: 2 to 3 sentences (max 90 words) framing the top story in this community right now.
- themes: 4 to 8 short noun phrases — the recurring topics ("waste collection reliability", "streetlight outages", "market drainage").
- sentimentMix: three numbers between 0 and 1 summing to ~1 — the overall balance across all comments.
- topAsks: 4 to 6 concrete things citizens are asking for across the corpus, in citizens' own voice where possible.
- recommendedActions: 3 to 5 actions the organization could take next, ordered by impact. Each is one imperative-mood sentence.`

func responseSchema() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"tldr":   {Type: genai.TypeString},
			"themes": {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
			"sentimentMix": {
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
		Required: []string{"tldr", "themes", "sentimentMix", "topAsks", "recommendedActions"},
	}
}

func (s *Service) Generate(ctx context.Context, communityID, bearerToken string) (*InsightsOutput, error) {
	if cached := s.getCache(ctx, communityID); cached != nil {
		cached.Cached = true
		return cached, nil
	}

	corpus, err := s.source.FetchCorpus(ctx, communityID, bearerToken, maxIssues, maxPetitions)
	if err != nil {
		return nil, err
	}

	prompt := buildPrompt(corpus)

	type geminiOut struct {
		TLDR               string       `json:"tldr"`
		Themes             []string     `json:"themes"`
		SentimentMix       SentimentMix `json:"sentimentMix"`
		TopAsks            []string     `json:"topAsks"`
		RecommendedActions []string     `json:"recommendedActions"`
	}
	var raw geminiOut
	if err := s.ai.GenerateJSON(ctx, systemInstruction, prompt, responseSchema(), &raw); err != nil {
		return nil, err
	}

	out := &InsightsOutput{
		CommunityID:        communityID,
		TLDR:               strings.TrimSpace(raw.TLDR),
		Themes:             raw.Themes,
		SentimentMix:       normalizeMix(raw.SentimentMix),
		TopAsks:            raw.TopAsks,
		RecommendedActions: raw.RecommendedActions,
		Activity: Activity{
			IssueCount:    len(corpus.Issues),
			PetitionCount: len(corpus.Petitions),
			CommentCount:  corpus.CommentCount,
		},
		Model:       s.ai.Model(),
		GeneratedAt: time.Now().UTC(),
	}
	s.setCache(ctx, communityID, out)
	return out, nil
}

func buildPrompt(c *Corpus) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Community activity corpus: %d issues, %d petitions, %d comments total (capped for prompt size).\n\n",
		len(c.Issues), len(c.Petitions), c.CommentCount)

	if len(c.Issues) > 0 {
		sb.WriteString("=== ISSUES ===\n")
		for i, iss := range c.Issues {
			fmt.Fprintf(&sb, "\n[I%d] %s (category=%s, status=%s)\n", i+1, iss.Title, iss.Category, iss.Status)
			sb.WriteString(iss.Description)
			sb.WriteString("\n")
			appendComments(&sb, c.CommentsByID[iss.ID], maxCommentsPerThread)
		}
	}

	if len(c.Petitions) > 0 {
		sb.WriteString("\n=== PETITIONS ===\n")
		for i, p := range c.Petitions {
			fmt.Fprintf(&sb, "\n[P%d] %s (status=%s)\n", i+1, p.Title, p.Status)
			sb.WriteString(p.Description)
			sb.WriteString("\n")
			appendComments(&sb, c.CommentsByID[p.ID], maxCommentsPerThread)
		}
	}

	return sb.String()
}

func appendComments(sb *strings.Builder, comments []Comment, capN int) {
	shown := 0
	for _, c := range comments {
		if c.IsHidden || strings.TrimSpace(c.Content) == "" {
			continue
		}
		if shown >= capN {
			fmt.Fprintf(sb, "  … (%d more comments truncated)\n", len(comments)-shown)
			return
		}
		role := c.AuthorRole
		if c.IsOfficialResponse {
			role += " · OFFICIAL"
		}
		fmt.Fprintf(sb, "  • %s (%s): %s\n", c.AuthorName, role, strings.TrimSpace(c.Content))
		shown++
	}
}

func normalizeMix(m SentimentMix) SentimentMix {
	if m.Positive < 0 {
		m.Positive = 0
	}
	if m.Neutral < 0 {
		m.Neutral = 0
	}
	if m.Negative < 0 {
		m.Negative = 0
	}
	total := m.Positive + m.Neutral + m.Negative
	if total == 0 {
		return SentimentMix{Neutral: 1}
	}
	return SentimentMix{
		Positive: m.Positive / total,
		Neutral:  m.Neutral / total,
		Negative: m.Negative / total,
	}
}

func (s *Service) cacheKey(communityID string) string {
	return "civicai:insights:" + communityID
}

func (s *Service) getCache(ctx context.Context, communityID string) *InsightsOutput {
	if s.rdb == nil {
		return nil
	}
	raw, err := s.rdb.Get(ctx, s.cacheKey(communityID)).Bytes()
	if err != nil {
		return nil
	}
	var out InsightsOutput
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return &out
}

func (s *Service) setCache(ctx context.Context, communityID string, out *InsightsOutput) {
	if s.rdb == nil || out == nil {
		return
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return
	}
	_ = s.rdb.Set(ctx, s.cacheKey(communityID), raw, cacheTTL).Err()
}
