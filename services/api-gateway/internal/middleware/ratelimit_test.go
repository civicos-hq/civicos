package middleware

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/civicos/api-gateway/pkg/ratelimit"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// TestLimitMiddleware exercises the full request/response contract of the
// middleware end-to-end — headers, 429 status, RATE_LIMITED body — against
// a real gin router and an in-process Redis. Anything less would leave the
// wire format untested.
func TestLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	lim := ratelimit.New(rdb)

	// Tight profile so we hit the limit in the same test tick.
	profile := Profile{Name: "test", Limit: 2, Window: time.Minute}

	r := gin.New()
	r.POST("/x", Limit(lim, profile), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	hit := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/x", nil)
		req.RemoteAddr = "1.2.3.4:5678"
		r.ServeHTTP(w, req)
		return w
	}

	// First two calls succeed. Remaining should tick down.
	for i := 0; i < 2; i++ {
		w := hit()
		if w.Code != http.StatusOK {
			t.Fatalf("call %d: expected 200, got %d", i, w.Code)
		}
		wantRemaining := 2 - (i + 1)
		if got := w.Header().Get("X-RateLimit-Remaining"); got != strconv.Itoa(wantRemaining) {
			t.Fatalf("call %d: X-RateLimit-Remaining=%q, want %d", i, got, wantRemaining)
		}
	}

	// Third call: 429 with the RATE_LIMITED error envelope and Retry-After header.
	w := hit()
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("over-limit call: expected 429, got %d (body=%s)", w.Code, w.Body.String())
	}
	if w.Header().Get("Retry-After") == "" {
		t.Fatalf("expected Retry-After header on 429")
	}
	if w.Header().Get("X-RateLimit-Remaining") != "0" {
		t.Fatalf("expected X-RateLimit-Remaining=0, got %q", w.Header().Get("X-RateLimit-Remaining"))
	}

	body, _ := io.ReadAll(w.Body)
	var resp map[string]any
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode body: %v (raw=%s)", err, body)
	}
	if resp["code"] != "RATE_LIMITED" {
		t.Fatalf("expected code=RATE_LIMITED, got %v", resp["code"])
	}
}

// TestLimitIsolatesUsers ensures user A's 429 doesn't spill into user B.
// This is the fix for the classic "one IP shares state across accounts"
// mistake with per-userID keying.
func TestLimitIsolatesUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	lim := ratelimit.New(rdb)
	profile := Profile{Name: "iso", Limit: 1, Window: time.Minute}

	// Fake JWTAuth: read a header and set userID in context.
	setUser := func(c *gin.Context) {
		if u := c.GetHeader("X-Test-User"); u != "" {
			c.Set("userID", u)
		}
		c.Next()
	}

	r := gin.New()
	r.POST("/x", setUser, Limit(lim, profile), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	call := func(user string) int {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/x", nil)
		req.RemoteAddr = "9.9.9.9:1234" // same IP for both users
		if user != "" {
			req.Header.Set("X-Test-User", user)
		}
		r.ServeHTTP(w, req)
		return w.Code
	}

	// Burn A's budget.
	if code := call("A"); code != http.StatusOK {
		t.Fatalf("A first call: got %d", code)
	}
	if code := call("A"); code != http.StatusTooManyRequests {
		t.Fatalf("A over-limit: expected 429, got %d", code)
	}

	// B on the same IP still gets their fresh budget.
	if code := call("B"); code != http.StatusOK {
		t.Fatalf("B first call (should share nothing with A): got %d", code)
	}
}
