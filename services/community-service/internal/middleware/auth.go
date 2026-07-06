package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/civicos/community-service/pkg/config"
	"github.com/civicos/community-service/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

// isSafeMethod skips the ban check for reads. See identity-service/
// internal/middleware/auth.go for the rationale.
func isSafeMethod(m string) bool {
	return m == "GET" || m == "HEAD" || m == "OPTIONS"
}

// isBanned checks banned_at directly against the users table on the
// shared DB. Fails open on infra errors — a banned user still gets
// caught on the next request when the DB is reachable again.
func isBanned(db *gorm.DB, userID string) bool {
	if db == nil || userID == "" {
		return false
	}
	var count int64
	if err := db.Raw(
		"SELECT COUNT(1) FROM users WHERE id = ? AND (banned_at IS NOT NULL OR deleted_at IS NOT NULL)",
		userID,
	).Scan(&count).Error; err != nil {
		return false
	}
	return count > 0
}

func activeCommunityID(db *gorm.DB, userID string) string {
	if db == nil || userID == "" {
		return ""
	}
	var row struct {
		CommunityID *string `gorm:"column:community_id"`
	}
	if err := db.Table("users").Select("community_id").Where("id = ?", userID).Scan(&row).Error; err != nil {
		return ""
	}
	if row.CommunityID == nil {
		return ""
	}
	return *row.CommunityID
}

type Claims struct {
	UserID        string `json:"sub"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	Role          string `json:"role"`
	EmailVerified bool   `json:"emailVerified"`
	jwt.RegisteredClaims
}

// JWTAuth validates the Bearer token. When db is non-nil, non-safe
// methods also check banned_at and return 403 ACCOUNT_BANNED — the ban
// enforcement guarantee that prevents a banned user from posting new
// content until their access token expires naturally.
func JWTAuth(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenStr string
		header := c.GetHeader("Authorization")
		switch {
		case strings.HasPrefix(header, "Bearer "):
			tokenStr = strings.TrimPrefix(header, "Bearer ")
		default:
			// Fallback for clients that can't set headers (e.g. EventSource).
			tokenStr = c.Query("access_token")
		}
		if tokenStr == "" {
			response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Missing or malformed token")
			c.Abort()
			return
		}
		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(cfg.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Token is invalid or expired")
			c.Abort()
			return
		}

		if !isSafeMethod(c.Request.Method) && isBanned(db, claims.UserID) {
			response.Error(c, http.StatusForbidden, "ACCOUNT_BANNED",
				"Your account has been suspended. Contact support if you believe this is a mistake.")
			c.Abort()
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("userName", claims.Name)
		c.Set("userRole", claims.Role)
		c.Set("emailVerified", claims.EmailVerified)
		c.Set("activeCommunityID", activeCommunityID(db, claims.UserID))
		c.Next()
	}
}

// RequireVerified blocks the request if the caller's JWT says the email is
// not yet verified. Designed to layer on top of JWTAuth — apply it only to
// write endpoints (file an issue, sign a petition, post a comment).
// Read endpoints stay open so unverified users can still explore.
func RequireVerified() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !c.GetBool("emailVerified") {
			response.Error(c, http.StatusForbidden, "EMAIL_NOT_VERIFIED", "Verify your email to perform this action")
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireRole returns a middleware that allows the request only if the
// caller's role (from a prior JWTAuth) is in the allowed set.
func RequireRole(allowed ...string) gin.HandlerFunc {
	set := make(map[string]struct{}, len(allowed))
	for _, r := range allowed {
		set[r] = struct{}{}
	}
	return func(c *gin.Context) {
		role, _ := c.Get("userRole")
		roleStr, _ := role.(string)
		if _, ok := set[roleStr]; !ok {
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "Your role is not permitted to perform this action")
			c.Abort()
			return
		}
		c.Next()
	}
}
