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
	Update(id string, updates map[string]any) error
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

// UpdateInput is the admin-editable surface of a community.
//
// Coordinates are the reason this exists: flood forecasts arrive as
// lat/lng and a community stored only as "Lagos / Ikeja" cannot be matched
// to one. There was no update endpoint at all before — communities were
// create-only — so an admin had no way to supply them.
type UpdateInput struct {
	Description *string `json:"description"`
	LogoURL     *string `json:"logoUrl"`
	// Latitude and Longitude must be sent together. Sending one alone is
	// rejected rather than stored, because half a coordinate is not a
	// location and silently keeping the stale other half would place the
	// community somewhere nobody chose.
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
	// ClearCoordinates removes the point. Distinguishable from "not
	// supplied" only by an explicit flag, since a nil pointer already means
	// "leave alone" for every other field here.
	ClearCoordinates bool `json:"clearCoordinates"`
}

func (s *Service) Update(id string, in UpdateInput) (*domain.Community, error) {
	if _, err := s.Get(id); err != nil {
		return nil, err
	}

	updates := map[string]any{}
	if in.Description != nil {
		updates["description"] = *in.Description
	}
	if in.LogoURL != nil {
		updates["logo_url"] = *in.LogoURL
	}

	switch {
	case in.ClearCoordinates:
		updates["latitude"] = nil
		updates["longitude"] = nil
	case in.Latitude != nil || in.Longitude != nil:
		if in.Latitude == nil || in.Longitude == nil {
			return nil, &AppError{
				Code:    "INCOMPLETE_COORDINATES",
				Message: "Latitude and longitude must be provided together",
				Status:  http.StatusBadRequest,
			}
		}
		if *in.Latitude < -90 || *in.Latitude > 90 {
			return nil, &AppError{Code: "INVALID_LATITUDE", Message: "Latitude must be between -90 and 90", Status: http.StatusBadRequest}
		}
		if *in.Longitude < -180 || *in.Longitude > 180 {
			return nil, &AppError{Code: "INVALID_LONGITUDE", Message: "Longitude must be between -180 and 180", Status: http.StatusBadRequest}
		}
		// 0,0 is in the Gulf of Guinea. It is what an uninitialised pair of
		// floats looks like, and for a feature that decides who gets a
		// flood warning, accepting it would put a community in the ocean.
		if *in.Latitude == 0 && *in.Longitude == 0 {
			return nil, &AppError{
				Code:    "INVALID_COORDINATES",
				Message: "0, 0 is not a valid location. Leave coordinates unset if you do not have them.",
				Status:  http.StatusBadRequest,
			}
		}
		updates["latitude"] = *in.Latitude
		updates["longitude"] = *in.Longitude
	}

	if len(updates) > 0 {
		if err := s.repo.Update(id, updates); err != nil {
			return nil, err
		}
	}
	return s.Get(id)
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
