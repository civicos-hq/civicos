package petitions

import (
	"net/http"
	"time"

	"github.com/civicos/community-service/internal/domain"
	"github.com/google/uuid"
)

type PetitionStore interface {
	FindAll(communityID, status string) ([]domain.Petition, error)
	FindByID(id string) (*domain.Petition, error)
	Create(p *domain.Petition) error
	AddSignature(petitionID, userID string) error
}

type Service struct{ repo PetitionStore }

func NewService(repo PetitionStore) *Service { return &Service{repo: repo} }

type CreateInput struct {
	Title       string  `json:"title" binding:"required,min=5"`
	Description string  `json:"description" binding:"required,min=10"`
	Goal        int     `json:"goal" binding:"required,min=1"`
	Deadline    *string `json:"deadline"`
	CommunityID string  `json:"communityId" binding:"required"`
}

func (s *Service) List(communityID, status string) ([]domain.Petition, error) {
	return s.repo.FindAll(communityID, status)
}

func (s *Service) Get(id string) (*domain.Petition, error) {
	p, err := s.repo.FindByID(id)
	if err != nil {
		return nil, &AppError{Code: "PETITION_NOT_FOUND", Message: "Petition not found", Status: http.StatusNotFound}
	}
	return p, nil
}

func (s *Service) Create(input CreateInput, createdByID string) (*domain.Petition, error) {
	var deadlinePtr *time.Time
	if input.Deadline != nil {
		if t, err := time.Parse(time.RFC3339, *input.Deadline); err == nil {
			deadlinePtr = &t
		}
	}

	p := &domain.Petition{
		ID:          uuid.New().String(),
		Title:       input.Title,
		Description: input.Description,
		Goal:        input.Goal,
		Deadline:    deadlinePtr,
		CommunityID: input.CommunityID,
		CreatedByID: createdByID,
	}
	if err := s.repo.Create(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) Sign(petitionID, userID string) error {
	if err := s.repo.AddSignature(petitionID, userID); err != nil {
		return err
	}
	return nil
}

type AppError struct {
	Code    string
	Message string
	Status  int
}

func (e *AppError) Error() string { return e.Message }
