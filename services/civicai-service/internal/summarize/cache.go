package summarize

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache stores generated summaries in Redis so a repeated click doesn't burn
// another Gemini call. TTL is short because petition / issue threads accrete
// new comments continuously and stale summaries mislead operators.
type Cache struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewCache(rdb *redis.Client, ttl time.Duration) *Cache {
	return &Cache{rdb: rdb, ttl: ttl}
}

func (c *Cache) key(kind, id string) string {
	return "civicai:summary:" + kind + ":" + id
}

// Get returns the cached summary or nil if absent. Any Redis error is
// swallowed — cache misses degrade to a fresh Gemini call, never a 500.
func (c *Cache) Get(ctx context.Context, kind, id string) *SummaryOutput {
	if c == nil || c.rdb == nil {
		return nil
	}
	raw, err := c.rdb.Get(ctx, c.key(kind, id)).Bytes()
	if err != nil {
		return nil
	}
	var out SummaryOutput
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return &out
}

func (c *Cache) Set(ctx context.Context, kind, id string, out *SummaryOutput) {
	if c == nil || c.rdb == nil || out == nil {
		return
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return
	}
	// Best-effort — a failed cache write is not worth failing the request.
	_ = c.rdb.Set(ctx, c.key(kind, id), raw, c.ttl).Err()
}
