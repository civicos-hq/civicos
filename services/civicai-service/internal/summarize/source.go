package summarize

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// SourceClient pulls petition / issue detail + comments from
// community-service. It forwards the caller's Bearer token so authorization
// naturally cascades — a user who can't read the resource can't summarize it.
type SourceClient struct {
	baseURL string
	http    *http.Client
}

func NewSourceClient(baseURL string) *SourceClient {
	return &SourceClient{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 10 * time.Second},
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

// Fetch loads the target resource + its comments in two calls to
// community-service. Returns a wrapped error whose Status maps to the
// HTTP code the handler should return (404, 403, 502).
func (s *SourceClient) Fetch(ctx context.Context, kind, id, bearerToken string) (*Resource, error) {
	var (
		detailPath   string
		commentsPath string
	)
	switch kind {
	case "petition":
		detailPath = fmt.Sprintf("/v1/petitions/%s", id)
		commentsPath = fmt.Sprintf("/v1/petitions/%s/comments", id)
	case "issue":
		detailPath = fmt.Sprintf("/v1/issues/%s", id)
		commentsPath = fmt.Sprintf("/v1/issues/%s/comments", id)
	default:
		return nil, &SourceError{Status: http.StatusBadRequest, Message: "unsupported resource kind"}
	}

	// Detail
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
	if err := s.getJSON(ctx, detailPath, bearerToken, &detail); err != nil {
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

	// Comments
	var commentsResp struct {
		Data struct {
			Comments []Comment `json:"comments"`
		} `json:"data"`
	}
	if err := s.getJSON(ctx, commentsPath, bearerToken, &commentsResp); err != nil {
		return nil, err
	}
	out.Comments = commentsResp.Data.Comments
	return out, nil
}

func (s *SourceClient) getJSON(ctx context.Context, path, bearer string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+path, nil)
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
