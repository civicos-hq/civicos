package representatives

import (
	"errors"
	"net/http"

	"github.com/civicos/community-service/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RepresentativeStore interface {
	FindAll(communityID string) ([]domain.Representative, error)
	FindByID(id string) (*domain.Representative, error)
	Create(rep *domain.Representative) error
	AddFollow(repID, userID string) error
	RemoveFollow(repID, userID string) error
	FindFollowedIDs(userID string) ([]string, error)
}

type Service struct{ repo RepresentativeStore }

func NewService(repo RepresentativeStore) *Service { return &Service{repo: repo} }

type CreateInput struct {
	Name         string  `json:"name" binding:"required,min=2"`
	Title        string  `json:"title" binding:"required"`
	Position     string  `json:"position" binding:"required"`
	Constituency string  `json:"constituency" binding:"required"`
	CommunityID  string  `json:"communityId" binding:"required"`
	Party        *string `json:"party"`
	Bio          *string `json:"bio"`
	AvatarURL    *string `json:"avatarUrl"`
}

type AppError struct {
	Code    string
	Message string
	Status  int
}

func (e *AppError) Error() string { return e.Message }

func (s *Service) List(communityID string) ([]domain.Representative, error) {
	return s.repo.FindAll(communityID)
}

func (s *Service) Get(id string) (*domain.Representative, error) {
	rep, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &AppError{Code: "REPRESENTATIVE_NOT_FOUND", Message: "Representative not found", Status: http.StatusNotFound}
	}
	return rep, err
}

func (s *Service) Follow(repID, userID string) error {
	return s.repo.AddFollow(repID, userID)
}

func (s *Service) Unfollow(repID, userID string) error {
	return s.repo.RemoveFollow(repID, userID)
}

func (s *Service) FollowedIDs(userID string) ([]string, error) {
	ids, err := s.repo.FindFollowedIDs(userID)
	if err != nil {
		return nil, err
	}
	if ids == nil {
		ids = []string{}
	}
	return ids, nil
}

func (s *Service) Create(input CreateInput, createdByID string) (*domain.Representative, error) {
	rep := &domain.Representative{
		ID:           uuid.New().String(),
		Name:         input.Name,
		Title:        input.Title,
		Position:     input.Position,
		Constituency: input.Constituency,
		Party:        input.Party,
		Bio:          input.Bio,
		AvatarURL:    input.AvatarURL,
		CommunityID:  input.CommunityID,
		CreatedByID:  createdByID,
	}
	return rep, s.repo.Create(rep)
}
