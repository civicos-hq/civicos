package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// newTestLimiter spins up a fake Redis (miniredis is an in-process
// implementation) so tests don't need docker or a live server.
func newTestLimiter(t *testing.T) (*Limiter, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return New(rdb), mr
}

func TestAllowUnderLimit(t *testing.T) {
	lim, _ := newTestLimiter(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		got, err := lim.Allow(ctx, "user-1", 5, time.Minute)
		if err != nil {
			t.Fatalf("allow #%d: %v", i, err)
		}
		if !got.Allowed {
			t.Fatalf("call #%d should have been allowed, remaining=%d", i, got.Remaining)
		}
		wantRemaining := 5 - (i + 1)
		if got.Remaining != wantRemaining {
			t.Fatalf("call #%d remaining=%d, want %d", i, got.Remaining, wantRemaining)
		}
	}
}

func TestAllowOverLimit(t *testing.T) {
	lim, _ := newTestLimiter(t)
	ctx := context.Background()

	// Consume the budget.
	for i := 0; i < 3; i++ {
		if _, err := lim.Allow(ctx, "user-2", 3, time.Minute); err != nil {
			t.Fatalf("warm-up %d: %v", i, err)
		}
	}
	// This one should be denied — the whole bucket is spent.
	got, err := lim.Allow(ctx, "user-2", 3, time.Minute)
	if err != nil {
		t.Fatalf("over-limit call: %v", err)
	}
	if got.Allowed {
		t.Fatalf("expected 4th call to be denied")
	}
	if got.Remaining != 0 {
		t.Fatalf("expected remaining=0 on denial, got %d", got.Remaining)
	}
	if got.RetryAfter <= 0 {
		t.Fatalf("expected RetryAfter > 0 on denial, got %v", got.RetryAfter)
	}
}

func TestPerKeyIsolation(t *testing.T) {
	lim, _ := newTestLimiter(t)
	ctx := context.Background()

	// User A burns their budget; user B should still be able to make calls.
	for i := 0; i < 2; i++ {
		if _, err := lim.Allow(ctx, "a", 2, time.Minute); err != nil {
			t.Fatalf("user a %d: %v", i, err)
		}
	}
	if _, err := lim.Allow(ctx, "a", 2, time.Minute); err != nil {
		t.Fatalf("user a over-limit: %v", err)
	}

	got, err := lim.Allow(ctx, "b", 2, time.Minute)
	if err != nil {
		t.Fatalf("user b: %v", err)
	}
	if !got.Allowed {
		t.Fatalf("user b should not share a's bucket")
	}
}

func TestWindowResetsAfterExpiry(t *testing.T) {
	lim, mr := newTestLimiter(t)
	ctx := context.Background()

	// Burn the budget.
	for i := 0; i < 3; i++ {
		if _, err := lim.Allow(ctx, "roll", 3, time.Minute); err != nil {
			t.Fatalf("warm-up %d: %v", i, err)
		}
	}
	if got, _ := lim.Allow(ctx, "roll", 3, time.Minute); got.Allowed {
		t.Fatalf("expected denial after budget exhausted")
	}

	// Fast-forward past the window; the next call should get a fresh bucket.
	mr.FastForward(2 * time.Minute)
	got, err := lim.Allow(ctx, "roll", 3, time.Minute)
	if err != nil {
		t.Fatalf("post-rollover: %v", err)
	}
	if !got.Allowed {
		t.Fatalf("expected allowed after window rollover, remaining=%d", got.Remaining)
	}
}

func TestNilLimiterAlwaysAllows(t *testing.T) {
	// Callers pass nil when Redis isn't configured — Allow must not panic.
	var lim *Limiter
	got, err := lim.Allow(context.Background(), "k", 5, time.Minute)
	if err != nil {
		t.Fatalf("nil limiter: %v", err)
	}
	if !got.Allowed {
		t.Fatalf("nil limiter should fail-open")
	}
}
