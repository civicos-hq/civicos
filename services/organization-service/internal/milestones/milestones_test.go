package milestones

import (
	"testing"
	"time"

	"github.com/civicos/organization-service/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// The spend plan must never promise more than the campaign is asking for.
// Checked on create and on edit — the edit path is the subtle one, because
// it has to exclude the milestone's own current target from the total or a
// no-op save would fail its own validation.
func TestTargetsCannotExceedGoal(t *testing.T) {
	store := newFakeStore()
	campaignID := store.addCampaign(domain.CampaignDraft, 100_000)
	svc := NewService(store)

	first, err := svc.Create(campaignID, CreateInput{Title: "Purchase supplies", TargetMinor: 60_000})
	if err != nil {
		t.Fatalf("create first: %v", err)
	}
	if first.Position != 1 {
		t.Fatalf("expected first milestone at position 1, got %d", first.Position)
	}
	if first.Status != domain.MilestonePlanned {
		t.Fatalf("expected PLANNED, got %s", first.Status)
	}

	// 60k + 50k > 100k → refused.
	if _, err := svc.Create(campaignID, CreateInput{Title: "Transport", TargetMinor: 50_000}); !isCode(err, "MILESTONES_EXCEED_GOAL") {
		t.Fatalf("expected MILESTONES_EXCEED_GOAL, got %v", err)
	}

	// 60k + 40k == 100k → exactly the goal is allowed.
	second, err := svc.Create(campaignID, CreateInput{Title: "Transport", TargetMinor: 40_000})
	if err != nil {
		t.Fatalf("create second: %v", err)
	}
	if second.Position != 2 {
		t.Fatalf("expected position 2, got %d", second.Position)
	}

	// Editing a milestone to the same value must not trip the cap — the
	// check has to exclude the row being edited.
	same := int64(40_000)
	if _, err := svc.Update(second.ID, UpdateInput{TargetMinor: &same}); err != nil {
		t.Fatalf("no-op target edit should be allowed: %v", err)
	}

	// Raising it past the remaining headroom must fail.
	tooBig := int64(40_001)
	if _, err := svc.Update(second.ID, UpdateInput{TargetMinor: &tooBig}); !isCode(err, "MILESTONES_EXCEED_GOAL") {
		t.Fatalf("expected MILESTONES_EXCEED_GOAL on edit, got %v", err)
	}

	// Zero and negative are rejected outright.
	for _, bad := range []int64{0, -1} {
		if _, err := svc.Create(campaignID, CreateInput{Title: "x", TargetMinor: bad}); !isCode(err, "INVALID_TARGET") {
			t.Fatalf("expected INVALID_TARGET for %d, got %v", bad, err)
		}
	}
}

// Plan edits are frozen once the campaign leaves the org's hands, but
// marking progress is not — an org must be able to report a milestone done
// on a live campaign.
func TestPlanFrozenButProgressAllowed(t *testing.T) {
	store := newFakeStore()
	campaignID := store.addCampaign(domain.CampaignDraft, 10_000)
	svc := NewService(store)

	m, err := svc.Create(campaignID, CreateInput{Title: "Supplies", TargetMinor: 5_000})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Campaign goes live.
	store.campaigns[campaignID].Status = domain.CampaignPublished

	// No new milestones.
	if _, err := svc.Create(campaignID, CreateInput{Title: "Extra", TargetMinor: 100}); !isCode(err, "CAMPAIGN_NOT_EDITABLE") {
		t.Fatalf("expected CAMPAIGN_NOT_EDITABLE on create, got %v", err)
	}
	// No plan edits.
	newTitle := "Renamed"
	if _, err := svc.Update(m.ID, UpdateInput{Title: &newTitle}); !isCode(err, "CAMPAIGN_NOT_EDITABLE") {
		t.Fatalf("expected CAMPAIGN_NOT_EDITABLE on title edit, got %v", err)
	}
	// No deletes.
	if err := svc.Delete(m.ID); !isCode(err, "CAMPAIGN_NOT_EDITABLE") {
		t.Fatalf("expected CAMPAIGN_NOT_EDITABLE on delete, got %v", err)
	}

	// But progress reporting works, and stamps completion.
	done, err := svc.SetStatus(m.ID, "completed")
	if err != nil {
		t.Fatalf("SetStatus: %v", err)
	}
	if done.Status != domain.MilestoneCompleted {
		t.Fatalf("expected COMPLETED, got %s", done.Status)
	}
	if done.CompletedAt == nil {
		t.Fatalf("completedAt should be stamped")
	}

	// Reverting clears the stamp rather than leaving a stale completion date.
	back, err := svc.SetStatus(m.ID, "IN_PROGRESS")
	if err != nil {
		t.Fatalf("SetStatus back: %v", err)
	}
	if back.CompletedAt != nil {
		t.Fatalf("completedAt should be cleared when un-completing")
	}

	if _, err := svc.SetStatus(m.ID, "DONE_ISH"); !isCode(err, "INVALID_STATUS") {
		t.Fatalf("expected INVALID_STATUS, got %v", err)
	}
}

func TestNeedsChangesIsEditable(t *testing.T) {
	store := newFakeStore()
	campaignID := store.addCampaign(domain.CampaignNeedsChanges, 10_000)
	svc := NewService(store)
	if _, err := svc.Create(campaignID, CreateInput{Title: "Revised line item", TargetMinor: 1_000}); err != nil {
		t.Fatalf("NEEDS_CHANGES should be editable: %v", err)
	}
}

// ─── Fake store ─────────────────────────────────────────────────────────

func isCode(err error, code string) bool {
	appErr, ok := err.(*AppError)
	return ok && appErr.Code == code
}

type fakeStore struct {
	campaigns  map[string]*domain.Campaign
	milestones map[string]*domain.Milestone
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		campaigns:  map[string]*domain.Campaign{},
		milestones: map[string]*domain.Milestone{},
	}
}

func (f *fakeStore) addCampaign(status domain.CampaignStatus, goal int64) string {
	id := uuid.NewString()
	f.campaigns[id] = &domain.Campaign{ID: id, Status: status, GoalMinor: goal}
	return id
}

func (f *fakeStore) FindByCampaign(campaignID string) ([]domain.Milestone, error) {
	var out []domain.Milestone
	for _, m := range f.milestones {
		if m.CampaignID == campaignID {
			out = append(out, *m)
		}
	}
	return out, nil
}

func (f *fakeStore) FindByID(id string) (*domain.Milestone, error) {
	m, ok := f.milestones[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copied := *m
	return &copied, nil
}

func (f *fakeStore) Create(m *domain.Milestone) error {
	copied := *m
	f.milestones[m.ID] = &copied
	return nil
}

func (f *fakeStore) Update(id string, updates map[string]any) error {
	m, ok := f.milestones[id]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	for k, v := range updates {
		switch k {
		case "title":
			m.Title = v.(string)
		case "description":
			s := v.(string)
			m.Description = &s
		case "target_minor":
			m.TargetMinor = v.(int64)
		case "position":
			m.Position = v.(int)
		case "status":
			m.Status = v.(domain.MilestoneStatus)
		case "completed_at":
			if v == nil {
				m.CompletedAt = nil
			} else if tv, ok := v.(time.Time); ok {
				m.CompletedAt = &tv
			}
		default:
			panic("fakeStore: unhandled column " + k)
		}
	}
	return nil
}

func (f *fakeStore) Delete(id string) error {
	delete(f.milestones, id)
	return nil
}

func (f *fakeStore) NextPosition(campaignID string) (int, error) {
	max := 0
	for _, m := range f.milestones {
		if m.CampaignID == campaignID && m.Position > max {
			max = m.Position
		}
	}
	return max + 1, nil
}

func (f *fakeStore) SumTargetsExcluding(campaignID, excludeID string) (int64, error) {
	var total int64
	for _, m := range f.milestones {
		if m.CampaignID == campaignID && m.ID != excludeID {
			total += m.TargetMinor
		}
	}
	return total, nil
}

func (f *fakeStore) Campaign(campaignID string) (*domain.Campaign, error) {
	c, ok := f.campaigns[campaignID]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copied := *c
	return &copied, nil
}
