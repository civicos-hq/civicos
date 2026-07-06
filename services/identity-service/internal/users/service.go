package users

import (
	"errors"
	"net/http"
	"time"

	"github.com/civicos/identity-service/internal/domain"
	"gorm.io/gorm"
)

type Store interface {
	Find(f ListFilters) ([]domain.User, int64, error)
	FindByID(id string) (*domain.User, error)
	Update(id string, updates map[string]any) error
}

type Service struct{ repo Store }

func NewService(repo Store) *Service { return &Service{repo: repo} }

type RoleUpdateInput struct {
	Role string `json:"role" binding:"required"`
}

type BanInput struct {
	Reason *string `json:"reason"`
}

func (s *Service) List(f ListFilters) ([]domain.User, int64, error) {
	return s.repo.Find(f)
}

func (s *Service) Get(id string) (*domain.User, error) {
	u, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &AppError{Code: "USER_NOT_FOUND", Message: "User not found", Status: http.StatusNotFound}
	}
	return u, err
}

func (s *Service) ChangeRole(id string, input RoleUpdateInput) (*domain.User, error) {
	if !validRole(input.Role) {
		return nil, &AppError{Code: "INVALID_ROLE", Message: "Unknown user role", Status: http.StatusBadRequest}
	}
	if err := s.repo.Update(id, map[string]any{"role": input.Role}); err != nil {
		return nil, err
	}
	return s.Get(id)
}

func (s *Service) Ban(id string, input BanInput, actorID string) (*domain.User, error) {
	u, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if u.BannedAt != nil {
		return u, nil // idempotent
	}
	if u.ID == actorID {
		return nil, &AppError{Code: "CANNOT_SELF_BAN", Message: "You cannot ban yourself", Status: http.StatusBadRequest}
	}
	now := time.Now().UTC()
	updates := map[string]any{
		"banned_at":    now,
		"banned_by_id": actorID,
	}
	if input.Reason != nil {
		updates["ban_reason"] = *input.Reason
	}
	if err := s.repo.Update(id, updates); err != nil {
		return nil, err
	}
	return s.Get(id)
}

func (s *Service) Unban(id string) (*domain.User, error) {
	if _, err := s.Get(id); err != nil {
		return nil, err
	}
	if err := s.repo.Update(id, map[string]any{
		"banned_at":    gorm.Expr("NULL"),
		"banned_by_id": gorm.Expr("NULL"),
		"ban_reason":   gorm.Expr("NULL"),
	}); err != nil {
		return nil, err
	}
	return s.Get(id)
}

func validRole(r string) bool {
	switch domain.UserRole(r) {
	case domain.RoleCitizen, domain.RoleRepresentative, domain.RoleGovernmentAdmin,
		domain.RoleNGO, domain.RoleModerator, domain.RolePlatformAdmin:
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
