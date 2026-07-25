// Package narrate turns raw platform metrics into a plain-language
// digest for admin overview surfaces. Gemini reads the numbers; the FE
// renders a human-friendly narrative alongside the existing dashboard.
package narrate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/civicos/civicai-service/internal/gemini"
	"github.com/redis/go-redis/v9"
	"google.golang.org/genai"
)

const cacheTTL = 15 * time.Minute

type Service struct {
	ai          *gemini.Client
	identityURL string
	rdb         *redis.Client
	http        *http.Client
}

func NewService(ai *gemini.Client, identityURL string, rdb *redis.Client) *Service {
	return &Service{
		ai:          ai,
		identityURL: identityURL,
		rdb:         rdb,
		http:        &http.Client{Timeout: 8 * time.Second},
	}
}

// PlatformMetrics mirrors identity-service adminmetrics.Metrics. We
// re-declare rather than import so civicai-service can stay independent
// of identity-service's Go module. If the shape changes upstream we'll
// hear about it via the JSON decode.
type PlatformMetrics struct {
	Users struct {
		Total        int64 `json:"total"`
		NewToday     int64 `json:"newToday"`
		NewThisWeek  int64 `json:"newThisWeek"`
		VerifiedRate int   `json:"verifiedRate"`
		BannedTotal  int64 `json:"bannedTotal"`
	} `json:"users"`
	Communities struct {
		Total int64 `json:"total"`
	} `json:"communities"`
	Issues struct {
		Total        int64            `json:"total"`
		ByStatus     map[string]int64 `json:"byStatus"`
		ResponseRate int              `json:"responseRate"`
	} `json:"issues"`
	Petitions struct {
		Total              int64 `json:"total"`
		SignaturesTotal    int64 `json:"signaturesTotal"`
		SignaturesThisWeek int64 `json:"signaturesThisWeek"`
	} `json:"petitions"`
	Representatives struct {
		Total int64 `json:"total"`
	} `json:"representatives"`
	Organizations struct {
		Total    int64 `json:"total"`
		Verified int64 `json:"verified"`
	} `json:"organizations"`
	Moderation struct {
		PendingFlags    int64 `json:"pendingFlags"`
		HiddenAllTime   int64 `json:"hiddenAllTime"`
		AuditLogEntries int64 `json:"auditLogEntries"`
	} `json:"moderation"`
}

type NarrationOutput struct {
	Scope           string    `json:"scope"`
	Headline        string    `json:"headline"`
	Narrative       string    `json:"narrative"`
	Highlights      []string  `json:"highlights"`
	Trends          []string  `json:"trends"`
	Recommendations []string  `json:"recommendations"`
	Metrics         any       `json:"metrics"` // echoed back so the FE has one payload
	Model           string    `json:"model"`
	GeneratedAt     time.Time `json:"generatedAt"`
	Cached          bool      `json:"cached"`
}

type SourceError struct {
	Status  int
	Message string
}

func (e *SourceError) Error() string { return e.Message }

const systemInstruction = `You are CivicAI, the intelligence layer of the CivicOS civic engagement platform.
An organization admin or platform operator is looking at a dashboard of numbers and wants a plain-language read on what those numbers mean — not a rehash of the raw data.

Ground rules:
- Base every claim on the metrics you were given. Never invent numbers, percentages, or comparisons.
- Be neutral and operator-focused. Prefer "the platform" over "we" or "you".
- Prefer specific numbers over vague adjectives — "83 new users this week" beats "user growth is strong".
- If a metric is zero or missing, do not manufacture a story about it.

Produce:
- headline: one sentence (max 15 words) that captures the top story in these numbers.
- narrative: 2 to 3 short paragraphs (max 120 words total) that a busy admin can read in twenty seconds. Weave the numbers into sentences rather than listing them.
- highlights: 3 to 5 short bullet-ready one-liners — the standout numbers worth surfacing.
- trends: 2 to 4 sentences noting where things appear to be improving, plateauing, or slipping. Only mention trends the data supports (e.g. "verification rate at X%", "N flags still pending").
- recommendations: 2 to 4 imperative-mood suggestions for the operator based on the numbers. Each is one sentence.`

func responseSchema() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"headline":        {Type: genai.TypeString},
			"narrative":       {Type: genai.TypeString},
			"highlights":      {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
			"trends":          {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
			"recommendations": {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
		},
		Required: []string{"headline", "narrative", "highlights", "trends", "recommendations"},
	}
}

// NarratePlatform pulls /v1/admin/metrics from identity-service with the
// caller's JWT (their PLATFORM_ADMIN role gates the read), then asks
// Gemini to explain the numbers. Cached 15 min per bearer token — the
// bearer token uniquely identifies the calling admin, but the metrics
// don't vary per admin, so we key by scope alone.
func (s *Service) NarratePlatform(ctx context.Context, bearerToken string) (*NarrationOutput, error) {
	if cached := s.getCache(ctx, "platform"); cached != nil {
		cached.Cached = true
		return cached, nil
	}

	metrics, err := s.fetchPlatformMetrics(ctx, bearerToken)
	if err != nil {
		return nil, err
	}

	prompt := buildPrompt("platform", metrics)

	type geminiOut struct {
		Headline        string   `json:"headline"`
		Narrative       string   `json:"narrative"`
		Highlights      []string `json:"highlights"`
		Trends          []string `json:"trends"`
		Recommendations []string `json:"recommendations"`
	}
	var raw geminiOut
	if err := s.ai.GenerateJSON(ctx, systemInstruction, prompt, responseSchema(), &raw); err != nil {
		return nil, err
	}

	out := &NarrationOutput{
		Scope:           "platform",
		Headline:        strings.TrimSpace(raw.Headline),
		Narrative:       strings.TrimSpace(raw.Narrative),
		Highlights:      raw.Highlights,
		Trends:          raw.Trends,
		Recommendations: raw.Recommendations,
		Metrics:         metrics,
		Model:           s.ai.Model(),
		GeneratedAt:     time.Now().UTC(),
	}
	s.setCache(ctx, "platform", out)
	return out, nil
}

func (s *Service) fetchPlatformMetrics(ctx context.Context, bearerToken string) (*PlatformMetrics, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.identityURL+"/v1/admin/metrics", nil)
	if err != nil {
		return nil, &SourceError{Status: http.StatusInternalServerError, Message: err.Error()}
	}
	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, &SourceError{Status: http.StatusBadGateway, Message: "identity-service unreachable: " + err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		msg := string(body)
		if len(msg) > 200 {
			msg = msg[:200] + "…"
		}
		return nil, &SourceError{Status: resp.StatusCode, Message: fmt.Sprintf("upstream %d: %s", resp.StatusCode, msg)}
	}
	var wrapper struct {
		Data struct {
			Metrics *PlatformMetrics `json:"metrics"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, &SourceError{Status: http.StatusBadGateway, Message: "decode: " + err.Error()}
	}
	if wrapper.Data.Metrics == nil {
		return nil, &SourceError{Status: http.StatusBadGateway, Message: "empty metrics payload"}
	}
	return wrapper.Data.Metrics, nil
}

func buildPrompt(scope string, m *PlatformMetrics) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Scope: %s\n\n", scope)
	fmt.Fprintf(&sb, "USERS\n  total=%d · newToday=%d · newThisWeek=%d · verifiedRate=%d%% · banned=%d\n\n",
		m.Users.Total, m.Users.NewToday, m.Users.NewThisWeek, m.Users.VerifiedRate, m.Users.BannedTotal)
	fmt.Fprintf(&sb, "COMMUNITIES\n  total=%d\n\n", m.Communities.Total)
	fmt.Fprintf(&sb, "ISSUES\n  total=%d · responseRate=%d%%\n", m.Issues.Total, m.Issues.ResponseRate)
	if len(m.Issues.ByStatus) > 0 {
		sb.WriteString("  byStatus:")
		for status, count := range m.Issues.ByStatus {
			fmt.Fprintf(&sb, " %s=%d", status, count)
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	fmt.Fprintf(&sb, "PETITIONS\n  total=%d · signaturesTotal=%d · signaturesThisWeek=%d\n\n",
		m.Petitions.Total, m.Petitions.SignaturesTotal, m.Petitions.SignaturesThisWeek)
	fmt.Fprintf(&sb, "REPRESENTATIVES\n  total=%d\n\n", m.Representatives.Total)
	fmt.Fprintf(&sb, "ORGANIZATIONS\n  total=%d · verified=%d\n\n", m.Organizations.Total, m.Organizations.Verified)
	fmt.Fprintf(&sb, "MODERATION\n  pendingFlags=%d · hiddenAllTime=%d · auditLogEntries=%d\n",
		m.Moderation.PendingFlags, m.Moderation.HiddenAllTime, m.Moderation.AuditLogEntries)
	return sb.String()
}

func (s *Service) cacheKey(scope string) string { return "civicai:narrate:" + scope }

func (s *Service) getCache(ctx context.Context, scope string) *NarrationOutput {
	if s.rdb == nil {
		return nil
	}
	raw, err := s.rdb.Get(ctx, s.cacheKey(scope)).Bytes()
	if err != nil {
		return nil
	}
	var out NarrationOutput
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return &out
}

func (s *Service) setCache(ctx context.Context, scope string, out *NarrationOutput) {
	if s.rdb == nil || out == nil {
		return
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return
	}
	_ = s.rdb.Set(ctx, s.cacheKey(scope), raw, cacheTTL).Err()
}
