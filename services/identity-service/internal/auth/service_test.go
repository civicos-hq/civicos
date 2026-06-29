package auth

import (
	"testing"

	"github.com/civicos/identity-service/internal/domain"
	"github.com/civicos/identity-service/pkg/config"
	"gorm.io/gorm"
)

type inMemoryUserStore struct {
	usersByID    map[string]*domain.User
	usersByEmail map[string]*domain.User
}

func newInMemoryUserStore() *inMemoryUserStore {
	return &inMemoryUserStore{
		usersByID:    make(map[string]*domain.User),
		usersByEmail: make(map[string]*domain.User),
	}
}

func (s *inMemoryUserStore) FindByEmail(email string) (*domain.User, error) {
	user, ok := s.usersByEmail[email]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return user, nil
}

func (s *inMemoryUserStore) FindByID(id string) (*domain.User, error) {
	user, ok := s.usersByID[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return user, nil
}

func (s *inMemoryUserStore) Create(user *domain.User) error {
	s.usersByID[user.ID] = user
	s.usersByEmail[user.Email] = user
	return nil
}

func TestRegisterAndLoginFlow(t *testing.T) {
	cfg := &config.Config{JWTSecret: "12345678901234567890123456789012"}
	repo := newInMemoryUserStore()
	svc := NewService(repo, cfg)

	user, err := svc.Register(RegisterInput{
		Name:     "Ada",
		Email:    "ada@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if user.Email != "ada@example.com" {
		t.Fatalf("expected registered email to be preserved, got %s", user.Email)
	}

	loggedInUser, tokens, err := svc.Login(LoginInput{
		Email:    "ada@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if loggedInUser.ID != user.ID {
		t.Fatalf("expected logged in user id %s, got %s", user.ID, loggedInUser.ID)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatalf("expected both tokens to be issued")
	}
}
