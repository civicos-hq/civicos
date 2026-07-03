package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/civicos/identity-service/internal/domain"
	"github.com/civicos/identity-service/pkg/config"
	"github.com/civicos/identity-service/pkg/mailer"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	verificationTokenTTL  = 24 * time.Hour
	passwordResetTokenTTL = 1 * time.Hour
)

type UserStore interface {
	FindByEmail(email string) (*domain.User, error)
	FindByID(id string) (*domain.User, error)
	Create(user *domain.User) error
	UpdateCommunity(userID, communityID string) error
	UpdateProfile(userID, name, email string) error
	SetVerificationToken(userID, tokenHash string, expiresAt time.Time) error
	FindByVerificationTokenHash(tokenHash string) (*domain.User, error)
	MarkVerified(userID string) error
	SetPasswordResetToken(userID, tokenHash string, expiresAt time.Time) error
	FindByPasswordResetTokenHash(tokenHash string) (*domain.User, error)
	ResetPassword(userID, newPasswordHash string) error
}

type Service struct {
	repo   UserStore
	cfg    *config.Config
	mailer mailer.Mailer
}

func NewService(repo UserStore, cfg *config.Config, m mailer.Mailer) *Service {
	return &Service{repo: repo, cfg: cfg, mailer: m}
}

type RegisterInput struct {
	Name     string `json:"name" binding:"required,min=2,max=100"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type LoginInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type TokenPair struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"`
}

type AuthClaims struct {
	UserID        string          `json:"sub"`
	Email         string          `json:"email"`
	Name          string          `json:"name"`
	Role          domain.UserRole `json:"role"`
	EmailVerified bool            `json:"emailVerified"`
	jwt.RegisteredClaims
}

func (s *Service) Register(input RegisterInput) (*domain.PublicUser, *TokenPair, error) {
	// Check for existing account
	_, err := s.repo.FindByEmail(input.Email)
	if err == nil {
		return nil, nil, errors.New("EMAIL_ALREADY_IN_USE")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), 12)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &domain.User{
		ID:           uuid.New().String(),
		Name:         input.Name,
		Email:        input.Email,
		PasswordHash: string(hash),
		Role:         domain.RoleCitizen,
	}

	if err := s.repo.Create(user); err != nil {
		return nil, nil, err
	}

	// Fire-and-log verification email. A mailer failure does not roll back the
	// registration — the user can hit "Resend" from the dashboard banner.
	if err := s.sendVerificationEmail(user); err != nil {
		log.Printf("[auth.Register] verification email failed for user=%s: %v", user.ID, err)
	}

	// Issue tokens so the client lands logged-in (emailVerified=false). The
	// resend-verification endpoint requires auth — auto-login keeps that path
	// reachable from the post-register "check your email" screen.
	tokens, err := s.signTokens(user)
	if err != nil {
		return nil, nil, err
	}

	public := user.ToPublic()
	return &public, tokens, nil
}

// VerifyEmail finds the user with this token hash, checks expiry, and flips
// the verified bit. The raw token is single-use — MarkVerified clears it.
// Returns fresh tokens so the client immediately picks up the new verified
// claim (the old access token would still read emailVerified=false).
func (s *Service) VerifyEmail(rawToken string) (*domain.PublicUser, *TokenPair, error) {
	if rawToken == "" {
		return nil, nil, errors.New("VERIFICATION_TOKEN_INVALID")
	}
	user, err := s.repo.FindByVerificationTokenHash(hashToken(rawToken))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, errors.New("VERIFICATION_TOKEN_INVALID")
		}
		return nil, nil, err
	}
	if user.EmailVerified {
		public := user.ToPublic()
		return &public, nil, nil
	}
	if user.EmailVerificationExpiresAt == nil || time.Now().After(*user.EmailVerificationExpiresAt) {
		return nil, nil, errors.New("VERIFICATION_TOKEN_EXPIRED")
	}
	if err := s.repo.MarkVerified(user.ID); err != nil {
		return nil, nil, err
	}

	fresh, err := s.repo.FindByID(user.ID)
	if err != nil {
		return nil, nil, err
	}
	tokens, err := s.signTokens(fresh)
	if err != nil {
		return nil, nil, err
	}
	public := fresh.ToPublic()
	return &public, tokens, nil
}

// ResendVerification re-issues a token for an already-registered user.
// Idempotent: calling it on a verified account returns success without sending.
func (s *Service) ResendVerification(userID string) error {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return errors.New("USER_NOT_FOUND")
	}
	if user.EmailVerified {
		return nil
	}
	return s.sendVerificationEmail(user)
}

// RequestPasswordReset always succeeds — even when the email doesn't match a
// registered account. Silent-on-unknown is the standard defence against email
// enumeration ("does this address have an account?"). We still log the miss
// server-side so operators can spot brute-force patterns.
func (s *Service) RequestPasswordReset(email string) error {
	user, err := s.repo.FindByEmail(email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("[auth.RequestPasswordReset] unknown email — silently ignored: %s", email)
			return nil
		}
		return err
	}

	raw, err := newRandomToken(32)
	if err != nil {
		return fmt.Errorf("generate reset token: %w", err)
	}
	expiresAt := time.Now().Add(passwordResetTokenTTL).UTC()
	if err := s.repo.SetPasswordResetToken(user.ID, hashToken(raw), expiresAt); err != nil {
		return fmt.Errorf("persist reset token: %w", err)
	}

	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.cfg.AppURL, url.QueryEscape(raw))
	subject, html, text := mailer.PasswordResetEmail(user.Name, resetURL)
	if err := s.mailer.Send(user.Email, subject, html, text); err != nil {
		log.Printf("[auth.RequestPasswordReset] mail send failed for user=%s: %v", user.ID, err)
		return err
	}
	return nil
}

// ResetPassword swaps the password hash and issues fresh tokens so the user
// lands logged-in on the dashboard. The raw token is single-use.
func (s *Service) ResetPassword(rawToken, newPassword string) (*domain.PublicUser, *TokenPair, error) {
	if rawToken == "" {
		return nil, nil, errors.New("RESET_TOKEN_INVALID")
	}
	if len(newPassword) < 8 {
		return nil, nil, errors.New("PASSWORD_TOO_SHORT")
	}
	user, err := s.repo.FindByPasswordResetTokenHash(hashToken(rawToken))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, errors.New("RESET_TOKEN_INVALID")
		}
		return nil, nil, err
	}
	if user.PasswordResetExpiresAt == nil || time.Now().After(*user.PasswordResetExpiresAt) {
		return nil, nil, errors.New("RESET_TOKEN_EXPIRED")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		return nil, nil, fmt.Errorf("hash password: %w", err)
	}
	if err := s.repo.ResetPassword(user.ID, string(hash)); err != nil {
		return nil, nil, err
	}

	fresh, err := s.repo.FindByID(user.ID)
	if err != nil {
		return nil, nil, err
	}
	tokens, err := s.signTokens(fresh)
	if err != nil {
		return nil, nil, err
	}
	public := fresh.ToPublic()
	return &public, tokens, nil
}

func (s *Service) sendVerificationEmail(user *domain.User) error {
	raw, err := newRandomToken(32)
	if err != nil {
		return fmt.Errorf("generate verification token: %w", err)
	}
	expiresAt := time.Now().Add(verificationTokenTTL).UTC()
	if err := s.repo.SetVerificationToken(user.ID, hashToken(raw), expiresAt); err != nil {
		return fmt.Errorf("persist verification token: %w", err)
	}

	verifyURL := fmt.Sprintf("%s/verify-email?token=%s", s.cfg.AppURL, url.QueryEscape(raw))
	subject, html, text := mailer.VerificationEmail(user.Name, verifyURL)
	return s.mailer.Send(user.Email, subject, html, text)
}

func newRandomToken(byteLen int) (string, error) {
	buf := make([]byte, byteLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func (s *Service) Login(input LoginInput) (*domain.PublicUser, *TokenPair, error) {
	user, err := s.repo.FindByEmail(input.Email)
	if err != nil {
		// Return the same error for both "not found" and "wrong password"
		// to prevent email enumeration
		return nil, nil, errors.New("INVALID_CREDENTIALS")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, nil, errors.New("INVALID_CREDENTIALS")
	}

	tokens, err := s.signTokens(user)
	if err != nil {
		return nil, nil, err
	}

	public := user.ToPublic()
	return &public, tokens, nil
}

func (s *Service) RefreshTokens(refreshToken string) (*TokenPair, error) {
	claims, err := s.parseToken(refreshToken)
	if err != nil {
		return nil, errors.New("INVALID_REFRESH_TOKEN")
	}

	user, err := s.repo.FindByID(claims.UserID)
	if err != nil {
		return nil, errors.New("INVALID_REFRESH_TOKEN")
	}

	return s.signTokens(user)
}

func (s *Service) GetMe(userID string) (*domain.PublicUser, error) {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return nil, errors.New("USER_NOT_FOUND")
	}
	public := user.ToPublic()
	return &public, nil
}

// UpdateProfileInput uses pointers so the handler can distinguish "leave alone"
// (nil) from "set to empty" — only fields the client actually sent are touched.
type UpdateProfileInput struct {
	Name  *string `json:"name" binding:"omitempty,min=2,max=100"`
	Email *string `json:"email" binding:"omitempty,email"`
}

func (s *Service) UpdateProfile(userID string, input UpdateProfileInput) (*domain.PublicUser, error) {
	if input.Name == nil && input.Email == nil {
		return s.GetMe(userID)
	}

	if input.Email != nil {
		existing, err := s.repo.FindByEmail(*input.Email)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		if existing != nil && existing.ID != userID {
			return nil, errors.New("EMAIL_ALREADY_IN_USE")
		}
	}

	name := ""
	if input.Name != nil {
		name = *input.Name
	}
	email := ""
	if input.Email != nil {
		email = *input.Email
	}
	if err := s.repo.UpdateProfile(userID, name, email); err != nil {
		return nil, err
	}
	return s.GetMe(userID)
}

func (s *Service) JoinCommunity(userID, communityID string) (*domain.PublicUser, error) {
	if err := s.repo.UpdateCommunity(userID, communityID); err != nil {
		return nil, err
	}
	return s.GetMe(userID)
}

// signTokens is the single place token pairs are issued — no duplicate logic.
func (s *Service) signTokens(user *domain.User) (*TokenPair, error) {
	accessExpiry := time.Now().Add(7 * 24 * time.Hour)
	refreshExpiry := time.Now().Add(30 * 24 * time.Hour)

	accessClaims := AuthClaims{
		UserID:        user.ID,
		Email:         user.Email,
		Name:          user.Name,
		Role:          user.Role,
		EmailVerified: user.EmailVerified,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExpiry),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	refreshClaims := AuthClaims{
		UserID:        user.ID,
		Email:         user.Email,
		Name:          user.Name,
		Role:          user.Role,
		EmailVerified: user.EmailVerified,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExpiry),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    accessExpiry.Unix(),
	}, nil
}

func (s *Service) parseToken(tokenStr string) (*AuthClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &AuthClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}
	claims, ok := token.Claims.(*AuthClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}
	return claims, nil
}
