package issues

import (
	"testing"

	"github.com/civicos/community-service/internal/domain"
	"gorm.io/gorm"
)

type fakeIssueRepo struct {
	items []domain.Issue
}

func (f *fakeIssueRepo) FindAll(communityID, status string) ([]domain.Issue, error) {
	var filtered []domain.Issue
	for _, item := range f.items {
		if communityID != "" && item.CommunityID != communityID {
			continue
		}
		if status != "" && string(item.Status) != status {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered, nil
}

func (f *fakeIssueRepo) FindByID(id string) (*domain.Issue, error) {
	for _, item := range f.items {
		if item.ID == id {
			copy := item
			return &copy, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeIssueRepo) Create(issue *domain.Issue) error {
	f.items = append(f.items, *issue)
	return nil
}

func (f *fakeIssueRepo) IncrementUpvote(id string) error {
	for i := range f.items {
		if f.items[i].ID == id {
			f.items[i].UpvoteCount++
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

func (f *fakeIssueRepo) UpdateStatus(id string, status domain.IssueStatus) error {
	for i := range f.items {
		if f.items[i].ID == id {
			f.items[i].Status = status
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

func TestCreateAndUpdateIssueFlow(t *testing.T) {
	repo := &fakeIssueRepo{}
	svc := NewService(repo)

	created, err := svc.Create(CreateInput{
		Title:       "Potholes on Main Road",
		Description: "The road is unsafe for pedestrians",
		Category:    domain.CategoryInfrastructure,
		CommunityID: "community-1",
	}, "user-1")
	if err != nil {
		t.Fatalf("create issue: %v", err)
	}
	if created.Status != domain.IssueStatusOpen {
		t.Fatalf("expected issue status OPEN, got %s", created.Status)
	}

	if err := svc.Upvote(created.ID); err != nil {
		t.Fatalf("upvote issue: %v", err)
	}
	if err := svc.UpdateStatus(created.ID, domain.IssueStatusInProgress); err != nil {
		t.Fatalf("update status: %v", err)
	}

	updated, err := svc.Get(created.ID)
	if err != nil {
		t.Fatalf("get issue: %v", err)
	}
	if updated.UpvoteCount != 1 {
		t.Fatalf("expected one upvote, got %d", updated.UpvoteCount)
	}
	if updated.Status != domain.IssueStatusInProgress {
		t.Fatalf("expected status IN_PROGRESS, got %s", updated.Status)
	}
}
