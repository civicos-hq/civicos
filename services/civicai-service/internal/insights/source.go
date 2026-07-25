// Package insights aggregates community-level activity (issues + petitions
// + their comments) and asks Gemini to produce a decision-support digest
// for organization staff. Unlike summarize (single thread) this endpoint
// fans out reads across community-service and bounds the total corpus
// before sending to Gemini.
package insights

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type SourceClient struct {
	baseURL string
	http    *http.Client
}

func NewSourceClient(baseURL string) *SourceClient {
	return &SourceClient{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 8 * time.Second},
	}
}

type IssueSummary struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
}

type PetitionSummary struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
}

type Comment struct {
	Content            string `json:"content"`
	AuthorName         string `json:"authorName"`
	AuthorRole         string `json:"authorRole"`
	IsOfficialResponse bool   `json:"isOfficialResponse"`
	IsHidden           bool   `json:"isHidden"`
}

type Corpus struct {
	Issues       []IssueSummary
	Petitions    []PetitionSummary
	CommentsByID map[string][]Comment
	CommentCount int
}

type SourceError struct {
	Status  int
	Message string
}

func (e *SourceError) Error() string { return e.Message }

// FetchCorpus loads the community's recent issues + petitions in parallel,
// then fetches comments for each in a bounded worker pool. Returns a
// *SourceError with an HTTP-mappable status when the upstream is unhealthy.
func (s *SourceClient) FetchCorpus(ctx context.Context, communityID, bearerToken string, maxIssues, maxPetitions int) (*Corpus, error) {
	var (
		issues    []IssueSummary
		petitions []PetitionSummary
		errIssues error
		errPets   error
	)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		errIssues = s.getJSON(ctx, fmt.Sprintf("/v1/issues?communityId=%s", communityID), bearerToken, &struct {
			Data struct {
				Issues *[]IssueSummary `json:"issues"`
			} `json:"data"`
		}{Data: struct {
			Issues *[]IssueSummary `json:"issues"`
		}{Issues: &issues}})
	}()
	go func() {
		defer wg.Done()
		errPets = s.getJSON(ctx, fmt.Sprintf("/v1/petitions?communityId=%s", communityID), bearerToken, &struct {
			Data struct {
				Petitions *[]PetitionSummary `json:"petitions"`
			} `json:"data"`
		}{Data: struct {
			Petitions *[]PetitionSummary `json:"petitions"`
		}{Petitions: &petitions}})
	}()
	wg.Wait()

	if errIssues != nil {
		return nil, errIssues
	}
	if errPets != nil {
		return nil, errPets
	}

	if len(issues) > maxIssues {
		issues = issues[:maxIssues]
	}
	if len(petitions) > maxPetitions {
		petitions = petitions[:maxPetitions]
	}

	// Bounded parallel comment fetches. 4 workers is enough for a demo
	// community (max ~15 targets); big-tenant scaling would want a
	// smarter budget + a sampling strategy.
	type target struct {
		kind string
		id   string
	}
	targets := make([]target, 0, len(issues)+len(petitions))
	for _, i := range issues {
		targets = append(targets, target{"issues", i.ID})
	}
	for _, p := range petitions {
		targets = append(targets, target{"petitions", p.ID})
	}

	commentsByID := make(map[string][]Comment, len(targets))
	var mu sync.Mutex
	commentTotal := 0

	work := make(chan target)
	var workers sync.WaitGroup
	for i := 0; i < 4; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for tgt := range work {
				var wrapper struct {
					Data struct {
						Comments []Comment `json:"comments"`
					} `json:"data"`
				}
				path := fmt.Sprintf("/v1/%s/%s/comments", tgt.kind, tgt.id)
				if err := s.getJSON(ctx, path, bearerToken, &wrapper); err != nil {
					// Best effort — a failed comment fetch shouldn't sink
					// the whole insights call.
					continue
				}
				mu.Lock()
				commentsByID[tgt.id] = wrapper.Data.Comments
				commentTotal += len(wrapper.Data.Comments)
				mu.Unlock()
			}
		}()
	}
	for _, t := range targets {
		work <- t
	}
	close(work)
	workers.Wait()

	return &Corpus{
		Issues:       issues,
		Petitions:    petitions,
		CommentsByID: commentsByID,
		CommentCount: commentTotal,
	}, nil
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
		msg := string(body)
		if len(msg) > 200 {
			msg = msg[:200] + "…"
		}
		return &SourceError{Status: resp.StatusCode, Message: fmt.Sprintf("upstream %d: %s", resp.StatusCode, msg)}
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
