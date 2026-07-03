package auth

import (
	"strings"
	"testing"
	"time"

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

func (s *inMemoryUserStore) UpdateCommunity(userID, communityID string) error {
	if user, ok := s.usersByID[userID]; ok {
		user.CommunityID = &communityID
	}
	return nil
}

func (s *inMemoryUserStore) UpdateProfile(userID, name, email string) error {
	user, ok := s.usersByID[userID]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	if name != "" {
		user.Name = name
	}
	if email != "" {
		delete(s.usersByEmail, user.Email)
		user.Email = email
		s.usersByEmail[email] = user
	}
	return nil
}

func (s *inMemoryUserStore) SetVerificationToken(userID, tokenHash string, expiresAt time.Time) error {
	user, ok := s.usersByID[userID]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	user.EmailVerificationTokenHash = &tokenHash
	user.EmailVerificationExpiresAt = &expiresAt
	return nil
}

func (s *inMemoryUserStore) FindByVerificationTokenHash(tokenHash string) (*domain.User, error) {
	for _, user := range s.usersByID {
		if user.EmailVerificationTokenHash != nil && *user.EmailVerificationTokenHash == tokenHash {
			return user, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (s *inMemoryUserStore) MarkVerified(userID string) error {
	user, ok := s.usersByID[userID]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	now := time.Now().UTC()
	user.EmailVerified = true
	user.EmailVerifiedAt = &now
	user.EmailVerificationTokenHash = nil
	user.EmailVerificationExpiresAt = nil
	return nil
}

// captureMailer records the last verification URL so VerifyEmail can be
// exercised without spinning up SMTP.
type captureMailer struct {
	lastTo      string
	lastSubject string
	lastText    string
}

func (m *captureMailer) Send(to, subject, htmlBody, textBody string) error {
	m.lastTo = to
	m.lastSubject = subject
	m.lastText = textBody
	return nil
}

func TestRegisterAndLoginFlow(t *testing.T) {
	cfg := &config.Config{JWTSecret: "12345678901234567890123456789012", AppURL: "http://localhost:5173"}
	repo := newInMemoryUserStore()
	svc := NewService(repo, cfg, &captureMailer{})

	user, regTokens, err := svc.Register(RegisterInput{
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
	if user.EmailVerified {
		t.Fatalf("expected fresh registration to be unverified")
	}
	if regTokens == nil || regTokens.AccessToken == "" {
		t.Fatalf("expected register to auto-login by returning a token pair")
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

func TestVerifyEmailFlow(t *testing.T) {
	cfg := &config.Config{JWTSecret: "12345678901234567890123456789012", AppURL: "http://localhost:5173"}
	repo := newInMemoryUserStore()
	mail := &captureMailer{}
	svc := NewService(repo, cfg, mail)

	if _, _, err := svc.Register(RegisterInput{
		Name:     "Ada",
		Email:    "ada@example.com",
		Password: "password123",
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	rawToken := extractToken(t, mail.lastText)

	publicUser, tokens, err := svc.VerifyEmail(rawToken)
	if err != nil {
		t.Fatalf("verify-email: %v", err)
	}
	if !publicUser.EmailVerified {
		t.Fatalf("expected user to be verified after VerifyEmail")
	}
	if tokens == nil || tokens.AccessToken == "" {
		t.Fatalf("expected fresh tokens to be reissued so the client picks up emailVerified=true")
	}

	// Single-use: replaying the same token must now fail.
	if _, _, err := svc.VerifyEmail(rawToken); err == nil {
		t.Fatalf("expected replayed token to be rejected")
	}
}

func extractToken(t *testing.T, mailText string) string {
	t.Helper()
	const marker = "token="
	idx := strings.Index(mailText, marker)
	if idx < 0 {
		t.Fatalf("expected mail body to contain token, got: %q", mailText)
	}
	tail := mailText[idx+len(marker):]
	end := strings.IndexAny(tail, " \n\r")
	if end < 0 {
		return tail
	}
	return tail[:end]
}
