package summarize

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// SourceClient pulls petition / issue / consultation detail + discussion
// text from the services that own each resource. It forwards the caller's
// Bearer token so authorization naturally cascades — a user who can't read
// the resource can't summarize it.
type SourceClient struct {
	communityURL    string
	organizationURL string
	http            *http.Client
}

// NewSourceClient wires the community + organization service base URLs.
// The organization URL is only consulted for the consultation kind; the
// petition / issue kinds still hit community-service.
func NewSourceClient(communityURL, organizationURL string) *SourceClient {
	return &SourceClient{
		communityURL:    communityURL,
		organizationURL: organizationURL,
		http:            &http.Client{Timeout: 10 * time.Second},
	}
}

// Comment is the minimal shape both petition and issue comments share on
// the community-service surface. Field names match its JSON output so the
// same decoder works for both endpoints.
type Comment struct {
	ID                 string    `json:"id"`
	Content            string    `json:"content"`
	AuthorName         string    `json:"authorName"`
	AuthorRole         string    `json:"authorRole"`
	IsOfficialResponse bool      `json:"isOfficialResponse"`
	IsHidden           bool      `json:"isHidden"`
	CreatedAt          time.Time `json:"createdAt"`
}

// Resource is the summarizable object plus the enough of its detail record
// to feed Gemini a title + description alongside the comment stream.
type Resource struct {
	Kind        string
	ID          string
	Title       string
	Description string
	Status      string
	Comments    []Comment
}

// Fetch loads the target resource + its discussion text. Returns a wrapped
// error whose Status maps to the HTTP code the handler should return
// (404, 403, 502). Consultations follow a different fan-out than petitions
// and issues, so branch here rather than uniformly.
func (s *SourceClient) Fetch(ctx context.Context, kind, id, bearerToken string) (*Resource, error) {
	switch kind {
	case "petition", "issue":
		return s.fetchCommunityResource(ctx, kind, id, bearerToken)
	case "consultation":
		return s.fetchConsultation(ctx, id, bearerToken)
	default:
		return nil, &SourceError{Status: http.StatusBadRequest, Message: "unsupported resource kind"}
	}
}

func (s *SourceClient) fetchCommunityResource(ctx context.Context, kind, id, bearerToken string) (*Resource, error) {
	var (
		detailPath   = fmt.Sprintf("/v1/%ss/%s", kind, id)
		commentsPath = fmt.Sprintf("/v1/%ss/%s/comments", kind, id)
	)

	var detail struct {
		Data struct {
			Petition *struct {
				Title       string `json:"title"`
				Description string `json:"description"`
				Status      string `json:"status"`
			} `json:"petition"`
			Issue *struct {
				Title       string `json:"title"`
				Description string `json:"description"`
				Status      string `json:"status"`
			} `json:"issue"`
		} `json:"data"`
	}
	if err := s.getJSON(ctx, s.communityURL+detailPath, bearerToken, &detail); err != nil {
		return nil, err
	}

	out := &Resource{Kind: kind, ID: id}
	switch {
	case detail.Data.Petition != nil:
		out.Title = detail.Data.Petition.Title
		out.Description = detail.Data.Petition.Description
		out.Status = detail.Data.Petition.Status
	case detail.Data.Issue != nil:
		out.Title = detail.Data.Issue.Title
		out.Description = detail.Data.Issue.Description
		out.Status = detail.Data.Issue.Status
	default:
		return nil, &SourceError{Status: http.StatusNotFound, Message: "resource not found"}
	}

	var commentsResp struct {
		Data struct {
			Comments []Comment `json:"comments"`
		} `json:"data"`
	}
	if err := s.getJSON(ctx, s.communityURL+commentsPath, bearerToken, &commentsResp); err != nil {
		return nil, err
	}
	out.Comments = commentsResp.Data.Comments
	return out, nil
}

// fetchConsultation loads consultation detail + questions + responses.
// Responses are structured (per-question answers), so we flatten each
// respondent's text into a synthetic "comment" — the existing Gemini
// prompt shape then works unchanged.
//
// The response list read is admin-only in organization-service, so this
// path relies on the caller's JWT carrying a staff role. The handler role
// gate should already have enforced that, but a 403 from upstream still
// propagates cleanly.
func (s *SourceClient) fetchConsultation(ctx context.Context, id, bearerToken string) (*Resource, error) {
	var detail struct {
		Data struct {
			Consultation *struct {
				Title       string `json:"title"`
				Summary     string `json:"summary"`
				Description string `json:"description"`
				Status      string `json:"status"`
			} `json:"consultation"`
		} `json:"data"`
	}
	if err := s.getJSON(ctx, s.organizationURL+"/v1/consultations/"+id, bearerToken, &detail); err != nil {
		return nil, err
	}
	if detail.Data.Consultation == nil {
		return nil, &SourceError{Status: http.StatusNotFound, Message: "consultation not found"}
	}

	// Questions provide the prompt text so we can label each answer.
	// Fetch is best-effort — a failure just means we render answers
	// without their question text.
	type qRow struct {
		ID     string `json:"id"`
		Prompt string `json:"prompt"`
	}
	var questionsResp struct {
		Data struct {
			Questions []qRow `json:"questions"`
		} `json:"data"`
	}
	promptByQID := make(map[string]string)
	if err := s.getJSON(ctx, s.organizationURL+"/v1/consultations/"+id+"/questions", bearerToken, &questionsResp); err == nil {
		for _, q := range questionsResp.Data.Questions {
			promptByQID[q.ID] = q.Prompt
		}
	}

	// Responses: staff-only endpoint. Answers are inline.
	type answer struct {
		QuestionID string   `json:"questionId"`
		TextValue  *string  `json:"textValue,omitempty"`
		Selections []string `json:"selections,omitempty"`
	}
	type responseRow struct {
		ID          string    `json:"id"`
		SubmittedAt time.Time `json:"submittedAt"`
		Answers     []answer  `json:"answers"`
	}
	var responsesResp struct {
		Data struct {
			Responses []responseRow `json:"responses"`
		} `json:"data"`
	}
	if err := s.getJSON(ctx, s.organizationURL+"/v1/consultations/"+id+"/responses", bearerToken, &responsesResp); err != nil {
		return nil, err
	}

	// Flatten each response into a synthetic Comment. Multi-answer
	// responses become one paragraph per answer, prefixed by the question
	// text. Author is generic — consultations don't publish respondent
	// identity, and Gemini gets more mileage from the answer content than
	// from anonymous author labels.
	comments := make([]Comment, 0, len(responsesResp.Data.Responses))
	for i, r := range responsesResp.Data.Responses {
		var sb strings.Builder
		for _, a := range r.Answers {
			label := promptByQID[a.QuestionID]
			if label == "" {
				label = "Question"
			}
			switch {
			case a.TextValue != nil && strings.TrimSpace(*a.TextValue) != "":
				fmt.Fprintf(&sb, "%s → %s\n", label, strings.TrimSpace(*a.TextValue))
			case len(a.Selections) > 0:
				fmt.Fprintf(&sb, "%s → %s\n", label, strings.Join(a.Selections, ", "))
			}
		}
		content := strings.TrimSpace(sb.String())
		if content == "" {
			continue
		}
		comments = append(comments, Comment{
			ID:         r.ID,
			Content:    content,
			AuthorName: fmt.Sprintf("Respondent #%d", i+1),
			AuthorRole: "CITIZEN",
			CreatedAt:  r.SubmittedAt,
		})
	}

	// Blend the consultation summary with description so Gemini has full
	// framing before it reads the responses.
	description := detail.Data.Consultation.Description
	if trimmed := strings.TrimSpace(detail.Data.Consultation.Summary); trimmed != "" {
		description = trimmed + "\n\n" + description
	}

	return &Resource{
		Kind:        "consultation",
		ID:          id,
		Title:       detail.Data.Consultation.Title,
		Description: description,
		Status:      detail.Data.Consultation.Status,
		Comments:    comments,
	}, nil
}

func (s *SourceClient) getJSON(ctx context.Context, fullURL, bearer string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return &SourceError{Status: http.StatusInternalServerError, Message: err.Error()}
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return &SourceError{Status: http.StatusBadGateway, Message: "upstream unavailable: " + err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return &SourceError{
			Status:  resp.StatusCode,
			Message: fmt.Sprintf("upstream %d: %s", resp.StatusCode, truncate(string(body), 200)),
		}
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

type SourceError struct {
	Status  int
	Message string
}

func (e *SourceError) Error() string { return e.Message }

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
