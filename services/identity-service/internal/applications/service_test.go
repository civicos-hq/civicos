package applications

import (
	"errors"
	"testing"
	"time"

	"github.com/civicos/identity-service/internal/domain"
	"gorm.io/gorm"
)

type fakeStore struct {
	user                     *domain.User
	rep                      *domain.RepresentativeApplication
	org                      *domain.OrganizationApplication
	reviewHistory            []domain.ApplicationReviewEvent
	approvedRepresentativeID string
	approvedOrganizationID   string
	approvedOrganizationRole domain.UserRole
	notifications            []string
}

func (f *fakeStore) FindUserByID(id string) (*domain.User, error) {
	if f.user != nil && f.user.ID == id {
		return f.user, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeStore) FindRepresentativeByUserID(userID string) (*domain.RepresentativeApplication, error) {
	if f.rep != nil && f.rep.UserID == userID {
		return f.rep, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeStore) FindOrganizationByUserID(userID string) (*domain.OrganizationApplication, error) {
	if f.org != nil && f.org.UserID == userID {
		return f.org, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeStore) FindRepresentativeByID(id string) (*domain.RepresentativeApplication, error) {
	if f.rep != nil && f.rep.ID == id {
		return f.rep, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeStore) FindOrganizationByID(id string) (*domain.OrganizationApplication, error) {
	if f.org != nil && f.org.ID == id {
		return f.org, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeStore) ListReviewHistory(kind domain.RequestedAccountType, applicationID string) ([]domain.ApplicationReviewEvent, error) {
	var out []domain.ApplicationReviewEvent
	for _, item := range f.reviewHistory {
		if item.ApplicationKind == kind && item.ApplicationID == applicationID {
			out = append(out, item)
		}
	}
	return out, nil
}

func (f *fakeStore) ListRepresentativeApplications(ListFilters) ([]domain.RepresentativeApplication, int64, error) {
	return nil, 0, nil
}

func (f *fakeStore) ListOrganizationApplications(ListFilters) ([]domain.OrganizationApplication, int64, error) {
	return nil, 0, nil
}

func (f *fakeStore) UpsertRepresentativeApplication(string, *domain.RepresentativeApplication) error {
	return errors.New("not implemented")
}

func (f *fakeStore) UpsertOrganizationApplication(string, *domain.OrganizationApplication) error {
	return errors.New("not implemented")
}

func (f *fakeStore) CreateNotification(userID, title, body string, linkURL *string) error {
	f.notifications = append(f.notifications, title)
	return nil
}

func (f *fakeStore) ApproveRepresentativeApplication(id, reviewerID string, note *string, reviewedAt time.Time) (*domain.RepresentativeApplication, error) {
	if f.rep == nil || f.rep.ID != id {
		return nil, gorm.ErrRecordNotFound
	}
	f.approvedRepresentativeID = id
	f.rep.Status = domain.ApprovalStatusApproved
	f.rep.ReviewedAt = &reviewedAt
	f.rep.ReviewedByUserID = &reviewerID
	f.rep.ReviewNote = note
	f.user.Role = domain.RoleRepresentative
	f.user.ApprovalStatus = domain.ApprovalStatusApproved
	return f.rep, nil
}

func (f *fakeStore) ApproveOrganizationApplication(id, reviewerID string, note *string, reviewedAt time.Time, userRole domain.UserRole) (*domain.OrganizationApplication, error) {
	if f.org == nil || f.org.ID != id {
		return nil, gorm.ErrRecordNotFound
	}
	f.approvedOrganizationID = id
	f.approvedOrganizationRole = userRole
	f.org.Status = domain.ApprovalStatusApproved
	f.org.ReviewedAt = &reviewedAt
	f.org.ReviewedByUserID = &reviewerID
	f.org.ReviewNote = note
	f.user.Role = userRole
	f.user.ApprovalStatus = domain.ApprovalStatusApproved
	return f.org, nil
}

func (f *fakeStore) ReviewRepresentativeApplication(id, reviewerID string, status domain.ApprovalStatus, note *string, reviewedAt time.Time) error {
	if f.rep == nil || f.rep.ID != id {
		return gorm.ErrRecordNotFound
	}
	f.rep.Status = status
	f.rep.ReviewedAt = &reviewedAt
	f.rep.ReviewedByUserID = &reviewerID
	f.rep.ReviewNote = note
	f.user.ApprovalStatus = status
	return nil
}

func (f *fakeStore) ReviewOrganizationApplication(id, reviewerID string, status domain.ApprovalStatus, note *string, reviewedAt time.Time) error {
	if f.org == nil || f.org.ID != id {
		return gorm.ErrRecordNotFound
	}
	f.org.Status = status
	f.org.ReviewedAt = &reviewedAt
	f.org.ReviewedByUserID = &reviewerID
	f.org.ReviewNote = note
	f.user.ApprovalStatus = status
	return nil
}

func TestReviewRepresentativeApprovalPromotesRole(t *testing.T) {
	store := &fakeStore{
		user: &domain.User{ID: "user-1", Role: domain.RoleCitizen, ApprovalStatus: domain.ApprovalStatusPending},
		rep:  &domain.RepresentativeApplication{ID: "rep-app-1", UserID: "user-1", Status: domain.ApprovalStatusPending},
	}
	svc := NewService(store)

	result, err := svc.Review("REPRESENTATIVE", "rep-app-1", "admin-1", ReviewInput{Status: "APPROVED"})
	if err != nil {
		t.Fatalf("review representative: %v", err)
	}
	if store.approvedRepresentativeID != "rep-app-1" {
		t.Fatalf("expected representative approval side effect to run")
	}
	if store.user.Role != domain.RoleRepresentative {
		t.Fatalf("expected user role to become representative, got %s", store.user.Role)
	}
	if result.Status != domain.ApprovalStatusApproved {
		t.Fatalf("expected approved detail, got %s", result.Status)
	}
	if len(store.notifications) == 0 {
		t.Fatalf("expected applicant notification to be emitted")
	}
}

func TestReviewOrganizationApprovalAssignsExpectedRole(t *testing.T) {
	store := &fakeStore{
		user: &domain.User{ID: "user-1", Role: domain.RoleCitizen, ApprovalStatus: domain.ApprovalStatusPending},
		org:  &domain.OrganizationApplication{ID: "org-app-1", UserID: "user-1", Status: domain.ApprovalStatusPending, Kind: "NGO"},
	}
	svc := NewService(store)

	_, err := svc.Review("ORGANIZATION", "org-app-1", "admin-1", ReviewInput{Status: "APPROVED"})
	if err != nil {
		t.Fatalf("review organization: %v", err)
	}
	if store.approvedOrganizationID != "org-app-1" {
		t.Fatalf("expected organization approval side effect to run")
	}
	if store.approvedOrganizationRole != domain.RoleNGO {
		t.Fatalf("expected NGO role, got %s", store.approvedOrganizationRole)
	}
}

func TestReviewNeedsChangesRequiresNote(t *testing.T) {
	store := &fakeStore{
		user: &domain.User{ID: "user-1", Role: domain.RoleCitizen, ApprovalStatus: domain.ApprovalStatusPending},
		rep:  &domain.RepresentativeApplication{ID: "rep-app-1", UserID: "user-1", Status: domain.ApprovalStatusPending},
	}
	svc := NewService(store)

	_, err := svc.Review("REPRESENTATIVE", "rep-app-1", "admin-1", ReviewInput{Status: "NEEDS_CHANGES"})
	if err == nil {
		t.Fatalf("expected missing note to be rejected")
	}
	var appErr *AppError
	if !errors.As(err, &appErr) || appErr.Code != "REVIEW_NOTE_REQUIRED" {
		t.Fatalf("expected REVIEW_NOTE_REQUIRED, got %v", err)
	}
}
