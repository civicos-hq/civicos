package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/civicos/identity-service/internal/auth"
	"github.com/civicos/identity-service/pkg/config"
	"github.com/civicos/identity-service/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

// isSafeMethod reports whether the request is a read that doesn't need
// the ban check. Ban enforcement runs on writes only so we don't add a
// DB roundtrip to every GET (dashboards, feeds, etc.).
func isSafeMethod(m string) bool {
	return m == "GET" || m == "HEAD" || m == "OPTIONS"
}

// isBanned checks banned_at directly against the users table. Cheap
// index lookup on the primary key + partial predicate — sub-millisecond
// in practice. Never blocks the request if the query itself errors
// (fail open on infrastructure hiccups; a banned user still gets caught
// on the next request when the DB is back).
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

// JWTAuth validates the Bearer token and injects userID + role into the
// context. When db is non-nil, non-safe methods also check banned_at
// and return 403 ACCOUNT_BANNED if the user is suspended — the "ban
// actually blocks" guarantee that the citizen surface needs.
func JWTAuth(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Missing or malformed token")
			c.Abort()
			return
		}

		tokenStr := strings.TrimPrefix(header, "Bearer ")
		claims := &auth.AuthClaims{}

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
		c.Set("userEmail", claims.Email)
		c.Set("userName", claims.Name)
		c.Set("userRole", string(claims.Role))
		c.Set("emailVerified", claims.EmailVerified)
		c.Next()
	}
}

// RequireVerified blocks the request if the caller's JWT says their email
// isn't yet verified. Layered on top of JWTAuth for endpoints that
// citizens shouldn't be able to hit while their account is still
// unverified (like filing a moderation flag).
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
