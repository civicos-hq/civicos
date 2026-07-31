// Package spend records what an organization says it did with the money.
//
// Because donations settle straight to the organization's own Paystack
// sub-account, CivicOS never takes custody and therefore cannot verify any
// of this. What is recorded here is the organization's own account of its
// spending — a claim, published under its name, tied to the plan donors were
// shown before they gave.
//
// That is a weaker guarantee than holding the money and releasing it against
// milestones, and it is the guarantee the merchant-of-record decision left
// us with. The design leans into it: make the claim specific, dated,
// itemised and attributed, so that it is checkable by the people who paid
// even though it is not enforceable by us.
package spend

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/civicos/organization-service/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Store interface {
	Create(r *domain.SpendRecord) error
	Get(id string) (*domain.SpendRecord, error)
	Update(r *domain.SpendRecord) error
	Delete(id string) error
	ListForCampaign(campaignID string) ([]domain.SpendRecord, error)
	Campaign(id string) (*domain.Campaign, error)
	Milestone(id string) (*domain.Milestone, error)
}

type Service struct{ repo Store }

func NewService(repo Store) *Service { return &Service{repo: repo} }

type CreateInput struct {
	MilestoneID string  `json:"milestoneId" binding:"required,uuid"`
	AmountMinor int64   `json:"amountMinor" binding:"required"`
	Description string  `json:"description" binding:"required,min=3,max=2000"`
	SpentAt     string  `json:"spentAt" binding:"required"`
	ReceiptURL  *string `json:"receiptUrl"`
}

type UpdateInput struct {
	AmountMinor *int64  `json:"amountMinor"`
	Description *string `json:"description"`
	SpentAt     *string `json:"spentAt"`
	ReceiptURL  *string `json:"receiptUrl"`
}

// maxAmountMinor mirrors the donations ledger's ceiling so a typo cannot
// publish an absurd figure that dwarfs every real number on the page.
const maxAmountMinor int64 = 10_000_000_000_000

// Create publishes a spend record against a milestone of a campaign.
func (s *Service) Create(campaignID string, in CreateInput, byID, byName string) (*domain.SpendRecord, error) {
	c, err := s.repo.Campaign(campaignID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &AppError{Code: "CAMPAIGN_NOT_FOUND", Message: "Campaign not found", Status: http.StatusNotFound}
	} else if err != nil {
		return nil, err
	}

	// Reporting spend before a campaign is live would describe money that
	// cannot have been raised through it.
	if !reportableStatus(c.Status) {
		return nil, &AppError{
			Code:    "CAMPAIGN_NOT_REPORTABLE",
			Message: "Spend can only be reported on a campaign that has been published",
			Status:  http.StatusConflict,
		}
	}

	m, err := s.repo.Milestone(in.MilestoneID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &AppError{Code: "MILESTONE_NOT_FOUND", Message: "Milestone not found", Status: http.StatusNotFound}
	} else if err != nil {
		return nil, err
	}
	// A milestone from another campaign would attribute this org's spending
	// to someone else's plan.
	if m.CampaignID != c.ID {
		return nil, &AppError{
			Code:    "MILESTONE_MISMATCH",
			Message: "That milestone belongs to a different campaign",
			Status:  http.StatusBadRequest,
		}
	}

	amount, err := validAmount(in.AmountMinor)
	if err != nil {
		return nil, err
	}
	spentAt, err := parseSpentAt(in.SpentAt)
	if err != nil {
		return nil, err
	}

	rec := &domain.SpendRecord{
		ID:              uuid.New().String(),
		CampaignID:      c.ID,
		MilestoneID:     m.ID,
		OrganizationID:  c.OrganizationID,
		AmountMinor:     amount,
		Currency:        c.Currency,
		Description:     strings.TrimSpace(in.Description),
		SpentAt:         spentAt,
		ReceiptURL:      trimmedPtr(in.ReceiptURL),
		PublishedByID:   byID,
		PublishedByName: byName,
	}
	if err := s.repo.Create(rec); err != nil {
		return nil, err
	}
	return rec, nil
}

func (s *Service) Get(id string) (*domain.SpendRecord, error) {
	r, err := s.repo.Get(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &AppError{Code: "SPEND_NOT_FOUND", Message: "Spend record not found", Status: http.StatusNotFound}
	}
	return r, err
}

func (s *Service) Update(rec *domain.SpendRecord, in UpdateInput) (*domain.SpendRecord, error) {
	if in.AmountMinor != nil {
		amount, err := validAmount(*in.AmountMinor)
		if err != nil {
			return nil, err
		}
		rec.AmountMinor = amount
	}
	if in.Description != nil {
		d := strings.TrimSpace(*in.Description)
		if len(d) < 3 {
			return nil, &AppError{Code: "VALIDATION_ERROR", Message: "Description is too short", Status: http.StatusBadRequest}
		}
		rec.Description = d
	}
	if in.SpentAt != nil {
		t, err := parseSpentAt(*in.SpentAt)
		if err != nil {
			return nil, err
		}
		rec.SpentAt = t
	}
	if in.ReceiptURL != nil {
		rec.ReceiptURL = trimmedPtr(in.ReceiptURL)
	}
	if err := s.repo.Update(rec); err != nil {
		return nil, err
	}
	return rec, nil
}

func (s *Service) Delete(id string) error { return s.repo.Delete(id) }

func (s *Service) ListForCampaign(campaignID string) ([]domain.SpendRecord, error) {
	return s.repo.ListForCampaign(campaignID)
}

// ─── Aggregation ────────────────────────────────────────────────────────

// Summary is the campaign-level accounting shown to the public.
//
// Deliberately does NOT compute a single "remaining balance" figure. CivicOS
// does not hold the money and cannot know an organization's actual balance —
// only what came through this platform and what the organization says it
// spent. Presenting a derived balance as fact would be inventing a number.
type Summary struct {
	Currency string `json:"currency"`
	// ReceivedMinor is ledger truth: settled donations through CivicOS.
	ReceivedMinor int64 `json:"receivedMinor"`
	// ReportedMinor is the organization's claim, summed.
	ReportedMinor int64 `json:"reportedMinor"`
	// UnreportedMinor is what has come in but has not been accounted for.
	// Floored at zero — see ExceedsReceived.
	UnreportedMinor int64 `json:"unreportedMinor"`
	// ExceedsReceived is true when an organization reports spending more
	// than it raised here. That is legitimate (they may have topped up from
	// other funds) and is surfaced rather than hidden, because silently
	// clamping it would make the arithmetic on the page not add up.
	ExceedsReceived bool `json:"exceedsReceived"`
	// RecordCount is how many individual entries back the total.
	RecordCount int `json:"recordCount"`
	// PerMilestone is the reported total for each milestone id.
	PerMilestone map[string]int64 `json:"perMilestone"`
}

// Summarise aggregates spend records against what the campaign received.
func Summarise(records []domain.SpendRecord, receivedMinor int64, currency string) Summary {
	sum := Summary{
		Currency:      currency,
		ReceivedMinor: receivedMinor,
		PerMilestone:  map[string]int64{},
		RecordCount:   len(records),
	}
	for _, r := range records {
		sum.ReportedMinor += r.AmountMinor
		sum.PerMilestone[r.MilestoneID] += r.AmountMinor
	}
	if sum.ReportedMinor > receivedMinor {
		sum.ExceedsReceived = true
		sum.UnreportedMinor = 0
	} else {
		sum.UnreportedMinor = receivedMinor - sum.ReportedMinor
	}
	return sum
}

// ─── Helpers ────────────────────────────────────────────────────────────

// reportableStatus lists the campaign states in which spend may be
// published. Anything before PUBLISHED has not taken money through CivicOS.
func reportableStatus(s domain.CampaignStatus) bool {
	switch s {
	case domain.CampaignPublished, domain.CampaignFunded, domain.CampaignPaused,
		domain.CampaignCompleted, domain.CampaignReported, domain.CampaignArchived:
		return true
	}
	return false
}

func validAmount(v int64) (int64, error) {
	if v <= 0 {
		return 0, &AppError{Code: "INVALID_AMOUNT", Message: "Amount must be greater than zero", Status: http.StatusBadRequest}
	}
	if v > maxAmountMinor {
		return 0, &AppError{Code: "INVALID_AMOUNT", Message: "Amount is implausibly large", Status: http.StatusBadRequest}
	}
	return v, nil
}

// parseSpentAt accepts a date or a full timestamp and normalises to UTC.
//
// A future date is refused: spending cannot be reported before it happens,
// and allowing it would let a campaign show its money accounted for on the
// strength of a typo.
func parseSpentAt(v string) (time.Time, error) {
	v = strings.TrimSpace(v)
	var t time.Time
	var err error
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		if t, err = time.Parse(layout, v); err == nil {
			break
		}
	}
	if err != nil {
		return time.Time{}, &AppError{
			Code:    "VALIDATION_ERROR",
			Message: "spentAt must be a date (YYYY-MM-DD) or an RFC3339 timestamp",
			Status:  http.StatusBadRequest,
		}
	}
	t = t.UTC()
	// A day of slack absorbs timezone skew from a client sending a local date.
	if t.After(time.Now().UTC().Add(24 * time.Hour)) {
		return time.Time{}, &AppError{
			Code:    "VALIDATION_ERROR",
			Message: "spentAt cannot be in the future",
			Status:  http.StatusBadRequest,
		}
	}
	return t, nil
}

func trimmedPtr(p *string) *string {
	if p == nil {
		return nil
	}
	v := strings.TrimSpace(*p)
	if v == "" {
		return nil
	}
	return &v
}

type AppError struct {
	Code    string
	Message string
	Status  int
}

func (e *AppError) Error() string { return e.Code + ": " + e.Message }
