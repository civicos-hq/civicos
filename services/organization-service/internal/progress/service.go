package progress

import (
	"errors"
	"net/http"

	"github.com/civicos/organization-service/internal/domain"
	"github.com/civicos/organization-service/internal/organizations"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Store interface {
	Find(f ListFilters) ([]domain.ProgressUpdate, error)
	FindByID(id string) (*domain.ProgressUpdate, error)
	Create(p *domain.ProgressUpdate) error
	Delete(id string) error
}

type Service struct {
	repo Store
	orgs *organizations.Service
}

func NewService(repo Store, orgs *organizations.Service) *Service {
	return &Service{repo: repo, orgs: orgs}
}

type CreateInput struct {
	IssueID   *string `json:"issueId"`
	ProjectID *string `json:"projectId"`
	Body      string  `json:"body" binding:"required,min=2"`
	IsPublic  *bool   `json:"isPublic"`
}

func (s *Service) List(f ListFilters) ([]domain.ProgressUpdate, error) {
	return s.repo.Find(f)
}

func (s *Service) Get(id string) (*domain.ProgressUpdate, error) {
	p, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &AppError{Code: "UPDATE_NOT_FOUND", Message: "Progress update not found", Status: http.StatusNotFound}
	}
	return p, err
}

func (s *Service) Create(orgID string, input CreateInput, authorID, authorName string) (*domain.ProgressUpdate, error) {
	// Exactly one of {issueId, projectId} must be set — an update always
	// hangs off either an assigned report or a project it's reporting on.
	if (input.IssueID == nil) == (input.ProjectID == nil) {
		return nil, &AppError{Code: "INVALID_TARGET", Message: "Set exactly one of issueId or projectId", Status: http.StatusBadRequest}
	}
	public := true
	if input.IsPublic != nil {
		public = *input.IsPublic
	}
	p := &domain.ProgressUpdate{
		ID:             uuid.New().String(),
		OrganizationID: orgID,
		IssueID:        input.IssueID,
		ProjectID:      input.ProjectID,
		Body:           input.Body,
		IsPublic:       public,
		AuthorID:       authorID,
		AuthorName:     authorName,
	}
	if err := s.repo.Create(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) Delete(id string) error {
	if _, err := s.Get(id); err != nil {
		return err
	}
	return s.repo.Delete(id)
}

type AppError struct {
	Code    string
	Message string
	Status  int
}

func (e *AppError) Error() string { return e.Message }
