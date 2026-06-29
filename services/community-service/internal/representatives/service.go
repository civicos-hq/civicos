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
	Update(id string, updates map[string]any) error
	AddFollow(repID, userID string) error
	RemoveFollow(repID, userID string) error
	FindFollowedIDs(userID string) ([]string, error)
	FindFollowerIDs(repID string) ([]string, error)
	ListComments(repID string) ([]domain.RepresentativeComment, error)
	AddComment(comment *domain.RepresentativeComment) error
}

// OfficialRoles is the set of roles whose comments are flagged as official
// responses on a representative profile.
var OfficialRoles = map[string]bool{
	"REPRESENTATIVE":   true,
	"GOVERNMENT_ADMIN": true,
	"PLATFORM_ADMIN":   true,
	"NGO":              true,
	"MODERATOR":        true,
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
	Email        *string `json:"email" binding:"omitempty,email"`
	Phone        *string `json:"phone"`
	Website      *string `json:"website" binding:"omitempty,url"`
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
		Email:        input.Email,
		Phone:        input.Phone,
		Website:      input.Website,
		CommunityID:  input.CommunityID,
		CreatedByID:  createdByID,
	}
	return rep, s.repo.Create(rep)
}

type CommentInput struct {
	Content string `json:"content" binding:"required,min=1,max=2000"`
}

// UpdateInput uses pointer fields so the handler can distinguish "leave
// unchanged" (nil) from "clear it" (pointer to empty string).
type UpdateInput struct {
	Name         *string `json:"name" binding:"omitempty,min=2"`
	Title        *string `json:"title"`
	Position     *string `json:"position"`
	Constituency *string `json:"constituency"`
	Party        *string `json:"party"`
	Bio          *string `json:"bio"`
	AvatarURL    *string `json:"avatarUrl"`
	Email        *string `json:"email" binding:"omitempty,email"`
	Phone        *string `json:"phone"`
	Website      *string `json:"website" binding:"omitempty,url"`
}

func (s *Service) Update(id string, input UpdateInput) (*domain.Representative, error) {
	updates := map[string]any{}
	if input.Name != nil {
		updates["name"] = *input.Name
	}
	if input.Title != nil {
		updates["title"] = *input.Title
	}
	if input.Position != nil {
		updates["position"] = *input.Position
	}
	if input.Constituency != nil {
		updates["constituency"] = *input.Constituency
	}
	// For nullable string columns we want to allow clearing (empty string → NULL).
	if input.Party != nil {
		updates["party"] = nilIfEmpty(*input.Party)
	}
	if input.Bio != nil {
		updates["bio"] = nilIfEmpty(*input.Bio)
	}
	if input.AvatarURL != nil {
		updates["avatar_url"] = nilIfEmpty(*input.AvatarURL)
	}
	if input.Email != nil {
		updates["email"] = nilIfEmpty(*input.Email)
	}
	if input.Phone != nil {
		updates["phone"] = nilIfEmpty(*input.Phone)
	}
	if input.Website != nil {
		updates["website"] = nilIfEmpty(*input.Website)
	}
	if len(updates) > 0 {
		if err := s.repo.Update(id, updates); err != nil {
			return nil, err
		}
	}
	return s.Get(id)
}

func nilIfEmpty(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func (s *Service) ListComments(repID string) ([]domain.RepresentativeComment, error) {
	return s.repo.ListComments(repID)
}

func (s *Service) AddComment(repID, authorID, authorName, authorRole, content string) (*domain.RepresentativeComment, error) {
	comment := &domain.RepresentativeComment{
		ID:                 uuid.New().String(),
		Content:            content,
		RepresentativeID:   repID,
		AuthorID:           authorID,
		AuthorName:         authorName,
		AuthorRole:         authorRole,
		IsOfficialResponse: OfficialRoles[authorRole],
	}
	return comment, s.repo.AddComment(comment)
}

func (s *Service) FollowerIDs(repID string) ([]string, error) {
	ids, err := s.repo.FindFollowerIDs(repID)
	if err != nil {
		return nil, err
	}
	if ids == nil {
		ids = []string{}
	}
	return ids, nil
}
