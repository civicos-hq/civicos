package issues

import (
	"errors"
	"net/http"

	"github.com/civicos/community-service/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IssueStore interface {
	FindAll(communityID, status, category string) ([]domain.Issue, error)
	FindByID(id string) (*domain.Issue, error)
	Create(issue *domain.Issue) error
	HasUserUpvoted(issueID, userID string) (bool, error)
	AddUpvote(issueID, userID string) (int, error)
	RemoveUpvote(issueID, userID string) (int, error)
	ListUpvotedIssueIDsByUser(userID string) ([]string, error)
	UpdateStatus(id string, status domain.IssueStatus) error
	ListComments(issueID string) ([]domain.IssueComment, error)
	AddComment(comment *domain.IssueComment) error
}

// OfficialRoles is the set of roles whose comments are flagged as official responses.
var OfficialRoles = map[string]bool{
	"REPRESENTATIVE":   true,
	"GOVERNMENT_ADMIN": true,
	"PLATFORM_ADMIN":   true,
	"NGO":              true,
	"MODERATOR":        true,
}

type Service struct{ repo IssueStore }

func NewService(repo IssueStore) *Service { return &Service{repo: repo} }

type CreateInput struct {
	Title       string               `json:"title" binding:"required,min=5"`
	Description string               `json:"description" binding:"required,min=10"`
	Category    domain.IssueCategory `json:"category" binding:"required"`
	CommunityID string               `json:"communityId" binding:"required"`
	Location    *string              `json:"location"`
	ImageURLs   []string             `json:"imageUrls"`
}

type AppError struct {
	Code    string
	Message string
	Status  int
}

func (e *AppError) Error() string { return e.Message }

func (s *Service) List(communityID, status, category string) ([]domain.Issue, error) {
	return s.repo.FindAll(communityID, status, category)
}

func (s *Service) Get(id string) (*domain.Issue, error) {
	issue, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &AppError{Code: "ISSUE_NOT_FOUND", Message: "Issue not found", Status: http.StatusNotFound}
	}
	return issue, err
}

func (s *Service) Create(input CreateInput, reportedByID string) (*domain.Issue, error) {
	images := input.ImageURLs
	if images == nil {
		images = []string{}
	}
	issue := &domain.Issue{
		ID:           uuid.New().String(),
		Title:        input.Title,
		Description:  input.Description,
		Category:     input.Category,
		Status:       domain.IssueStatusOpen,
		Location:     input.Location,
		ImageURLs:    images,
		CommunityID:  input.CommunityID,
		ReportedByID: reportedByID,
	}
	return issue, s.repo.Create(issue)
}

// ToggleUpvote flips the caller's upvote on the issue. Returns the state
// they should now see (upvoted true/false) and the fresh counter. Prior
// behaviour blindly incremented per click — this dedups via the unique
// (issue_id, user_id) constraint on IssueUpvote.
func (s *Service) ToggleUpvote(issueID, userID string) (upvoted bool, count int, err error) {
	if _, err := s.Get(issueID); err != nil {
		return false, 0, err
	}
	has, err := s.repo.HasUserUpvoted(issueID, userID)
	if err != nil {
		return false, 0, err
	}
	if has {
		n, err := s.repo.RemoveUpvote(issueID, userID)
		return false, n, err
	}
	n, err := s.repo.AddUpvote(issueID, userID)
	return true, n, err
}

func (s *Service) ListUpvotedIssueIDs(userID string) ([]string, error) {
	return s.repo.ListUpvotedIssueIDsByUser(userID)
}

func (s *Service) UpdateStatus(id string, status domain.IssueStatus) error {
	return s.repo.UpdateStatus(id, status)
}

func (s *Service) ListComments(issueID string) ([]domain.IssueComment, error) {
	return s.repo.ListComments(issueID)
}

type CommentInput struct {
	Content string `json:"content" binding:"required,min=1,max=2000"`
}

func (s *Service) AddComment(issueID, authorID, authorName, authorRole, content string) (*domain.IssueComment, error) {
	comment := &domain.IssueComment{
		ID:                 uuid.New().String(),
		Content:            content,
		IssueID:            issueID,
		AuthorID:           authorID,
		AuthorName:         authorName,
		AuthorRole:         authorRole,
		IsOfficialResponse: OfficialRoles[authorRole],
	}
	return comment, s.repo.AddComment(comment)
}
