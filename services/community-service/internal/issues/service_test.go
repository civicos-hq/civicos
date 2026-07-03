package issues

import (
	"testing"

	"github.com/civicos/community-service/internal/domain"
	"gorm.io/gorm"
)

type fakeIssueRepo struct {
	items    []domain.Issue
	comments map[string][]domain.IssueComment
	upvotes  map[string]map[string]bool // issueID -> userID -> present
}

func (f *fakeIssueRepo) HasUserUpvoted(issueID, userID string) (bool, error) {
	if f.upvotes == nil {
		return false, nil
	}
	return f.upvotes[issueID][userID], nil
}

func (f *fakeIssueRepo) AddUpvote(issueID, userID string) (int, error) {
	if f.upvotes == nil {
		f.upvotes = map[string]map[string]bool{}
	}
	if f.upvotes[issueID] == nil {
		f.upvotes[issueID] = map[string]bool{}
	}
	if !f.upvotes[issueID][userID] {
		f.upvotes[issueID][userID] = true
		for i := range f.items {
			if f.items[i].ID == issueID {
				f.items[i].UpvoteCount++
				return f.items[i].UpvoteCount, nil
			}
		}
	}
	for _, it := range f.items {
		if it.ID == issueID {
			return it.UpvoteCount, nil
		}
	}
	return 0, gorm.ErrRecordNotFound
}

func (f *fakeIssueRepo) RemoveUpvote(issueID, userID string) (int, error) {
	if f.upvotes != nil && f.upvotes[issueID] != nil && f.upvotes[issueID][userID] {
		delete(f.upvotes[issueID], userID)
		for i := range f.items {
			if f.items[i].ID == issueID && f.items[i].UpvoteCount > 0 {
				f.items[i].UpvoteCount--
				return f.items[i].UpvoteCount, nil
			}
		}
	}
	for _, it := range f.items {
		if it.ID == issueID {
			return it.UpvoteCount, nil
		}
	}
	return 0, gorm.ErrRecordNotFound
}

func (f *fakeIssueRepo) ListUpvotedIssueIDsByUser(userID string) ([]string, error) {
	var ids []string
	for issueID, voters := range f.upvotes {
		if voters[userID] {
			ids = append(ids, issueID)
		}
	}
	return ids, nil
}

func (f *fakeIssueRepo) FindAll(communityID, status, category string) ([]domain.Issue, error) {
	var filtered []domain.Issue
	for _, item := range f.items {
		if communityID != "" && item.CommunityID != communityID {
			continue
		}
		if status != "" && string(item.Status) != status {
			continue
		}
		if category != "" && string(item.Category) != category {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered, nil
}

func (f *fakeIssueRepo) ListComments(issueID string) ([]domain.IssueComment, error) {
	return f.comments[issueID], nil
}

func (f *fakeIssueRepo) AddComment(comment *domain.IssueComment) error {
	if f.comments == nil {
		f.comments = map[string][]domain.IssueComment{}
	}
	f.comments[comment.IssueID] = append(f.comments[comment.IssueID], *comment)
	for i := range f.items {
		if f.items[i].ID == comment.IssueID {
			f.items[i].CommentCount++
			break
		}
	}
	return nil
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

	// First click: adds an upvote.
	upvoted, count, err := svc.ToggleUpvote(created.ID, "user-1")
	if err != nil {
		t.Fatalf("first upvote: %v", err)
	}
	if !upvoted || count != 1 {
		t.Fatalf("expected upvoted=true count=1, got upvoted=%v count=%d", upvoted, count)
	}

	// Second click from the same user: this is where the regression lived —
	// the old service kept incrementing. Must now toggle back to 0.
	upvoted, count, err = svc.ToggleUpvote(created.ID, "user-1")
	if err != nil {
		t.Fatalf("second upvote (same user): %v", err)
	}
	if upvoted || count != 0 {
		t.Fatalf("expected toggle-off: upvoted=false count=0, got upvoted=%v count=%d", upvoted, count)
	}

	// Two different users each get one vote — final count is 2.
	if _, _, err := svc.ToggleUpvote(created.ID, "user-1"); err != nil {
		t.Fatalf("re-upvote user-1: %v", err)
	}
	_, count, err = svc.ToggleUpvote(created.ID, "user-2")
	if err != nil {
		t.Fatalf("upvote user-2: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected count=2 with two distinct voters, got %d", count)
	}

	ids, err := svc.ListUpvotedIssueIDs("user-1")
	if err != nil {
		t.Fatalf("list upvoted: %v", err)
	}
	if len(ids) != 1 || ids[0] != created.ID {
		t.Fatalf("expected user-1 to have exactly one upvoted issue, got %v", ids)
	}

	if err := svc.UpdateStatus(created.ID, domain.IssueStatusInProgress); err != nil {
		t.Fatalf("update status: %v", err)
	}

	updated, err := svc.Get(created.ID)
	if err != nil {
		t.Fatalf("get issue: %v", err)
	}
	if updated.Status != domain.IssueStatusInProgress {
		t.Fatalf("expected status IN_PROGRESS, got %s", updated.Status)
	}
}
