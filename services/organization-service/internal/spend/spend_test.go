package spend

import (
	"errors"
	"testing"
	"time"

	"github.com/civicos/organization-service/internal/domain"
	"gorm.io/gorm"
)

type fakeStore struct {
	records   map[string]*domain.SpendRecord
	campaigns map[string]*domain.Campaign
	milestone map[string]*domain.Milestone
}

func newFake() *fakeStore {
	return &fakeStore{
		records:   map[string]*domain.SpendRecord{},
		campaigns: map[string]*domain.Campaign{},
		milestone: map[string]*domain.Milestone{},
	}
}

func (f *fakeStore) Create(r *domain.SpendRecord) error {
	cp := *r
	f.records[r.ID] = &cp
	return nil
}
func (f *fakeStore) Get(id string) (*domain.SpendRecord, error) {
	if r, ok := f.records[id]; ok {
		return r, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *fakeStore) Update(r *domain.SpendRecord) error { f.records[r.ID] = r; return nil }
func (f *fakeStore) Delete(id string) error             { delete(f.records, id); return nil }
func (f *fakeStore) ListForCampaign(id string) ([]domain.SpendRecord, error) {
	var out []domain.SpendRecord
	for _, r := range f.records {
		if r.CampaignID == id {
			out = append(out, *r)
		}
	}
	return out, nil
}
func (f *fakeStore) Campaign(id string) (*domain.Campaign, error) {
	if c, ok := f.campaigns[id]; ok {
		return c, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *fakeStore) Milestone(id string) (*domain.Milestone, error) {
	if m, ok := f.milestone[id]; ok {
		return m, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func fixture(t *testing.T) (*fakeStore, *Service, string, string) {
	t.Helper()
	st := newFake()
	campID, msID := "camp-1", "ms-1"
	st.campaigns[campID] = &domain.Campaign{
		ID: campID, OrganizationID: "org-1", Status: domain.CampaignPublished,
		Currency: "NGN", GoalMinor: 200_000_000,
	}
	st.milestone[msID] = &domain.Milestone{ID: msID, CampaignID: campID, Title: "Food and water"}
	return st, NewService(st), campID, msID
}

func isCode(err error, code string) bool {
	var e *AppError
	return errors.As(err, &e) && e.Code == code
}

func mustCreate(t *testing.T, svc *Service, campID, msID string, amount int64) *domain.SpendRecord {
	t.Helper()
	r, err := svc.Create(campID, CreateInput{
		MilestoneID: msID, AmountMinor: amount,
		Description: "Bought 200 bags of rice", SpentAt: "2026-07-20",
	}, "user-1", "Ada Owner")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	return r
}

// ─── Publishing ─────────────────────────────────────────────────────────

func TestCreate_RecordsTheClaimWithAttribution(t *testing.T) {
	st, svc, campID, msID := fixture(t)

	r := mustCreate(t, svc, campID, msID, 5_000_00)

	if r.CampaignID != campID || r.MilestoneID != msID {
		t.Fatalf("record not tied to the plan: %+v", r)
	}
	if r.OrganizationID != "org-1" {
		t.Fatalf("organizationId = %q, want the campaign's org", r.OrganizationID)
	}
	if r.Currency != "NGN" {
		t.Fatalf("currency = %q — it must come from the campaign, not the caller", r.Currency)
	}
	// Attribution is part of accountability: a claim published under nobody's
	// name is not accountable to anyone.
	if r.PublishedByID != "user-1" || r.PublishedByName != "Ada Owner" {
		t.Fatalf("attribution missing: %+v", r)
	}
	if _, ok := st.records[r.ID]; !ok {
		t.Fatal("record not persisted")
	}
}

// Spend attributed to another campaign's milestone would credit this
// organization's spending against someone else's plan.
func TestCreate_RejectsAMilestoneFromAnotherCampaign(t *testing.T) {
	st, svc, campID, _ := fixture(t)
	st.milestone["ms-other"] = &domain.Milestone{ID: "ms-other", CampaignID: "some-other-campaign"}

	_, err := svc.Create(campID, CreateInput{
		MilestoneID: "ms-other", AmountMinor: 1000, Description: "x", SpentAt: "2026-07-20",
	}, "u", "n")

	if !isCode(err, "MILESTONE_MISMATCH") {
		t.Fatalf("want MILESTONE_MISMATCH, got %v", err)
	}
}

// A campaign that never went live cannot have raised money through CivicOS,
// so reporting spend against it describes something that did not happen here.
func TestCreate_RefusedBeforeTheCampaignIsLive(t *testing.T) {
	st, svc, campID, msID := fixture(t)
	for _, status := range []domain.CampaignStatus{
		domain.CampaignDraft, domain.CampaignPendingReview,
		domain.CampaignApproved, domain.CampaignRejected,
	} {
		st.campaigns[campID].Status = status
		_, err := svc.Create(campID, CreateInput{
			MilestoneID: msID, AmountMinor: 1000, Description: "x", SpentAt: "2026-07-20",
		}, "u", "n")
		if !isCode(err, "CAMPAIGN_NOT_REPORTABLE") {
			t.Errorf("status %s: want CAMPAIGN_NOT_REPORTABLE, got %v", status, err)
		}
	}
}

// A paused campaign must still be able to report — pausing stops new
// donations, and an org under investigation is exactly who should be
// accounting for what it already took.
func TestCreate_AllowedWhilePaused(t *testing.T) {
	st, svc, campID, msID := fixture(t)
	st.campaigns[campID].Status = domain.CampaignPaused

	if _, err := svc.Create(campID, CreateInput{
		MilestoneID: msID, AmountMinor: 1000, Description: "x", SpentAt: "2026-07-20",
	}, "u", "n"); err != nil {
		t.Fatalf("a paused campaign must still be able to account for its money: %v", err)
	}
}

func TestCreate_RejectsNonsenseAmounts(t *testing.T) {
	_, svc, campID, msID := fixture(t)
	for _, amount := range []int64{0, -1, maxAmountMinor + 1} {
		_, err := svc.Create(campID, CreateInput{
			MilestoneID: msID, AmountMinor: amount, Description: "x", SpentAt: "2026-07-20",
		}, "u", "n")
		if !isCode(err, "INVALID_AMOUNT") {
			t.Errorf("amount %d: want INVALID_AMOUNT, got %v", amount, err)
		}
	}
}

// Future-dated spend would let a campaign show its money accounted for on
// the strength of a typo.
func TestCreate_RejectsFutureSpend(t *testing.T) {
	_, svc, campID, msID := fixture(t)
	future := time.Now().UTC().Add(72 * time.Hour).Format("2006-01-02")

	_, err := svc.Create(campID, CreateInput{
		MilestoneID: msID, AmountMinor: 1000, Description: "x", SpentAt: future,
	}, "u", "n")

	if !isCode(err, "VALIDATION_ERROR") {
		t.Fatalf("want VALIDATION_ERROR for a future date, got %v", err)
	}
}

func TestCreate_AcceptsDateOrTimestampAndNormalisesToUTC(t *testing.T) {
	_, svc, campID, msID := fixture(t)
	for _, in := range []string{"2026-07-20", "2026-07-20T14:30:00+01:00"} {
		r, err := svc.Create(campID, CreateInput{
			MilestoneID: msID, AmountMinor: 1000, Description: "x", SpentAt: in,
		}, "u", "n")
		if err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		if r.SpentAt.Location() != time.UTC {
			t.Errorf("%s stored as %v, want UTC", in, r.SpentAt.Location())
		}
	}
}

// ─── Aggregation ────────────────────────────────────────────────────────

func TestSummarise_AccountsForWhatCameIn(t *testing.T) {
	records := []domain.SpendRecord{
		{MilestoneID: "ms-1", AmountMinor: 300_000},
		{MilestoneID: "ms-1", AmountMinor: 200_000},
		{MilestoneID: "ms-2", AmountMinor: 100_000},
	}

	s := Summarise(records, 1_000_000, "NGN")

	if s.ReportedMinor != 600_000 {
		t.Fatalf("reported = %d, want 600000", s.ReportedMinor)
	}
	if s.UnreportedMinor != 400_000 {
		t.Fatalf("unreported = %d, want 400000", s.UnreportedMinor)
	}
	if s.ExceedsReceived {
		t.Fatal("reported is below received; ExceedsReceived should be false")
	}
	if s.PerMilestone["ms-1"] != 500_000 || s.PerMilestone["ms-2"] != 100_000 {
		t.Fatalf("per-milestone wrong: %v", s.PerMilestone)
	}
	if s.RecordCount != 3 {
		t.Fatalf("recordCount = %d, want 3", s.RecordCount)
	}
}

// An organization may legitimately spend more than it raised here, topping up
// from its own funds. Silently clamping that would make the page's
// arithmetic not add up; it is surfaced instead.
func TestSummarise_SurfacesSpendingBeyondWhatWasRaised(t *testing.T) {
	s := Summarise([]domain.SpendRecord{{MilestoneID: "ms-1", AmountMinor: 1_500_000}}, 1_000_000, "NGN")

	if !s.ExceedsReceived {
		t.Fatal("ExceedsReceived should be true when reported outstrips received")
	}
	if s.UnreportedMinor != 0 {
		t.Fatalf("unreported = %d, want 0 — it must never go negative", s.UnreportedMinor)
	}
	if s.ReportedMinor != 1_500_000 {
		t.Fatalf("reported figure was altered: %d", s.ReportedMinor)
	}
}

func TestSummarise_NothingReportedYet(t *testing.T) {
	s := Summarise(nil, 1_000_000, "NGN")

	if s.ReportedMinor != 0 || s.UnreportedMinor != 1_000_000 {
		t.Fatalf("got reported=%d unreported=%d, want 0 and 1000000", s.ReportedMinor, s.UnreportedMinor)
	}
	if s.PerMilestone == nil {
		t.Fatal("perMilestone must serialise as an object, not null")
	}
}

// ─── Amending ───────────────────────────────────────────────────────────

func TestUpdate_ChangesTheFigure(t *testing.T) {
	st, svc, campID, msID := fixture(t)
	r := mustCreate(t, svc, campID, msID, 500_000)

	newAmount := int64(450_000)
	updated, err := svc.Update(r, UpdateInput{AmountMinor: &newAmount})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.AmountMinor != 450_000 {
		t.Fatalf("amount = %d, want 450000", updated.AmountMinor)
	}
	if st.records[r.ID].AmountMinor != 450_000 {
		t.Fatal("change not persisted")
	}
}

func TestUpdate_StillRefusesNonsense(t *testing.T) {
	_, svc, campID, msID := fixture(t)
	r := mustCreate(t, svc, campID, msID, 500_000)

	bad := int64(0)
	if _, err := svc.Update(r, UpdateInput{AmountMinor: &bad}); !isCode(err, "INVALID_AMOUNT") {
		t.Fatalf("want INVALID_AMOUNT, got %v", err)
	}
	future := time.Now().UTC().Add(72 * time.Hour).Format("2006-01-02")
	if _, err := svc.Update(r, UpdateInput{SpentAt: &future}); !isCode(err, "VALIDATION_ERROR") {
		t.Fatalf("want VALIDATION_ERROR, got %v", err)
	}
}
