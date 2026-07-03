package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/civicos/organization-service/pkg/config"
	"github.com/civicos/organization-service/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID        string `json:"sub"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	Role          string `json:"role"`
	EmailVerified bool   `json:"emailVerified"`
	jwt.RegisteredClaims
}

func JWTAuth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenStr string
		header := c.GetHeader("Authorization")
		if strings.HasPrefix(header, "Bearer ") {
			tokenStr = strings.TrimPrefix(header, "Bearer ")
		} else {
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

		c.Set("userID", claims.UserID)
		c.Set("userName", claims.Name)
		c.Set("userRole", claims.Role)
		c.Set("emailVerified", claims.EmailVerified)
		c.Next()
	}
}

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
