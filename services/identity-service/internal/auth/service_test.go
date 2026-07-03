package auth

import (
	"errors"
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

func (s *inMemoryUserStore) SetPasswordResetToken(userID, tokenHash string, expiresAt time.Time) error {
	user, ok := s.usersByID[userID]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	user.PasswordResetTokenHash = &tokenHash
	user.PasswordResetExpiresAt = &expiresAt
	return nil
}

func (s *inMemoryUserStore) FindByPasswordResetTokenHash(tokenHash string) (*domain.User, error) {
	for _, user := range s.usersByID {
		if user.PasswordResetTokenHash != nil && *user.PasswordResetTokenHash == tokenHash {
			return user, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (s *inMemoryUserStore) ResetPassword(userID, newPasswordHash string) error {
	user, ok := s.usersByID[userID]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	user.PasswordHash = newPasswordHash
	user.PasswordResetTokenHash = nil
	user.PasswordResetExpiresAt = nil
	return nil
}

// inMemoryRefreshStore lets rotation + replay tests run without Postgres.
// Keyed by token hash — mirrors the real repo's uniqueIndex.
type inMemoryRefreshStore struct {
	byID   map[string]*domain.RefreshToken
	byHash map[string]*domain.RefreshToken
}

func newInMemoryRefreshStore() *inMemoryRefreshStore {
	return &inMemoryRefreshStore{
		byID:   map[string]*domain.RefreshToken{},
		byHash: map[string]*domain.RefreshToken{},
	}
}

func (s *inMemoryRefreshStore) Create(t *domain.RefreshToken) error {
	if _, exists := s.byHash[t.TokenHash]; exists {
		return errors.New("duplicate token hash")
	}
	s.byID[t.ID] = t
	s.byHash[t.TokenHash] = t
	return nil
}

func (s *inMemoryRefreshStore) FindByHash(hash string) (*domain.RefreshToken, error) {
	if tok, ok := s.byHash[hash]; ok {
		return tok, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (s *inMemoryRefreshStore) Consume(id string, at time.Time) (int64, error) {
	tok, ok := s.byID[id]
	if !ok || tok.ConsumedAt != nil || tok.RevokedAt != nil {
		return 0, nil
	}
	tok.ConsumedAt = &at
	return 1, nil
}

func (s *inMemoryRefreshStore) RevokeFamily(familyID string, at time.Time) error {
	for _, tok := range s.byID {
		if tok.FamilyID == familyID && tok.RevokedAt == nil {
			revoked := at
			tok.RevokedAt = &revoked
		}
	}
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
	svc := NewService(repo, newInMemoryRefreshStore(), cfg, &captureMailer{})

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
	svc := NewService(repo, newInMemoryRefreshStore(), cfg, mail)

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

func TestForgotAndResetPasswordFlow(t *testing.T) {
	cfg := &config.Config{JWTSecret: "12345678901234567890123456789012", AppURL: "http://localhost:5173"}
	repo := newInMemoryUserStore()
	mail := &captureMailer{}
	svc := NewService(repo, newInMemoryRefreshStore(), cfg, mail)

	if _, _, err := svc.Register(RegisterInput{
		Name:     "Ada",
		Email:    "ada@example.com",
		Password: "originalPassword",
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	// Unknown email must return nil error (silent to prevent enumeration) and
	// must not send a mail.
	mail.lastText = ""
	if err := svc.RequestPasswordReset("nobody@example.com"); err != nil {
		t.Fatalf("expected nil error for unknown email, got %v", err)
	}
	if mail.lastText != "" {
		t.Fatalf("expected no mail sent for unknown email")
	}

	// Known email — must send a reset link.
	if err := svc.RequestPasswordReset("ada@example.com"); err != nil {
		t.Fatalf("forgot-password: %v", err)
	}
	if mail.lastSubject != "Reset your CivicOS password" {
		t.Fatalf("expected reset subject, got %q", mail.lastSubject)
	}
	rawToken := extractToken(t, mail.lastText)

	// Reset with a too-short password must be rejected before touching the DB.
	if _, _, err := svc.ResetPassword(rawToken, "short"); err == nil ||
		err.Error() != "PASSWORD_TOO_SHORT" {
		t.Fatalf("expected PASSWORD_TOO_SHORT, got %v", err)
	}

	publicUser, tokens, err := svc.ResetPassword(rawToken, "brandNewPassword")
	if err != nil {
		t.Fatalf("reset-password: %v", err)
	}
	if publicUser.Email != "ada@example.com" {
		t.Fatalf("expected same user, got %s", publicUser.Email)
	}
	if tokens == nil || tokens.AccessToken == "" {
		t.Fatalf("expected fresh tokens (auto-login) after reset")
	}

	// Old password must no longer log in; new one must.
	if _, _, err := svc.Login(LoginInput{Email: "ada@example.com", Password: "originalPassword"}); err == nil {
		t.Fatalf("expected old password to be rejected after reset")
	}
	if _, _, err := svc.Login(LoginInput{Email: "ada@example.com", Password: "brandNewPassword"}); err != nil {
		t.Fatalf("expected new password to work, got %v", err)
	}

	// Replaying the same reset token must fail (single-use).
	if _, _, err := svc.ResetPassword(rawToken, "anotherOne"); err == nil {
		t.Fatalf("expected replayed reset token to be rejected")
	}
}

func TestRefreshTokenRotation(t *testing.T) {
	cfg := &config.Config{JWTSecret: "12345678901234567890123456789012", AppURL: "http://localhost:5173"}
	repo := newInMemoryUserStore()
	refresh := newInMemoryRefreshStore()
	svc := NewService(repo, refresh, cfg, &captureMailer{})

	_, tokens, err := svc.Register(RegisterInput{
		Name: "Ada", Email: "ada@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	firstRefresh := tokens.RefreshToken
	if firstRefresh == "" {
		t.Fatalf("expected an initial refresh token")
	}

	// A successful rotation must return NEW tokens — the old refresh must no
	// longer work.
	rotated, err := svc.RefreshTokens(firstRefresh)
	if err != nil {
		t.Fatalf("first rotation: %v", err)
	}
	if rotated.RefreshToken == firstRefresh {
		t.Fatalf("rotation should hand out a fresh refresh token")
	}

	// Replay of the original consumed token: must (a) fail and (b) revoke
	// the whole family, so even the freshly-rotated token becomes unusable.
	if _, err := svc.RefreshTokens(firstRefresh); err == nil {
		t.Fatalf("replay of consumed refresh must be rejected")
	}
	if _, err := svc.RefreshTokens(rotated.RefreshToken); err == nil {
		t.Fatalf("family revocation must invalidate the rotated token too")
	}
}

func TestLogoutRevokesFamily(t *testing.T) {
	cfg := &config.Config{JWTSecret: "12345678901234567890123456789012", AppURL: "http://localhost:5173"}
	repo := newInMemoryUserStore()
	refresh := newInMemoryRefreshStore()
	svc := NewService(repo, refresh, cfg, &captureMailer{})

	_, tokens, err := svc.Register(RegisterInput{
		Name: "Ada", Email: "ada@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	// Rotate once so there are two tokens in the family — a live one and a
	// consumed one. Logout must invalidate both.
	rotated, err := svc.RefreshTokens(tokens.RefreshToken)
	if err != nil {
		t.Fatalf("rotate before logout: %v", err)
	}
	if err := svc.Logout(rotated.RefreshToken); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, err := svc.RefreshTokens(rotated.RefreshToken); err == nil {
		t.Fatalf("post-logout rotation must fail")
	}

	// Logout with an unknown refresh must NOT return an error — clients call
	// this from a stale session and expect to always succeed locally.
	if err := svc.Logout("does-not-exist"); err != nil {
		t.Fatalf("logout with unknown token should be a no-op, got %v", err)
	}
}

// Prevent unused-import complaints in the rare test-only build case.
var _ = errors.New

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
