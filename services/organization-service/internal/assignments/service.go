package assignments

import (
	"errors"
	"net/http"

	"github.com/civicos/organization-service/internal/domain"
	"github.com/civicos/organization-service/internal/organizations"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Store interface {
	FindByOrg(orgID, status string) ([]domain.IssueAssignment, error)
	FindByIssue(issueID string) ([]domain.IssueAssignment, error)
	FindByID(id string) (*domain.IssueAssignment, error)
	FindExact(orgID, issueID string) (*domain.IssueAssignment, error)
	Create(a *domain.IssueAssignment) error
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

type AssignInput struct {
	IssueID string  `json:"issueId" binding:"required"`
	Note    *string `json:"note"`
}

type StatusInput struct {
	Status string  `json:"status" binding:"required"`
	Note   *string `json:"note"`
}

func (s *Service) ListByOrg(orgID, status string) ([]domain.IssueAssignment, error) {
	return s.repo.FindByOrg(orgID, status)
}

func (s *Service) ListByIssue(issueID string) ([]domain.IssueAssignment, error) {
	return s.repo.FindByIssue(issueID)
}

func (s *Service) Get(id string) (*domain.IssueAssignment, error) {
	a, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &AppError{Code: "ASSIGNMENT_NOT_FOUND", Message: "Assignment not found", Status: http.StatusNotFound}
	}
	return a, err
}

func (s *Service) Create(orgID string, input AssignInput, assignedByID, assignedByName string) (*domain.IssueAssignment, error) {
	if _, err := s.repo.FindExact(orgID, input.IssueID); err == nil {
		return nil, &AppError{Code: "ALREADY_ASSIGNED", Message: "This issue is already assigned to this organization", Status: http.StatusConflict}
	}
	a := &domain.IssueAssignment{
		ID:             uuid.New().String(),
		OrganizationID: orgID,
		IssueID:        input.IssueID,
		Status:         domain.AssignmentReceived,
		Note:           input.Note,
		AssignedByID:   assignedByID,
		AssignedByName: assignedByName,
	}
	if err := s.repo.Create(a); err != nil {
		return nil, err
	}
	_ = s.repo.BumpOrgCount(orgID, 1)
	return a, nil
}

func (s *Service) UpdateStatus(id string, input StatusInput) (*domain.IssueAssignment, error) {
	if !validStatus(input.Status) {
		return nil, &AppError{Code: "INVALID_STATUS", Message: "Unknown assignment status", Status: http.StatusBadRequest}
	}
	updates := map[string]any{"status": input.Status}
	if input.Note != nil {
		updates["note"] = *input.Note
	}
	if err := s.repo.Update(id, updates); err != nil {
		return nil, err
	}
	return s.Get(id)
}

func (s *Service) Delete(id string) error {
	a, err := s.Get(id)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(id); err != nil {
		return err
	}
	_ = s.repo.BumpOrgCount(a.OrganizationID, -1)
	return nil
}

func validStatus(s string) bool {
	switch domain.AssignmentStatus(s) {
	case domain.AssignmentReceived, domain.AssignmentInProgress,
		domain.AssignmentCompleted, domain.AssignmentRejected:
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
