// Package ratelimit implements a per-key fixed-window rate limiter backed
// by Redis. Each Allow call is a single INCR + EXPIRE round-trip; when the
// current count crosses the configured limit the call reports "denied" and
// tells the caller how long to wait before retrying.
//
// Design notes:
//
//   - Fixed windows are simpler than sliding windows and adequate for
//     defense-in-depth at short windows (60s). The classic "boundary burst"
//     issue would let a caller send 2*limit requests around a window flip;
//     we accept that trade for the ~4x drop in Redis ops vs sorted-set
//     sliding windows.
//   - Redis outage → fail OPEN (Allow returns Allowed=true, RetryAfter=0).
//     The alternative is fail-closed, which would take the whole API down
//     the moment Redis blips. In this MVP we prefer availability; the fact
//     that we ping Redis at boot means the operator sees the outage in the
//     logs.
package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Result captures the outcome of a single Allow call. Callers should surface
// Remaining and ResetAt as X-RateLimit-* headers so clients can back off
// intelligently without spinning.
type Result struct {
	Allowed    bool
	Limit      int
	Remaining  int
	RetryAfter time.Duration // zero when Allowed
	ResetAt    time.Time
}

// Limiter is the caller-facing entry point. It's safe for concurrent use.
// A nil *Limiter (i.e. no Redis configured) is a valid value and returns
// Allowed=true for every request — this lets callers wire the middleware
// even in environments without Redis.
type Limiter struct {
	rdb *redis.Client
}

// New constructs a Limiter for the given Redis client. Pass nil to get a
// "no-op" limiter that always allows.
func New(rdb *redis.Client) *Limiter {
	return &Limiter{rdb: rdb}
}

// Allow reports whether one request may proceed under the (limit, window)
// budget for the given key. If the caller has already spent its allowance,
// Result.Allowed is false and RetryAfter tells them how long to wait.
func (l *Limiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (Result, error) {
	if l == nil || l.rdb == nil {
		// Fail-open when misconfigured — better than 500-ing every request.
		return Result{Allowed: true, Limit: limit, Remaining: limit - 1, ResetAt: time.Now().Add(window)}, nil
	}

	// One key per window. When the bucket's time frame rolls over the next
	// INCR creates a fresh key and EXPIRE resets the TTL.
	bucket := fmt.Sprintf("rl:%s:%d", key, time.Now().Unix()/int64(window.Seconds()))

	pipe := l.rdb.TxPipeline()
	incr := pipe.Incr(ctx, bucket)
	pipe.Expire(ctx, bucket, window)
	if _, err := pipe.Exec(ctx); err != nil {
		// Fail-open on Redis errors — see package doc.
		return Result{Allowed: true, Limit: limit, Remaining: limit - 1, ResetAt: time.Now().Add(window)}, nil
	}

	count := int(incr.Val())
	// Reset time = end of the current window bucket. Users get an exact
	// point-in-time to retry, not just a duration.
	windowStart := time.Unix((time.Now().Unix()/int64(window.Seconds()))*int64(window.Seconds()), 0)
	resetAt := windowStart.Add(window)

	if count > limit {
		return Result{
			Allowed:    false,
			Limit:      limit,
			Remaining:  0,
			RetryAfter: time.Until(resetAt),
			ResetAt:    resetAt,
		}, nil
	}
	return Result{
		Allowed:   true,
		Limit:     limit,
		Remaining: limit - count,
		ResetAt:   resetAt,
	}, nil
}

// Connect parses a Redis URL and returns a ready-to-use client. Pings the
// server before returning so the caller knows about network / auth issues
// at boot rather than on the first real request.
func Connect(ctx context.Context, url string) (*redis.Client, error) {
	if url == "" {
		return nil, errors.New("empty Redis URL")
	}
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	rdb := redis.NewClient(opts)
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return rdb, nil
}
