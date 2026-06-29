package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/civicos/community-service/pkg/config"
	"github.com/civicos/community-service/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID string `json:"sub"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func JWTAuth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Missing or malformed token")
			c.Abort()
			return
		}

		tokenStr := strings.TrimPrefix(header, "Bearer ")
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

		c.Set("userID", claims.UserID)
		c.Set("userRole", claims.Role)
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
