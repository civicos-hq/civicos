package communities

import (
	"testing"

	"github.com/civicos/community-service/internal/domain"
	"gorm.io/gorm"
)

type fakeCommunityRepo struct {
	items []domain.Community
}

func (f *fakeCommunityRepo) FindAll() ([]domain.Community, error) {
	return append([]domain.Community{}, f.items...), nil
}

func (f *fakeCommunityRepo) FindByID(id string) (*domain.Community, error) {
	for _, item := range f.items {
		if item.ID == id {
			copy := item
			return &copy, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeCommunityRepo) Create(c *domain.Community) error {
	f.items = append(f.items, *c)
	return nil
}

func TestCreateAndListCommunities(t *testing.T) {
	repo := &fakeCommunityRepo{}
	svc := NewService(repo)
	desc := "A vibrant local community"

	created, err := svc.Create(CreateInput{
		Name:        "Lekki",
		Slug:        "lekki",
		State:       "Lagos",
		LGA:         "Eti-Osa",
		Description: &desc,
	}, "user-1")
	if err != nil {
		t.Fatalf("create community: %v", err)
	}
	if created.Slug != "lekki" {
		t.Fatalf("expected slug lekki, got %s", created.Slug)
	}

	items, err := svc.List()
	if err != nil {
		t.Fatalf("list communities: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 community, got %d", len(items))
	}

	got, err := svc.Get(created.ID)
	if err != nil {
		t.Fatalf("get community: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("expected community id %s, got %s", created.ID, got.ID)
	}
}
