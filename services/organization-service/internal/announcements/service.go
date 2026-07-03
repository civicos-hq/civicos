package announcements

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
	FindByOrg(orgID string, includeDrafts bool) ([]domain.Announcement, error)
	FindPublished(limit int) ([]domain.Announcement, error)
	FindByID(id string) (*domain.Announcement, error)
	Create(a *domain.Announcement) error
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
	Title   string `json:"title" binding:"required,min=2"`
	Body    string `json:"body" binding:"required,min=10"`
	Publish bool   `json:"publish"`
}

type UpdateInput struct {
	Title *string `json:"title"`
	Body  *string `json:"body"`
}

func (s *Service) ListByOrg(orgID string, includeDrafts bool) ([]domain.Announcement, error) {
	return s.repo.FindByOrg(orgID, includeDrafts)
}

func (s *Service) ListPublishedGlobal(limit int) ([]domain.Announcement, error) {
	return s.repo.FindPublished(limit)
}

func (s *Service) Get(id string) (*domain.Announcement, error) {
	a, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &AppError{Code: "ANNOUNCEMENT_NOT_FOUND", Message: "Announcement not found", Status: http.StatusNotFound}
	}
	return a, err
}

func (s *Service) Create(orgID string, input CreateInput, authorID, authorName string) (*domain.Announcement, error) {
	a := &domain.Announcement{
		ID:             uuid.New().String(),
		OrganizationID: orgID,
		Title:          input.Title,
		Body:           input.Body,
		Status:         domain.AnnouncementDraft,
		AuthorID:       authorID,
		AuthorName:     authorName,
	}
	if input.Publish {
		now := time.Now().UTC()
		a.Status = domain.AnnouncementPublished
		a.PublishedAt = &now
	}
	if err := s.repo.Create(a); err != nil {
		return nil, err
	}
	if input.Publish {
		_ = s.repo.BumpOrgCount(orgID, 1)
	}
	return a, nil
}

func (s *Service) Update(id string, input UpdateInput) (*domain.Announcement, error) {
	updates := map[string]any{}
	if input.Title != nil {
		updates["title"] = *input.Title
	}
	if input.Body != nil {
		updates["body"] = *input.Body
	}
	if len(updates) == 0 {
		return s.Get(id)
	}
	if err := s.repo.Update(id, updates); err != nil {
		return nil, err
	}
	return s.Get(id)
}

func (s *Service) Publish(id string) (*domain.Announcement, error) {
	a, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if a.Status == domain.AnnouncementPublished {
		return a, nil
	}
	now := time.Now().UTC()
	if err := s.repo.Update(id, map[string]any{
		"status":       domain.AnnouncementPublished,
		"published_at": now,
	}); err != nil {
		return nil, err
	}
	_ = s.repo.BumpOrgCount(a.OrganizationID, 1)
	return s.Get(id)
}

func (s *Service) Archive(id string) (*domain.Announcement, error) {
	a, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Update(id, map[string]any{"status": domain.AnnouncementArchived}); err != nil {
		return nil, err
	}
	if a.Status == domain.AnnouncementPublished {
		_ = s.repo.BumpOrgCount(a.OrganizationID, -1)
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
	if a.Status == domain.AnnouncementPublished {
		_ = s.repo.BumpOrgCount(a.OrganizationID, -1)
	}
	return nil
}

type AppError struct {
	Code    string
	Message string
	Status  int
}

func (e *AppError) Error() string { return e.Message }
