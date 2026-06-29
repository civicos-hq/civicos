package petitions

import (
	"testing"

	"github.com/civicos/community-service/internal/domain"
	"gorm.io/gorm"
)

type fakePetitionRepo struct {
	items    []domain.Petition
	sigs     map[string]map[string]bool
	comments map[string][]domain.PetitionComment
}

func newFakePetitionRepo() *fakePetitionRepo {
	return &fakePetitionRepo{
		items:    []domain.Petition{},
		sigs:     map[string]map[string]bool{},
		comments: map[string][]domain.PetitionComment{},
	}
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

func (f *fakePetitionRepo) AddSignature(petitionID, userID string) (bool, int, error) {
	if f.sigs[petitionID] == nil {
		f.sigs[petitionID] = map[string]bool{}
	}
	for i := range f.items {
		if f.items[i].ID != petitionID {
			continue
		}
		if f.sigs[petitionID][userID] {
			return false, f.items[i].SignatureCount, nil
		}
		f.sigs[petitionID][userID] = true
		f.items[i].SignatureCount++
		return true, f.items[i].SignatureCount, nil
	}
	return false, 0, gorm.ErrRecordNotFound
}

func (f *fakePetitionRepo) ListComments(petitionID string) ([]domain.PetitionComment, error) {
	return f.comments[petitionID], nil
}

func (f *fakePetitionRepo) AddComment(comment *domain.PetitionComment) error {
	f.comments[comment.PetitionID] = append(f.comments[comment.PetitionID], *comment)
	for i := range f.items {
		if f.items[i].ID == comment.PetitionID {
			f.items[i].CommentCount++
			break
		}
	}
	return nil
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

	res, err := svc.Sign(created.ID, "user-2")
	if err != nil {
		t.Fatalf("sign petition: %v", err)
	}
	if !res.Added || res.NewCount != 1 {
		t.Fatalf("expected added=true count=1, got added=%v count=%d", res.Added, res.NewCount)
	}

	p, err := svc.Get(created.ID)
	if err != nil {
		t.Fatalf("get petition: %v", err)
	}
	if p.SignatureCount != 1 {
		t.Fatalf("expected 1 signature, got %d", p.SignatureCount)
	}
}
