package middleware

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/civicos/api-gateway/pkg/ratelimit"
	"github.com/gin-gonic/gin"
)

// Profile groups an endpoint into a shared budget so we don't have to reason
// about every route individually. Values are chosen for defence-in-depth,
// not fairness — a legitimate user should never hit these under normal
// operation. Tighten later once we have metrics on real traffic.
type Profile struct {
	Name   string        // used in the Redis key and X-RateLimit-* prefix
	Limit  int           // requests per window
	Window time.Duration // rolling window
}

// limitFromEnv lets operators retune a tier from the environment
// (RATE_LIMIT_STRICT / _STANDARD / _LENIENT, requests per minute) without a
// rebuild — rate-limit tuning has proven to be an ops decision, not a code
// decision. Invalid or missing values keep the compiled default.
func limitFromEnv(envKey string, fallback int) int {
	if v := os.Getenv(envKey); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return fallback
}

var (
	// Auth endpoints that are the most abused in the wild: register, login,
	// forgot-password. All strict routes share one bucket per IP, and many
	// ISPs (notably Nigerian mobile carriers) put whole subscriber pools
	// behind one CGNAT address, so per-IP budgets must absorb dozens of
	// legitimate users plus dev/test bursts. 60/min still caps a
	// single-IP brute force at ~1 guess/sec against bcrypt-cost-12 hashes;
	// account-level lockout is the real defence to add before scale.
	Strict = Profile{Name: "strict", Limit: limitFromEnv("RATE_LIMIT_STRICT", 60), Window: time.Minute}

	// Standard authed writes: upvote, sign, follow, comment, upload, verify,
	// resend — keyed by userID. Sized so bulk admin sessions (seeding
	// communities, creating representatives back-to-back) don't trip it.
	Standard = Profile{Name: "standard", Limit: limitFromEnv("RATE_LIMIT_STANDARD", 60), Window: time.Minute}

	// High-frequency lifecycle calls: token refresh, logout. The frontend
	// interceptor coalesces refreshes across concurrent 401s, so this is
	// deliberately generous and mostly there to catch runaway loops.
	Lenient = Profile{Name: "lenient", Limit: limitFromEnv("RATE_LIMIT_LENIENT", 120), Window: time.Minute}
)

// Limit returns a Gin middleware that enforces the given profile.
//
// Keying:
//   - If a userID is present in the context (set by JWTAuth), we key by
//     "u:<userID>" — same user can't burn separate quotas per IP.
//   - Otherwise we key by the remote IP.
//
// Applied AFTER JWTAuth for authenticated routes so we can use the userID.
// For anonymous routes, apply this on its own.
//
// A nil *ratelimit.Limiter is a valid dependency and turns the middleware
// into a no-op. That lets us wire the middleware unconditionally even in
// dev environments without Redis.
func Limit(lim *ratelimit.Limiter, p Profile) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := rateLimitKey(c, p.Name)

		res, err := lim.Allow(c.Request.Context(), key, p.Limit, p.Window)
		if err != nil {
			// Fail-open — Allow already returned {Allowed:true} on Redis
			// errors, so we should never actually see this branch. Left as
			// belt-and-braces in case a future variant surfaces errors.
			c.Next()
			return
		}

		// Always advertise the current budget — clients that pay attention
		// can pace themselves before we start rejecting.
		c.Header("X-RateLimit-Limit", strconv.Itoa(res.Limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(res.Remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(res.ResetAt.Unix(), 10))

		if !res.Allowed {
			retrySeconds := int(res.RetryAfter.Seconds())
			if retrySeconds < 1 {
				retrySeconds = 1
			}
			c.Header("Retry-After", strconv.Itoa(retrySeconds))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"code":    "RATE_LIMITED",
				"message": fmt.Sprintf("Too many requests. Try again in %ds.", retrySeconds),
				"data":    gin.H{"retryAfter": retrySeconds},
			})
			return
		}

		c.Next()
	}
}

// rateLimitKey builds the per-caller key. Prefixing with the profile name
// keeps buckets for different limit tiers isolated on the same key — the
// register-tier bucket for user X doesn't burn the upvote-tier bucket.
func rateLimitKey(c *gin.Context, profileName string) string {
	if raw, ok := c.Get("userID"); ok {
		if userID, ok := raw.(string); ok && userID != "" {
			return profileName + ":u:" + userID
		}
	}
	// ClientIP walks X-Forwarded-For if the proxy is trusted; falls back
	// to the direct socket peer otherwise.
	return profileName + ":ip:" + c.ClientIP()
}
