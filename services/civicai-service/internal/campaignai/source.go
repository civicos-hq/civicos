// Package campaignai holds the CivicAI surfaces for Community Funding.
//
// Six task-shaped endpoints, all advisory. None of them writes anything, and
// none is on the path of a donation: per the funding plan, "AI never gates
// money" and "fail closed on money, open on everything else". If Gemini is
// unreachable, every one of these degrades to the organization or admin doing
// the work by hand, which is exactly what they do today.
//
// The one that needs watching is assess-campaign-risk. It produces a fraud
// signal about a real organization asking for money for, often, a flood or a
// clinic. It is admin-only, it never auto-acts, and the code enforces both —
// see risk.go and handler.go.
package campaignai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// SourceClient reads campaign context from organization-service.
//
// It forwards the caller's Bearer token, so authorization cascades: the
// campaign read is gated by CanReadInternal on the owning org, meaning an
// org admin can only ask CivicAI about their own campaigns and a platform
// admin can ask about any. There is deliberately no service-to-service
// credential here — adding one would silently widen who can have a campaign
// assessed.
type SourceClient struct {
	organizationURL string
	http            *http.Client
}

func NewSourceClient(organizationURL string) *SourceClient {
	return &SourceClient{
		organizationURL: organizationURL,
		// Four fan-out reads behind a 15s Gemini call; keep each one short so
		// a slow upstream cannot hold the request open past the caller's
		// patience.
		http: &http.Client{Timeout: 8 * time.Second},
	}
}

// Milestone is the org's own plan for the money, set before it could publish.
type Milestone struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	TargetMinor int64  `json:"targetMinor"`
	Status      string `json:"status"`
}

// SpendRecord is a CLAIM by the organization, not an observed fact. CivicOS
// never held the money and cannot verify any of these. Every prompt that
// consumes them says so, because a model told "here is what was spent" will
// happily write a summary asserting it as truth.
type SpendRecord struct {
	Description string    `json:"description"`
	AmountMinor int64     `json:"amountMinor"`
	SpentAt     time.Time `json:"spentAt"`
}

// Update is a progress post the organization published to its funding feed.
type Update struct {
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
}

// Campaign is the subset of the campaign record these prompts need.
//
// Deliberately excludes the review trail (reviewNote, reviewedById) and any
// donor identity. A model does not need to know who objected to a campaign in
// order to summarise its impact, and anything put in a prompt can come back
// out in the completion.
type Campaign struct {
	ID             string     `json:"id"`
	Title          string     `json:"title"`
	Summary        string     `json:"summary"`
	Description    string     `json:"description"`
	Category       string     `json:"category"`
	Status         string     `json:"status"`
	Currency       string     `json:"currency"`
	GoalMinor      int64      `json:"goalMinor"`
	RaisedMinor    int64      `json:"raisedMinor"`
	DonorCount     int        `json:"donorCount"`
	IsEmergency    bool       `json:"isEmergency"`
	State          *string    `json:"state"`
	LGA            *string    `json:"lga"`
	OrganizationID string     `json:"organizationId"`
	StartDate      *time.Time `json:"startDate"`
	EndDate        *time.Time `json:"endDate"`
	PublishedAt    *time.Time `json:"publishedAt"`
	CreatedAt      time.Time  `json:"createdAt"`
}

// Context is everything the campaign prompts are allowed to see.
type Context struct {
	Campaign   Campaign
	Milestones []Milestone
	Spend      []SpendRecord
	Updates    []Update
}

// ReportedMinor totals what the organization SAYS it has spent.
func (c Context) ReportedMinor() int64 {
	var t int64
	for _, s := range c.Spend {
		t += s.AmountMinor
	}
	return t
}

// UnaccountedMinor is money received through CivicOS that the organization
// has not yet accounted for. May be negative when an org reports spending
// more than it raised here — which is normal (they have other funding) and
// must not be clamped to zero, or a prompt would be handed a tidier picture
// than the truth.
func (c Context) UnaccountedMinor() int64 {
	return c.Campaign.RaisedMinor - c.ReportedMinor()
}

// SourceError carries the HTTP status the handler should surface, so a
// caller asking about a campaign they cannot read gets 403/404 rather than a
// generic AI failure.
type SourceError struct {
	Status  int
	Code    string
	Message string
}

func (e *SourceError) Error() string { return e.Message }

// Fetch loads the campaign and everything published about it.
//
// The campaign read is required; milestones, spend and updates are best
// effort. A campaign with no spend records yet is not an error — it is a
// fact about the campaign, and often the most interesting one.
func (s *SourceClient) Fetch(ctx context.Context, campaignID, bearer string) (*Context, error) {
	var campaignEnv struct {
		Data struct {
			Campaign Campaign `json:"campaign"`
		} `json:"data"`
	}
	if err := s.get(ctx, "/v1/campaigns/"+campaignID, bearer, &campaignEnv); err != nil {
		return nil, err
	}
	if campaignEnv.Data.Campaign.ID == "" {
		return nil, &SourceError{Status: http.StatusNotFound, Code: "CAMPAIGN_NOT_FOUND", Message: "Campaign not found"}
	}
	out := &Context{Campaign: campaignEnv.Data.Campaign}

	var msEnv struct {
		Data struct {
			Milestones []Milestone `json:"milestones"`
		} `json:"data"`
	}
	if err := s.get(ctx, "/v1/campaigns/"+campaignID+"/milestones", bearer, &msEnv); err == nil {
		out.Milestones = msEnv.Data.Milestones
	}

	var spendEnv struct {
		Data struct {
			Spend []SpendRecord `json:"spend"`
		} `json:"data"`
	}
	if err := s.get(ctx, "/v1/campaigns/"+campaignID+"/spend", bearer, &spendEnv); err == nil {
		out.Spend = spendEnv.Data.Spend
	}

	var updEnv struct {
		Data struct {
			Updates []Update `json:"updates"`
		} `json:"data"`
	}
	if err := s.get(ctx, "/v1/campaigns/"+campaignID+"/updates", bearer, &updEnv); err == nil {
		out.Updates = updEnv.Data.Updates
	}

	return out, nil
}

func (s *SourceClient) get(ctx context.Context, path, bearer string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.organizationURL+path, nil)
	if err != nil {
		return &SourceError{Status: http.StatusInternalServerError, Code: "INTERNAL_ERROR", Message: err.Error()}
	}
	if bearer != "" {
		req.Header.Set("Authorization", bearer)
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return &SourceError{Status: http.StatusBadGateway, Code: "SOURCE_UNREACHABLE", Message: "Could not reach the campaign service"}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return &SourceError{Status: http.StatusNotFound, Code: "CAMPAIGN_NOT_FOUND", Message: "Campaign not found"}
	}
	// Pass the upstream's refusal through unchanged rather than reinterpreting
	// it: if the caller may not read this campaign, that is the answer.
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
		return &SourceError{Status: resp.StatusCode, Code: "FORBIDDEN", Message: "You do not have access to this campaign"}
	}
	if resp.StatusCode >= 300 {
		return &SourceError{Status: http.StatusBadGateway, Code: "SOURCE_ERROR",
			Message: fmt.Sprintf("Campaign service returned %d", resp.StatusCode)}
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return &SourceError{Status: http.StatusBadGateway, Code: "SOURCE_ERROR", Message: "Could not read campaign data"}
	}
	if err := json.Unmarshal(body, out); err != nil {
		return &SourceError{Status: http.StatusBadGateway, Code: "SOURCE_ERROR", Message: "Campaign data was not in the expected shape"}
	}
	return nil
}
