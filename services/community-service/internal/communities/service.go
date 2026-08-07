package communities

import (
	"errors"
	"net/http"

	"github.com/civicos/community-service/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CommunityStore interface {
	Search(p SearchParams) ([]domain.Community, int64, error)
	FindByID(id string) (*domain.Community, error)
	FindByIDs(ids []string) ([]domain.Community, error)
	Create(c *domain.Community) error
}

// Page bounds for the community list. The browse page used to load every
// community in one unpaginated response; that was survivable at a couple of
// dozen rows and stops being so once every university is seeded.
const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

type Service struct{ repo CommunityStore }

func NewService(repo CommunityStore) *Service { return &Service{repo: repo} }

type CreateInput struct {
	Name        string  `json:"name" binding:"required,min=2"`
	Slug        string  `json:"slug" binding:"required,min=2"`
	State       string  `json:"state" binding:"required"`
	LGA         string  `json:"lga" binding:"required"`
	Description *string `json:"description"`
}

// ListResult carries the page plus enough metadata for the caller to page
// through it without a second "how many are there" request.
type ListResult struct {
	Communities []domain.Community `json:"communities"`
	Total       int64              `json:"total"`
	Limit       int                `json:"limit"`
	Offset      int                `json:"offset"`
}

func (s *Service) List(p SearchParams) (*ListResult, error) {
	if p.Limit <= 0 {
		p.Limit = DefaultPageSize
	}
	if p.Limit > MaxPageSize {
		p.Limit = MaxPageSize
	}
	if p.Offset < 0 {
		p.Offset = 0
	}

	list, total, err := s.repo.Search(p)
	if err != nil {
		return nil, err
	}
	return &ListResult{Communities: list, Total: total, Limit: p.Limit, Offset: p.Offset}, nil
}

// Resolve loads a set of communities by ID and fails if any is unknown, so a
// batch join either validates completely or writes nothing.
func (s *Service) Resolve(ids []string) ([]domain.Community, error) {
	found, err := s.repo.FindByIDs(ids)
	if err != nil {
		return nil, err
	}
	if len(found) != len(ids) {
		return nil, &AppError{
			Code:    "COMMUNITY_NOT_FOUND",
			Message: "One or more communities do not exist",
			Status:  http.StatusNotFound,
		}
	}
	return found, nil
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
