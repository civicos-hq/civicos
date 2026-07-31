package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/civicos/api-gateway/pkg/config"
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
		var tokenStr string
		header := c.GetHeader("Authorization")
		switch {
		case strings.HasPrefix(header, "Bearer "):
			tokenStr = strings.TrimPrefix(header, "Bearer ")
		default:
			// Fallback for clients that can't set headers (e.g. EventSource).
			if q := c.Query("access_token"); q != "" {
				tokenStr = q
				c.Request.Header.Set("Authorization", "Bearer "+q)
			}
		}
		if tokenStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   gin.H{"code": "UNAUTHORIZED", "message": "Missing or malformed token"},
			})
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
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   gin.H{"code": "UNAUTHORIZED", "message": "Token is invalid or expired"},
			})
			return
		}

		// Forward identity headers to downstream services
		c.Request.Header.Set("X-User-ID", claims.UserID)
		c.Request.Header.Set("X-User-Email", claims.Email)
		c.Request.Header.Set("X-User-Role", claims.Role)
		// Also expose identity on the gin context so any middleware
		// chained after JWTAuth on this same gateway route can read it
		// without re-parsing the JWT. The rate-limit middleware relies
		// on this: without it, `Limit` fell back to per-IP keying,
		// which was silently the case in production until this fix.
		c.Set("userID", claims.UserID)
		c.Set("userRole", claims.Role)
		c.Next()
	}
}

// OptionalJWTAuth attributes a request to a signed-in user when a valid
// token is present, and lets it through untouched when it is not.
//
// It exists for donations. Giving is deliberately open to guests — a donor
// should not need an account to help — but the route carrying no auth at all
// meant a SIGNED-IN donor was also anonymous: the downstream service never
// saw a user id, so no donation was ever linked to an account. That in turn
// made every donor-facing notification unreachable, because the audience is
// resolved from donations that have a user id.
//
// An invalid or expired token is ignored rather than rejected. The caller
// asked to donate, not to authenticate, and refusing the payment because a
// stale token happened to be in localStorage would cost a donation to punish
// a session problem.
func OptionalJWTAuth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.Next()
			return
		}
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(strings.TrimPrefix(header, "Bearer "), claims, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(cfg.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			// Proceed as a guest. See the note above on why this is not a 401.
			c.Next()
			return
		}
		c.Request.Header.Set("X-User-ID", claims.UserID)
		c.Request.Header.Set("X-User-Email", claims.Email)
		c.Request.Header.Set("X-User-Role", claims.Role)
		c.Set("userID", claims.UserID)
		c.Set("userRole", claims.Role)
		c.Next()
	}
}
