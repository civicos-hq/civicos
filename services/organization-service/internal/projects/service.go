package projects

import (
	"errors"
	"net/http"
	"time"

	"github.com/civicos/organization-service/internal/domain"
	"github.com/civicos/organization-service/internal/organizations"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Store interface {
	Find(f ListFilters) ([]domain.Project, error)
	FindByID(id string) (*domain.Project, error)
	Create(p *domain.Project) error
	Update(id string, updates map[string]any) error
	Delete(id string) error
	BumpOrgCount(orgID string, delta int) error
}

type Service struct {
	repo Store
	orgs *organizations.Service
}

func NewService(repo Store, orgs *organizations.Service) *Service {
	return &Service{repo: repo, orgs: orgs}
}

type CreateInput struct {
	Title           string     `json:"title" binding:"required,min=2"`
	Description     string     `json:"description" binding:"required,min=10"`
	Status          string     `json:"status"`
	StartDate       *time.Time `json:"startDate"`
	ExpectedEndDate *time.Time `json:"expectedEndDate"`
	BudgetKobo      *int64     `json:"budgetKobo"`
	CommunityID     *string    `json:"communityId"`
}

type UpdateInput struct {
	Title           *string    `json:"title"`
	Description     *string    `json:"description"`
	Status          *string    `json:"status"`
	StartDate       *time.Time `json:"startDate"`
	ExpectedEndDate *time.Time `json:"expectedEndDate"`
	BudgetKobo      *int64     `json:"budgetKobo"`
	CommunityID     *string    `json:"communityId"`
}

func (s *Service) List(f ListFilters) ([]domain.Project, error) {
	return s.repo.Find(f)
}

func (s *Service) Get(id string) (*domain.Project, error) {
	p, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &AppError{Code: "PROJECT_NOT_FOUND", Message: "Project not found", Status: http.StatusNotFound}
	}
	return p, err
}

func (s *Service) Create(orgID string, input CreateInput, createdByID string) (*domain.Project, error) {
	status := domain.ProjectPlanned
	if input.Status != "" {
		if !validStatus(input.Status) {
			return nil, &AppError{Code: "INVALID_STATUS", Message: "Unknown project status", Status: http.StatusBadRequest}
		}
		status = domain.ProjectStatus(input.Status)
	}
	p := &domain.Project{
		ID:              uuid.New().String(),
		OrganizationID:  orgID,
		Title:           input.Title,
		Description:     input.Description,
		Status:          status,
		StartDate:       input.StartDate,
		ExpectedEndDate: input.ExpectedEndDate,
		BudgetKobo:      input.BudgetKobo,
		CommunityID:     input.CommunityID,
		CreatedByID:     createdByID,
	}
	if err := s.repo.Create(p); err != nil {
		return nil, err
	}
	_ = s.repo.BumpOrgCount(orgID, 1)
	return p, nil
}

func (s *Service) Update(id string, input UpdateInput) (*domain.Project, error) {
	updates := map[string]any{}
	if input.Title != nil {
		updates["title"] = *input.Title
	}
	if input.Description != nil {
		updates["description"] = *input.Description
	}
	if input.Status != nil {
		if !validStatus(*input.Status) {
			return nil, &AppError{Code: "INVALID_STATUS", Message: "Unknown project status", Status: http.StatusBadRequest}
		}
		updates["status"] = *input.Status
	}
	if input.StartDate != nil {
		updates["start_date"] = *input.StartDate
	}
	if input.ExpectedEndDate != nil {
		updates["expected_end_date"] = *input.ExpectedEndDate
	}
	if input.BudgetKobo != nil {
		updates["budget_kobo"] = *input.BudgetKobo
	}
	if input.CommunityID != nil {
		updates["community_id"] = *input.CommunityID
	}
	if len(updates) == 0 {
		return s.Get(id)
	}
	if err := s.repo.Update(id, updates); err != nil {
		return nil, err
	}
	return s.Get(id)
}

func (s *Service) Delete(id string) error {
	p, err := s.Get(id)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(id); err != nil {
		return err
	}
	_ = s.repo.BumpOrgCount(p.OrganizationID, -1)
	return nil
}

func validStatus(s string) bool {
	switch domain.ProjectStatus(s) {
	case domain.ProjectPlanned, domain.ProjectActive, domain.ProjectPaused,
		domain.ProjectCompleted, domain.ProjectCancelled:
		return true
	}
	return false
}

type AppError struct {
	Code    string
	Message string
	Status  int
}

func (e *AppError) Error() string { return e.Message }
