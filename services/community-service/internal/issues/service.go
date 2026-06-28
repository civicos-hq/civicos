package issues

import (
	"errors"
	"net/http"

	"github.com/civicos/community-service/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service struct{ repo *Repository }

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

type CreateInput struct {
	Title       string              `json:"title" binding:"required,min=5"`
	Description string              `json:"description" binding:"required,min=10"`
	Category    domain.IssueCategory `json:"category" binding:"required"`
	CommunityID string              `json:"communityId" binding:"required"`
	Location    *string             `json:"location"`
}

type AppError struct {
	Code    string
	Message string
	Status  int
}
func (e *AppError) Error() string { return e.Message }

func (s *Service) List(communityID, status string) ([]domain.Issue, error) {
	return s.repo.FindAll(communityID, status)
}

func (s *Service) Get(id string) (*domain.Issue, error) {
	issue, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &AppError{Code: "ISSUE_NOT_FOUND", Message: "Issue not found", Status: http.StatusNotFound}
	}
	return issue, err
}

func (s *Service) Create(input CreateInput, reportedByID string) (*domain.Issue, error) {
	issue := &domain.Issue{
		ID:           uuid.New().String(),
		Title:        input.Title,
		Description:  input.Description,
		Category:     input.Category,
		Status:       domain.IssueStatusOpen,
		Location:     input.Location,
		ImageURLs:    []string{},
		CommunityID:  input.CommunityID,
		ReportedByID: reportedByID,
	}
	return issue, s.repo.Create(issue)
}

func (s *Service) Upvote(id string) error {
	return s.repo.IncrementUpvote(id)
}

func (s *Service) UpdateStatus(id string, status domain.IssueStatus) error {
	return s.repo.UpdateStatus(id, status)
}
