package discover

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/civicos/community-service/internal/domain"
	"github.com/civicos/community-service/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Tier labels every feed item with how close it is to the requesting user.
// COMMUNITY = same community as the user; LGA = same LGA, different community;
// STATE = same state, different LGA; COUNTRY = same country, different state.
type Tier string

const (
	TierCommunity Tier = "COMMUNITY"
	TierLGA       Tier = "LGA"
	TierState     Tier = "STATE"
	TierCountry   Tier = "COUNTRY"
)

const (
	perTierLimit = 12
	totalLimit   = 40

	// When filtering to a single tier we lift the per-entity cap so pagination
	// has room to walk. For dataset sizes this app expects in the near term,
	// in-memory filtering is fine — at real scale we'd push the tier filter
	// into SQL using a community_id IN (...) clause.
	flatScanLimit  = 1000
	defaultPageSize = 20
	maxPageSize     = 50
)

// FeedItem is what the API returns: a discriminated union of issue/petition
// with the resolving community summary attached so the UI doesn't need a
// second roundtrip.
type FeedItem struct {
	Kind        string                 `json:"kind"` // "issue" | "petition"
	Tier        Tier                   `json:"tier"`
	CreatedAt   time.Time              `json:"createdAt"`
	CommunityID string                 `json:"communityId"`
	Community   *CommunitySummary      `json:"community,omitempty"`
	Issue       *domain.Issue          `json:"issue,omitempty"`
	Petition    *domain.Petition       `json:"petition,omitempty"`
}

type CommunitySummary struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	State string `json:"state"`
	LGA   string `json:"lga"`
}

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, auth gin.HandlerFunc) {
	rg.GET("/feed", auth, h.feed)
}

func (h *Handler) feed(c *gin.Context) {
	// The user's communityId is held in identity-service, not the JWT, so we
	// trust the caller to pass it as a query param. Empty = no personalization,
	// everything falls to TierCountry sorted by recency.
	tier := Tier(strings.ToUpper(c.Query("tier")))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", strconv.Itoa(defaultPageSize)))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit <= 0 {
		limit = defaultPageSize
	}
	if limit > maxPageSize {
		limit = maxPageSize
	}
	if offset < 0 {
		offset = 0
	}

	result, err := h.svc.Feed(c.Query("communityId"), tier, limit, offset)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to load feed")
		return
	}
	response.Success(c, http.StatusOK, gin.H{
		"items":      result.Items,
		"nextOffset": result.NextOffset,
	})
}

// FeedResult is the paginated wrapper returned by Service.Feed.
type FeedResult struct {
	Items      []FeedItem `json:"items"`
	NextOffset *int       `json:"nextOffset,omitempty"`
}

// Feed returns items ranked by proximity tier and recency.
//
// - When tierFilter is empty, the response is the curated grouped view: top
//   results per tier up to totalLimit, no pagination cursor.
// - When tierFilter is set, items are filtered to that tier and paginated via
//   offset/limit, with NextOffset != nil when more items exist.
//
// userCommunityID may be empty — in that case base is nil and every item
// resolves to TierCountry.
func (s *Service) Feed(userCommunityID string, tierFilter Tier, limit, offset int) (FeedResult, error) {
	communities, err := s.loadCommunities()
	if err != nil {
		return FeedResult{}, err
	}

	var base *domain.Community
	if userCommunityID != "" {
		if c, ok := communities[userCommunityID]; ok {
			base = c
		}
	}

	// When tier filtering, lift the SQL cap so pagination has headroom.
	scanLimit := 200
	if tierFilter != "" {
		scanLimit = flatScanLimit
	}

	issues, err := s.recentIssues(scanLimit)
	if err != nil {
		return FeedResult{}, err
	}
	petitions, err := s.recentPetitions(scanLimit)
	if err != nil {
		return FeedResult{}, err
	}

	all := make([]FeedItem, 0, len(issues)+len(petitions))
	for i := range issues {
		issue := issues[i]
		comm := communities[issue.CommunityID]
		all = append(all, FeedItem{
			Kind:        "issue",
			Tier:        tierFor(base, comm),
			CreatedAt:   issue.CreatedAt,
			CommunityID: issue.CommunityID,
			Community:   summaryOf(comm),
			Issue:       &issue,
		})
	}
	for i := range petitions {
		p := petitions[i]
		comm := communities[p.CommunityID]
		all = append(all, FeedItem{
			Kind:        "petition",
			Tier:        tierFor(base, comm),
			CreatedAt:   p.CreatedAt,
			CommunityID: p.CommunityID,
			Community:   summaryOf(comm),
			Petition:    &p,
		})
	}

	// Sort: tier rank ascending, then newest first within each tier.
	sort.SliceStable(all, func(i, j int) bool {
		ri, rj := tierRank(all[i].Tier), tierRank(all[j].Tier)
		if ri != rj {
			return ri < rj
		}
		return all[i].CreatedAt.After(all[j].CreatedAt)
	})

	if tierFilter == "" {
		// Curated grouped view — no pagination cursor.
		return FeedResult{Items: capByTier(all, perTierLimit, totalLimit)}, nil
	}

	// Single-tier flat view with offset/limit pagination.
	filtered := make([]FeedItem, 0, len(all))
	for _, it := range all {
		if it.Tier == tierFilter {
			filtered = append(filtered, it)
		}
	}

	start := offset
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	page := filtered[start:end]

	var next *int
	if end < len(filtered) {
		v := end
		next = &v
	}
	return FeedResult{Items: page, NextOffset: next}, nil
}

func (s *Service) loadCommunities() (map[string]*domain.Community, error) {
	var list []domain.Community
	if err := s.db.Find(&list).Error; err != nil {
		return nil, err
	}
	out := make(map[string]*domain.Community, len(list))
	for i := range list {
		out[list[i].ID] = &list[i]
	}
	return out, nil
}

func (s *Service) recentIssues(limit int) ([]domain.Issue, error) {
	var list []domain.Issue
	err := s.db.Order("created_at desc").Limit(limit).Find(&list).Error
	return list, err
}

func (s *Service) recentPetitions(limit int) ([]domain.Petition, error) {
	var list []domain.Petition
	err := s.db.Order("created_at desc").Limit(limit).Find(&list).Error
	return list, err
}

func tierFor(base, item *domain.Community) Tier {
	if base == nil || item == nil {
		return TierCountry
	}
	if base.ID == item.ID {
		return TierCommunity
	}
	if base.State == item.State && base.LGA == item.LGA {
		return TierLGA
	}
	if base.State == item.State {
		return TierState
	}
	return TierCountry
}

func tierRank(t Tier) int {
	switch t {
	case TierCommunity:
		return 0
	case TierLGA:
		return 1
	case TierState:
		return 2
	default:
		return 3
	}
}

func summaryOf(c *domain.Community) *CommunitySummary {
	if c == nil {
		return nil
	}
	return &CommunitySummary{ID: c.ID, Name: c.Name, State: c.State, LGA: c.LGA}
}

// capByTier enforces per-tier and overall caps so a noisy community can't
// crowd out other tiers entirely.
func capByTier(items []FeedItem, perTier, total int) []FeedItem {
	counts := map[Tier]int{}
	out := make([]FeedItem, 0, len(items))
	for _, it := range items {
		if counts[it.Tier] >= perTier {
			continue
		}
		out = append(out, it)
		counts[it.Tier]++
		if len(out) >= total {
			break
		}
	}
	return out
}
