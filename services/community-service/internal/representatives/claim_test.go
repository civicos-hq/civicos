package representatives

import (
	"testing"

	"github.com/civicos/community-service/internal/domain"
	"gorm.io/gorm"
)

type fakeClaimRepo struct {
	RepresentativeStore
	reps  map[string]*domain.Representative
	users map[string]*userRecord
}

func (f *fakeClaimRepo) FindByID(id string) (*domain.Representative, error) {
	if r, ok := f.reps[id]; ok {
		return r, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeClaimRepo) FindUser(id string) (*userRecord, error) {
	if u, ok := f.users[id]; ok {
		return u, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeClaimRepo) ClaimProfile(repID, userID string) error {
	rep, ok := f.reps[repID]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	// Mirrors the repository's `WHERE user_id IS NULL` guard.
	if rep.UserID != nil && *rep.UserID != "" {
		return gorm.ErrRecordNotFound
	}
	rep.UserID = &userID
	return nil
}

func claimFixture() (*Service, *fakeClaimRepo) {
	repo := &fakeClaimRepo{
		reps: map[string]*domain.Representative{
			"rep-1": {ID: "rep-1", Name: "Ada Okafor"},
		},
		users: map[string]*userRecord{
			"user-1": {ID: "user-1", Name: "Ada Okafor", Role: "REPRESENTATIVE"},
		},
	}
	return NewService(repo), repo
}

func TestClaimLinksUnclaimedProfile(t *testing.T) {
	svc, repo := claimFixture()

	rep, err := svc.ClaimProfile("rep-1", "user-1", "PLATFORM_ADMIN")
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if rep.UserID == nil || *rep.UserID != "user-1" {
		t.Fatalf("expected profile linked to user-1, got %v", rep.UserID)
	}
	_ = repo
}

// The gate is deliberately narrower than the rest of this module, which
// also admits GOVERNMENT_ADMIN and NGO.
func TestClaimIsPlatformAdminOnly(t *testing.T) {
	for _, role := range []string{"GOVERNMENT_ADMIN", "NGO", "REPRESENTATIVE", "CITIZEN", ""} {
		svc, _ := claimFixture()
		_, err := svc.ClaimProfile("rep-1", "user-1", role)
		var appErr *AppError
		if !asAppError(err, &appErr) || appErr.Code != "FORBIDDEN" {
			t.Fatalf("role %q must be refused, got %v", role, err)
		}
	}
}

// Reassigning a claimed profile would hand one official's constituents —
// and their donors — to a different account. It has to be a deliberate
// unlink first, never a side effect of claiming.
func TestClaimWillNotDisplaceAnExistingClaim(t *testing.T) {
	svc, repo := claimFixture()
	existing := "someone-else"
	repo.reps["rep-1"].UserID = &existing
	repo.users["user-2"] = &userRecord{ID: "user-2", Name: "Bode", Role: "REPRESENTATIVE"}

	_, err := svc.ClaimProfile("rep-1", "user-2", "PLATFORM_ADMIN")
	var appErr *AppError
	if !asAppError(err, &appErr) || appErr.Code != "REPRESENTATIVE_ALREADY_CLAIMED" {
		t.Fatalf("expected REPRESENTATIVE_ALREADY_CLAIMED, got %v", err)
	}
	if *repo.reps["rep-1"].UserID != "someone-else" {
		t.Fatal("the existing claim must be untouched")
	}
}

// Linking is not a promotion: the role change goes through the approval
// flow, which records a reviewer. Granting it here would leave no trace of
// who decided this person holds office.
func TestClaimRequiresTheAccountToAlreadyBeARepresentative(t *testing.T) {
	svc, repo := claimFixture()
	repo.users["citizen-1"] = &userRecord{ID: "citizen-1", Name: "Chidi", Role: "CITIZEN"}

	_, err := svc.ClaimProfile("rep-1", "citizen-1", "PLATFORM_ADMIN")
	var appErr *AppError
	if !asAppError(err, &appErr) || appErr.Code != "USER_NOT_REPRESENTATIVE" {
		t.Fatalf("expected USER_NOT_REPRESENTATIVE, got %v", err)
	}
}

func TestClaimRejectsDeletedAccount(t *testing.T) {
	svc, repo := claimFixture()
	when := "2026-01-01T00:00:00Z"
	repo.users["user-1"].DeletedAt = &when

	_, err := svc.ClaimProfile("rep-1", "user-1", "PLATFORM_ADMIN")
	var appErr *AppError
	if !asAppError(err, &appErr) || appErr.Code != "USER_DELETED" {
		t.Fatalf("expected USER_DELETED, got %v", err)
	}
}

func TestClaimRejectsUnknownProfileAndUser(t *testing.T) {
	svc, _ := claimFixture()

	if _, err := svc.ClaimProfile("nope", "user-1", "PLATFORM_ADMIN"); err == nil {
		t.Fatal("expected unknown profile to fail")
	}
	_, err := svc.ClaimProfile("rep-1", "nobody", "PLATFORM_ADMIN")
	var appErr *AppError
	if !asAppError(err, &appErr) || appErr.Code != "USER_NOT_FOUND" {
		t.Fatalf("expected USER_NOT_FOUND, got %v", err)
	}
}

func asAppError(err error, target **AppError) bool {
	if e, ok := err.(*AppError); ok {
		*target = e
		return true
	}
	return false
}
