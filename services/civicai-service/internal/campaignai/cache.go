package campaignai

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// Caching, per CivicAI plan principle 5: "cache aggressively — Gemini calls
// cost real money".
//
// Only two of the six surfaces are cacheable, and the split is not arbitrary:
//
//   - summarize-campaign-impact and assess-campaign-risk are pure functions of
//     a campaign's published state. The same campaign asked twice in an hour
//     has the same answer, and both are read repeatedly — an impact summary
//     sits on a page, and a reviewer working a queue re-opens items.
//   - The three drafting surfaces take a free-text brief. Caching them would
//     either key on the brief (near-zero hit rate) or return a stale draft for
//     a changed one. The existing draft handler makes the same call for
//     announcements, deliberately: a second attempt should give the author a
//     fresh variation to compare.
//   - classify-campaign is one keystroke-cheap call during authoring.
//
// TTLs are short. Both cached surfaces describe money, and a summary that
// still says "nothing has been reported" an hour after the organization
// published its accounts is worse than a second API call.
type Cache struct {
	rdb *redis.Client
}

func NewCache(rdb *redis.Client) *Cache { return &Cache{rdb: rdb} }

const (
	impactTTL = 20 * time.Minute
	riskTTL   = 30 * time.Minute
)

// get decodes a cached value into out. Every failure path — no Redis, a miss,
// corrupt JSON — returns false and lets the caller fall through to Gemini. A
// cache must never be able to fail a request.
func (c *Cache) get(ctx context.Context, key string, out any) bool {
	if c == nil || c.rdb == nil {
		return false
	}
	raw, err := c.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return false
	}
	return json.Unmarshal(raw, out) == nil
}

func (c *Cache) set(ctx context.Context, key string, v any, ttl time.Duration) {
	if c == nil || c.rdb == nil {
		return
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return
	}
	_ = c.rdb.Set(ctx, key, raw, ttl).Err()
}

// Keys are namespaced per surface. The risk assessment is admin-only and the
// impact summary is not, so they must never be able to collide on a campaign
// id — an impact summary served from a risk key would put fraud signals about
// an organization onto a page an org admin can read.
func impactKey(campaignID string) string { return "civicai:campaign:impact:" + campaignID }
func riskKey(campaignID string) string   { return "civicai:campaign:risk:" + campaignID }
