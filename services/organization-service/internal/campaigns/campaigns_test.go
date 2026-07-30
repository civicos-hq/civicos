package campaigns

import (
	"testing"
	"time"

	"github.com/civicos/organization-service/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ─── Transition guard ───────────────────────────────────────────────────
//
// The guard table is the safety property of Phase 1: it is what stops a
// campaign reaching a fundable state without passing review. These tests
// hit CanTransition directly — no DB, no HTTP — so a regression in the
// lifecycle fails fast and unambiguously.

func TestCanTransition_HappyPath(t *testing.T) {
	steps := []struct {
		from, to domain.CampaignStatus
		actor    Actor
	}{
		{domain.CampaignDraft, domain.CampaignPendingReview, ActorOrg},
		{domain.CampaignPendingReview, domain.CampaignApproved, ActorPlatform},
		{domain.CampaignApproved, domain.CampaignPublished, ActorOrg},
		{domain.CampaignPublished, domain.CampaignFunded, ActorSystem},
		{domain.CampaignFunded, domain.CampaignCompleted, ActorOrg},
		{domain.CampaignCompleted, domain.CampaignReported, ActorOrg},
		{domain.CampaignReported, domain.CampaignArchived, ActorPlatform},
	}
	for _, s := range steps {
		if err := CanTransition(s.from, s.to, s.actor); err != nil {
			t.Errorf("%s → %s by %s should be allowed, got %v", s.from, s.to, s.actor, err)
		}
	}
}

// The core safety property: there is no path from DRAFT to a fundable
// status that skips PENDING_REVIEW.
func TestCanTransition_CannotSkipReview(t *testing.T) {
	forbidden := []domain.CampaignStatus{
		domain.CampaignApproved,
		domain.CampaignPublished,
		domain.CampaignFunded,
		domain.CampaignCompleted,
		domain.CampaignReported,
		domain.CampaignArchived,
	}
	for _, to := range forbidden {
		for _, actor := range []Actor{ActorOrg, ActorPlatform, ActorSystem} {
			if err := CanTransition(domain.CampaignDraft, to, actor); err == nil {
				t.Errorf("DRAFT → %s by %s must be refused", to, actor)
			}
		}
	}
}

// An organization must not be able to approve its own campaign, whichever
// actor value it presents.
func TestCanTransition_OrgCannotSelfApprove(t *testing.T) {
	for _, to := range []domain.CampaignStatus{
		domain.CampaignApproved, domain.CampaignNeedsChanges, domain.CampaignRejected,
	} {
		err := CanTransition(domain.CampaignPendingReview, to, ActorOrg)
		if err == nil {
			t.Fatalf("org must not be able to set %s", to)
		}
		appErr, ok := err.(*AppError)
		if !ok || appErr.Code != "CAMPAIGN_TRANSITION_FORBIDDEN" {
			t.Fatalf("expected CAMPAIGN_TRANSITION_FORBIDDEN for %s, got %v", to, err)
		}
	}
}

// "Goal reached" is ledger truth, not a claim anyone may assert.
func TestCanTransition_OnlySystemMarksFunded(t *testing.T) {
	if err := CanTransition(domain.CampaignPublished, domain.CampaignFunded, ActorSystem); err != nil {
		t.Fatalf("system should mark FUNDED: %v", err)
	}
	for _, actor := range []Actor{ActorOrg, ActorPlatform} {
		if err := CanTransition(domain.CampaignPublished, domain.CampaignFunded, actor); err == nil {
			t.Errorf("%s must not be able to declare the goal reached", actor)
		}
	}
}

func TestCanTransition_TerminalStatesAreFinal(t *testing.T) {
	for _, from := range []domain.CampaignStatus{domain.CampaignRejected, domain.CampaignArchived} {
		for _, to := range []domain.CampaignStatus{
			domain.CampaignDraft, domain.CampaignPublished, domain.CampaignPendingReview,
		} {
			err := CanTransition(from, to, ActorPlatform)
			if err == nil {
				t.Errorf("%s is terminal but allowed → %s", from, to)
			}
			appErr, ok := err.(*AppError)
			if !ok || appErr.Code != "CAMPAIGN_INVALID_TRANSITION" {
				t.Errorf("expected CAMPAIGN_INVALID_TRANSITION from %s, got %v", from, err)
			}
		}
	}
}

// Governance: a live campaign can be paused and brought back.
func TestCanTransition_PauseResume(t *testing.T) {
	if err := CanTransition(domain.CampaignPublished, domain.CampaignPaused, ActorPlatform); err != nil {
		t.Fatalf("platform should pause a published campaign: %v", err)
	}
	if err := CanTransition(domain.CampaignPaused, domain.CampaignPublished, ActorPlatform); err != nil {
		t.Fatalf("platform should resume a paused campaign: %v", err)
	}
	if err := CanTransition(domain.CampaignPublished, domain.CampaignPaused, ActorOrg); err == nil {
		t.Fatalf("an org must not be able to pause its own campaign to dodge scrutiny")
	}
}

// ─── Service behaviour, against a fake store ────────────────────────────

func TestSubmit_RequiresVerifiedOrgMilestonesAndBudget(t *testing.T) {
	orgID := uuid.NewString()

	// 1. Unverified org is refused.
	store := newFakeStore()
	store.verified[orgID] = false
	svc := NewService(store)
	c := mustCreate(t, svc, orgID)
	store.milestoneCount[c.ID] = 1
	store.milestoneTotal[c.ID] = 100
	if _, err := svc.Submit(c.ID); !isCode(err, "ORG_NOT_VERIFIED") {
		t.Fatalf("expected ORG_NOT_VERIFIED, got %v", err)
	}

	// 2. Verified but no spend plan is refused.
	store = newFakeStore()
	store.verified[orgID] = true
	svc = NewService(store)
	c = mustCreate(t, svc, orgID)
	if _, err := svc.Submit(c.ID); !isCode(err, "NO_MILESTONES") {
		t.Fatalf("expected NO_MILESTONES, got %v", err)
	}

	// 3. Milestones promising more than the goal is refused.
	store.milestoneCount[c.ID] = 2
	store.milestoneTotal[c.ID] = c.GoalMinor + 1
	if _, err := svc.Submit(c.ID); !isCode(err, "MILESTONES_EXCEED_GOAL") {
		t.Fatalf("expected MILESTONES_EXCEED_GOAL, got %v", err)
	}

	// 4. All three satisfied → PENDING_REVIEW.
	store.milestoneTotal[c.ID] = c.GoalMinor
	got, err := svc.Submit(c.ID)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if got.Status != domain.CampaignPendingReview {
		t.Fatalf("expected PENDING_REVIEW, got %s", got.Status)
	}
	if got.ApprovalStatus != "PENDING" {
		t.Fatalf("expected approvalStatus PENDING, got %s", got.ApprovalStatus)
	}
	if got.SubmittedAt == nil {
		t.Fatalf("submittedAt should be stamped")
	}
}

func TestReview_RequiresNoteUnlessApproving(t *testing.T) {
	store := newFakeStore()
	orgID := uuid.NewString()
	store.verified[orgID] = true
	svc := NewService(store)
	c := mustCreate(t, svc, orgID)
	store.milestoneCount[c.ID] = 1
	store.milestoneTotal[c.ID] = 1000
	if _, err := svc.Submit(c.ID); err != nil {
		t.Fatalf("submit: %v", err)
	}

	reviewer := uuid.NewString()
	if _, err := svc.Review(c.ID, "REJECTED", nil, reviewer); !isCode(err, "REVIEW_NOTE_REQUIRED") {
		t.Fatalf("expected REVIEW_NOTE_REQUIRED, got %v", err)
	}
	blank := "   "
	if _, err := svc.Review(c.ID, "NEEDS_CHANGES", &blank, reviewer); !isCode(err, "REVIEW_NOTE_REQUIRED") {
		t.Fatalf("whitespace note should not count, got %v", err)
	}
	if _, err := svc.Review(c.ID, "MAYBE", nil, reviewer); !isCode(err, "INVALID_DECISION") {
		t.Fatalf("expected INVALID_DECISION, got %v", err)
	}

	got, err := svc.Review(c.ID, "APPROVED", nil, reviewer)
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if got.Status != domain.CampaignApproved {
		t.Fatalf("expected APPROVED, got %s", got.Status)
	}
}

func TestUpdate_FrozenOncePublished(t *testing.T) {
	store := newFakeStore()
	orgID := uuid.NewString()
	store.verified[orgID] = true
	svc := NewService(store)
	c := mustCreate(t, svc, orgID)

	// Editable as a draft.
	newTitle := "Kaduna North flood relief"
	if _, err := svc.Update(c.ID, UpdateInput{Title: &newTitle}); err != nil {
		t.Fatalf("draft should be editable: %v", err)
	}

	// Push it live, then confirm content is frozen.
	store.milestoneCount[c.ID] = 1
	store.milestoneTotal[c.ID] = 500
	if _, err := svc.Submit(c.ID); err != nil {
		t.Fatalf("submit: %v", err)
	}
	if _, err := svc.Update(c.ID, UpdateInput{Title: &newTitle}); !isCode(err, "CAMPAIGN_NOT_EDITABLE") {
		t.Fatalf("a campaign in review must not be editable, got %v", err)
	}
}

func TestCreate_Validation(t *testing.T) {
	svc := NewService(newFakeStore())
	orgID := uuid.NewString()
	base := func() CreateInput {
		return CreateInput{
			Title:       "Solar streetlights for Otukpo market road",
			Summary:     "Replace the failed lighting along the market approach.",
			Description: "The market road has had no working streetlights since March, and traders leaving after dark have been robbed.",
			Category:    string(domain.CategoryCommunityDevelopment),
			GoalMinor:   150_000_00,
		}
	}

	bad := base()
	bad.Category = "SPACE_TRAVEL"
	if _, err := svc.Create(orgID, bad, "u", "U"); !isCode(err, "INVALID_CATEGORY") {
		t.Fatalf("expected INVALID_CATEGORY, got %v", err)
	}

	bad = base()
	bad.Currency = "XYZ"
	if _, err := svc.Create(orgID, bad, "u", "U"); !isCode(err, "UNSUPPORTED_CURRENCY") {
		t.Fatalf("expected UNSUPPORTED_CURRENCY, got %v", err)
	}

	bad = base()
	bad.GoalMinor = 0
	if _, err := svc.Create(orgID, bad, "u", "U"); !isCode(err, "INVALID_GOAL") {
		t.Fatalf("expected INVALID_GOAL for zero, got %v", err)
	}

	bad = base()
	bad.GoalMinor = -5
	if _, err := svc.Create(orgID, bad, "u", "U"); !isCode(err, "INVALID_GOAL") {
		t.Fatalf("expected INVALID_GOAL for negative, got %v", err)
	}

	bad = base()
	bad.GoalMinor = maxGoalMinor + 1
	if _, err := svc.Create(orgID, bad, "u", "U"); !isCode(err, "INVALID_GOAL") {
		t.Fatalf("expected INVALID_GOAL for absurd, got %v", err)
	}

	// Currency defaults to NGN, and the projection starts at zero.
	ok, err := svc.Create(orgID, base(), "u", "U")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if ok.Currency != "NGN" {
		t.Fatalf("expected default NGN, got %s", ok.Currency)
	}
	if ok.RaisedMinor != 0 || ok.DonorCount != 0 {
		t.Fatalf("a new campaign must start at zero raised")
	}
	if ok.Status != domain.CampaignDraft {
		t.Fatalf("expected DRAFT, got %s", ok.Status)
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Solar streetlights for Otukpo market road": "solar-streetlights-for-otukpo-market-road",
		"Flood relief — Kaduna North LGA (2026)":    "flood-relief-kaduna-north-lga-2026",
		"   ": "campaign",
		"???": "campaign",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestUniqueSlug_SuffixesOnCollision(t *testing.T) {
	store := newFakeStore()
	store.slugs["flood-relief"] = true
	store.slugs["flood-relief-2"] = true
	svc := NewService(store)
	got, err := svc.uniqueSlug("flood-relief")
	if err != nil {
		t.Fatalf("uniqueSlug: %v", err)
	}
	if got != "flood-relief-3" {
		t.Fatalf("expected flood-relief-3, got %s", got)
	}
}

func TestDelete_OnlyDrafts(t *testing.T) {
	store := newFakeStore()
	orgID := uuid.NewString()
	store.verified[orgID] = true
	svc := NewService(store)
	c := mustCreate(t, svc, orgID)

	store.milestoneCount[c.ID] = 1
	store.milestoneTotal[c.ID] = 100
	if _, err := svc.Submit(c.ID); err != nil {
		t.Fatalf("submit: %v", err)
	}
	if err := svc.Delete(c.ID); !isCode(err, "CAMPAIGN_NOT_DELETABLE") {
		t.Fatalf("expected CAMPAIGN_NOT_DELETABLE, got %v", err)
	}

	d := mustCreate(t, svc, orgID)
	if err := svc.Delete(d.ID); err != nil {
		t.Fatalf("draft should be deletable: %v", err)
	}
}

// ─── Helpers ────────────────────────────────────────────────────────────

func mustCreate(t *testing.T, svc *Service, orgID string) *domain.Campaign {
	t.Helper()
	c, err := svc.Create(orgID, CreateInput{
		Title:       "Flood relief",
		Summary:     "Emergency supplies for displaced families.",
		Description: "Sustained flooding has displaced households across the ward and immediate relief is needed.",
		Category:    string(domain.CategoryEmergencyRelief),
		GoalMinor:   1_000_00,
	}, uuid.NewString(), "Ada Admin")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	return c
}

func isCode(err error, code string) bool {
	appErr, ok := err.(*AppError)
	return ok && appErr.Code == code
}

// fakeStore is an in-memory Store. Same approach as
// consultations_test.go — exercise the service's rules without a DB.
type fakeStore struct {
	items          map[string]*domain.Campaign
	slugs          map[string]bool
	verified       map[string]bool
	milestoneCount map[string]int64
	milestoneTotal map[string]int64
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		items:          map[string]*domain.Campaign{},
		slugs:          map[string]bool{},
		verified:       map[string]bool{},
		milestoneCount: map[string]int64{},
		milestoneTotal: map[string]int64{},
	}
}

func (f *fakeStore) Find(_ ListFilters) ([]domain.Campaign, error) {
	out := make([]domain.Campaign, 0, len(f.items))
	for _, c := range f.items {
		out = append(out, *c)
	}
	return out, nil
}

func (f *fakeStore) FindByID(id string) (*domain.Campaign, error) {
	c, ok := f.items[id]
	if !ok {
		return nil, gormNotFound()
	}
	copied := *c
	return &copied, nil
}

func (f *fakeStore) FindBySlug(slug string) (*domain.Campaign, error) {
	for _, c := range f.items {
		if c.Slug == slug {
			copied := *c
			return &copied, nil
		}
	}
	return nil, gormNotFound()
}

func (f *fakeStore) Create(c *domain.Campaign) error {
	copied := *c
	f.items[c.ID] = &copied
	f.slugs[c.Slug] = true
	return nil
}

// Update mirrors the column names the service writes, so a typo in a
// snake_case key shows up here rather than silently no-op'ing in Postgres.
func (f *fakeStore) Update(id string, updates map[string]any) error {
	c, ok := f.items[id]
	if !ok {
		return gormNotFound()
	}
	for k, v := range updates {
		switch k {
		case "status":
			c.Status = v.(domain.CampaignStatus)
		case "approval_status":
			c.ApprovalStatus = v.(string)
		case "title":
			c.Title = v.(string)
		case "summary":
			c.Summary = v.(string)
		case "description":
			c.Description = v.(string)
		case "category":
			c.Category = domain.CampaignCategory(v.(string))
		case "goal_minor":
			c.GoalMinor = v.(int64)
		case "is_emergency":
			c.IsEmergency = v.(bool)
		case "paused_reason":
			if v == nil {
				c.PausedReason = nil
			} else {
				s := v.(string)
				c.PausedReason = &s
			}
		case "review_note":
			if v == nil {
				c.ReviewNote = nil
			} else {
				s := v.(string)
				c.ReviewNote = &s
			}
		case "reviewed_by_id":
			if v == nil {
				c.ReviewedByID = nil
			} else {
				s := v.(string)
				c.ReviewedByID = &s
			}
		case "submitted_at", "published_at", "completed_at", "reviewed_at":
			setStamp(c, k, v)
		case "cover_image_url", "community_id", "project_id", "state", "lga",
			"start_date", "end_date":
			// Not asserted on in these tests.
		default:
			panic("fakeStore: unhandled column " + k)
		}
	}
	return nil
}

func (f *fakeStore) Delete(id string) error {
	delete(f.items, id)
	return nil
}

func (f *fakeStore) SlugExists(slug string) (bool, error) { return f.slugs[slug], nil }

func (f *fakeStore) OrgIsVerified(orgID string) (bool, error) { return f.verified[orgID], nil }

func (f *fakeStore) CountMilestones(campaignID string) (int64, error) {
	return f.milestoneCount[campaignID], nil
}

func (f *fakeStore) SumMilestoneTargets(campaignID string) (int64, error) {
	return f.milestoneTotal[campaignID], nil
}

// setStamp mirrors the service's nullable-timestamp writes so the tests can
// assert that a stamp was set (or cleared) without duplicating the field
// mapping in every case branch.
func setStamp(c *domain.Campaign, column string, v any) {
	var t *time.Time
	if v != nil {
		if tv, ok := v.(time.Time); ok {
			t = &tv
		}
	}
	switch column {
	case "submitted_at":
		c.SubmittedAt = t
	case "published_at":
		c.PublishedAt = t
	case "completed_at":
		c.CompletedAt = t
	case "reviewed_at":
		c.ReviewedAt = t
	}
}

func gormNotFound() error { return gorm.ErrRecordNotFound }
