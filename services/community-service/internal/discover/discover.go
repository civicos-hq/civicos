package discover

import (
	"net/http"
	"sort"
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
	items, err := h.svc.Feed(c.Query("communityId"))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to load feed")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"items": items})
}

// Feed returns up to totalLimit items ranked by proximity tier and recency.
// userCommunityID may be empty — in that case all items fall into TierCountry.
func (s *Service) Feed(userCommunityID string) ([]FeedItem, error) {
	communities, err := s.loadCommunities()
	if err != nil {
		return nil, err
	}

	var base *domain.Community
	if userCommunityID != "" {
		if c, ok := communities[userCommunityID]; ok {
			base = c
		}
	}

	issues, err := s.recentIssues()
	if err != nil {
		return nil, err
	}
	petitions, err := s.recentPetitions()
	if err != nil {
		return nil, err
	}

	out := make([]FeedItem, 0, len(issues)+len(petitions))
	for i := range issues {
		issue := issues[i]
		comm := communities[issue.CommunityID]
		out = append(out, FeedItem{
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
		out = append(out, FeedItem{
			Kind:        "petition",
			Tier:        tierFor(base, comm),
			CreatedAt:   p.CreatedAt,
			CommunityID: p.CommunityID,
			Community:   summaryOf(comm),
			Petition:    &p,
		})
	}

	// Sort: tier rank ascending, then newest first within each tier.
	sort.SliceStable(out, func(i, j int) bool {
		ri, rj := tierRank(out[i].Tier), tierRank(out[j].Tier)
		if ri != rj {
			return ri < rj
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})

	return capByTier(out, perTierLimit, totalLimit), nil
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

func (s *Service) recentIssues() ([]domain.Issue, error) {
	var list []domain.Issue
	// Cap on the SQL side so even with thousands of items we ship a bounded
	// payload back to the handler.
	err := s.db.Order("created_at desc").Limit(200).Find(&list).Error
	return list, err
}

func (s *Service) recentPetitions() ([]domain.Petition, error) {
	var list []domain.Petition
	err := s.db.Order("created_at desc").Limit(200).Find(&list).Error
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
