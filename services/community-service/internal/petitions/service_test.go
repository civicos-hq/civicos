package petitions

import (
	"testing"

	"github.com/civicos/community-service/internal/domain"
	"gorm.io/gorm"
)

type fakePetitionRepo struct {
	items []domain.Petition
	sigs  map[string]map[string]bool
}

func newFakePetitionRepo() *fakePetitionRepo {
	return &fakePetitionRepo{items: []domain.Petition{}, sigs: map[string]map[string]bool{}}
}

func (f *fakePetitionRepo) FindAll(communityID, status string) ([]domain.Petition, error) {
	var out []domain.Petition
	for _, p := range f.items {
		if communityID != "" && p.CommunityID != communityID {
			continue
		}
		if status != "" && string(p.Status) != status {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

func (f *fakePetitionRepo) FindByID(id string) (*domain.Petition, error) {
	for _, p := range f.items {
		if p.ID == id {
			copy := p
			return &copy, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakePetitionRepo) Create(p *domain.Petition) error {
	f.items = append(f.items, *p)
	return nil
}

func (f *fakePetitionRepo) AddSignature(petitionID, userID string) error {
	if f.sigs[petitionID] == nil {
		f.sigs[petitionID] = map[string]bool{}
	}
	if f.sigs[petitionID][userID] {
		return nil
	}
	f.sigs[petitionID][userID] = true
	// increment count
	for i := range f.items {
		if f.items[i].ID == petitionID {
			f.items[i].SignatureCount++
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

func TestCreateAndSignPetition(t *testing.T) {
	repo := newFakePetitionRepo()
	svc := NewService(repo)

	created, err := svc.Create(CreateInput{
		Title:       "Save the Park",
		Description: "We need to preserve green space",
		Goal:        100,
		CommunityID: "community-1",
	}, "user-1")
	if err != nil {
		t.Fatalf("create petition: %v", err)
	}
	if created.Title != "Save the Park" {
		t.Fatalf("unexpected title: %s", created.Title)
	}

	if err := svc.Sign(created.ID, "user-2"); err != nil {
		t.Fatalf("sign petition: %v", err)
	}

	p, err := svc.Get(created.ID)
	if err != nil {
		t.Fatalf("get petition: %v", err)
	}
	if p.SignatureCount != 1 {
		t.Fatalf("expected 1 signature, got %d", p.SignatureCount)
	}
}
