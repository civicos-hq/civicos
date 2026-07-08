package flags

import (
	"errors"
	"testing"

	"github.com/civicos/identity-service/internal/audit"
	"github.com/civicos/identity-service/internal/domain"
	"gorm.io/gorm"
)

type fakeStore struct {
	flag    *domain.ContentFlag
	created *domain.ContentFlag
	updated map[string]any
}

func (f *fakeStore) Find(ListFilters) ([]domain.ContentFlag, error) {
	return nil, nil
}

func (f *fakeStore) FindByID(id string) (*domain.ContentFlag, error) {
	if f.flag != nil && f.flag.ID == id {
		return f.flag, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeStore) Create(flag *domain.ContentFlag) error {
	f.created = flag
	return nil
}

func (f *fakeStore) Update(id string, updates map[string]any) error {
	if f.flag == nil || f.flag.ID != id {
		return gorm.ErrRecordNotFound
	}
	f.updated = updates
	return nil
}

func (f *fakeStore) CountByStatus() (map[string]int64, error) {
	return map[string]int64{}, nil
}

func TestResolveHiddenRequiresResolutionNote(t *testing.T) {
	store := &fakeStore{
		flag: &domain.ContentFlag{ID: "flag-1", Status: domain.FlagStatusPending},
	}
	svc := NewService(store, nil)

	_, err := svc.Resolve("flag-1", ResolveInput{Status: "HIDDEN"}, audit.Actor{ID: "admin-1", Name: "Admin"})
	if err == nil {
		t.Fatalf("expected note requirement error")
	}
	var appErr *AppError
	if !errors.As(err, &appErr) || appErr.Code != "RESOLUTION_NOTE_REQUIRED" {
		t.Fatalf("expected RESOLUTION_NOTE_REQUIRED, got %v", err)
	}
}

func TestDirectHideRequiresResolutionNote(t *testing.T) {
	store := &fakeStore{}
	svc := NewService(store, nil)

	_, err := svc.DirectHide(DirectHideInput{
		ContentType: "ISSUE_COMMENT",
		ContentID:   "3fa85f64-5717-4562-b3fc-2c963f66afa6",
		Reason:      "ABUSE",
	}, domain.User{ID: "admin-1", Name: "Admin"})
	if err == nil {
		t.Fatalf("expected note requirement error")
	}
	var appErr *AppError
	if !errors.As(err, &appErr) || appErr.Code != "RESOLUTION_NOTE_REQUIRED" {
		t.Fatalf("expected RESOLUTION_NOTE_REQUIRED, got %v", err)
	}
}
