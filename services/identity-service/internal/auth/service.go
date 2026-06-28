package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/civicos/identity-service/internal/domain"
	"github.com/civicos/identity-service/pkg/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Service struct {
	repo *Repository
	cfg  *config.Config
}

func NewService(repo *Repository, cfg *config.Config) *Service {
	return &Service{repo: repo, cfg: cfg}
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
	UserID string            `json:"sub"`
	Email  string            `json:"email"`
	Role   domain.UserRole   `json:"role"`
	jwt.RegisteredClaims
}

func (s *Service) Register(input RegisterInput) (*domain.PublicUser, error) {
	// Check for existing account
	_, err := s.repo.FindByEmail(input.Email)
	if err == nil {
		return nil, errors.New("EMAIL_ALREADY_IN_USE")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), 12)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &domain.User{
		ID:           uuid.New().String(),
		Name:         input.Name,
		Email:        input.Email,
		PasswordHash: string(hash),
		Role:         domain.RoleCitizen,
	}

	if err := s.repo.Create(user); err != nil {
		return nil, err
	}

	public := user.ToPublic()
	return &public, nil
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

// signTokens is the single place token pairs are issued — no duplicate logic.
func (s *Service) signTokens(user *domain.User) (*TokenPair, error) {
	accessExpiry := time.Now().Add(7 * 24 * time.Hour)
	refreshExpiry := time.Now().Add(30 * 24 * time.Hour)

	accessClaims := AuthClaims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
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
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
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
