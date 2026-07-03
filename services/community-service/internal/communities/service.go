package communities

import (
	"errors"
	"net/http"

	"github.com/civicos/community-service/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CommunityStore interface {
	FindAll() ([]domain.Community, error)
	FindByLocation(state, lga string) ([]domain.Community, error)
	FindByID(id string) (*domain.Community, error)
	Create(c *domain.Community) error
}

type Service struct{ repo CommunityStore }

func NewService(repo CommunityStore) *Service { return &Service{repo: repo} }

type CreateInput struct {
	Name        string  `json:"name" binding:"required,min=2"`
	Slug        string  `json:"slug" binding:"required,min=2"`
	State       string  `json:"state" binding:"required"`
	LGA         string  `json:"lga" binding:"required"`
	Description *string `json:"description"`
}

func (s *Service) List(state, lga string) ([]domain.Community, error) {
	if state == "" && lga == "" {
		return s.repo.FindAll()
	}
	return s.repo.FindByLocation(state, lga)
}

func (s *Service) Get(id string) (*domain.Community, error) {
	c, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &AppError{Code: "COMMUNITY_NOT_FOUND", Message: "Community not found", Status: http.StatusNotFound}
	}
	return c, err
}

func (s *Service) Create(input CreateInput, createdByID string) (*domain.Community, error) {
	c := &domain.Community{
		ID:          uuid.New().String(),
		Name:        input.Name,
		Slug:        input.Slug,
		State:       input.State,
		LGA:         input.LGA,
		Description: input.Description,
		CreatedByID: createdByID,
	}
	return c, s.repo.Create(c)
}

type AppError struct {
	Code    string
	Message string
	Status  int
}

func (e *AppError) Error() string { return e.Message }
