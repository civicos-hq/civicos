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
	flatScanLimit   = 1000
	defaultPageSize = 20
	maxPageSize     = 50
)

// FeedItem is what the API returns: a discriminated union of the five
// entity kinds with the resolving community/org summary attached so
// the UI doesn't need a second roundtrip.
//
// Exactly one of Issue / Petition / Announcement / Project /
// Consultation is set, keyed by the Kind discriminator. CommunityID
// may be empty for announcements + un-scoped projects/consultations
// (no community anchor).
type FeedItem struct {
	Kind         string            `json:"kind"` // "issue" | "petition" | "announcement" | "project" | "consultation"
	Tier         Tier              `json:"tier"`
	CreatedAt    time.Time         `json:"createdAt"`
	CommunityID  string            `json:"communityId,omitempty"`
	Community    *CommunitySummary `json:"community,omitempty"`
	Organization *OrgSummary       `json:"organization,omitempty"`
	Issue        *domain.Issue     `json:"issue,omitempty"`
	Petition     *domain.Petition  `json:"petition,omitempty"`
	Announcement *Announcement     `json:"announcement,omitempty"`
	Project      *Project          `json:"project,omitempty"`
	Consultation *Consultation     `json:"consultation,omitempty"`
}

type CommunitySummary struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	State string `json:"state"`
	LGA   string `json:"lga"`
}

// OrgSummary is the minimal org context we ship with announcements +
// projects so the citizen frontend can attribute the item without a
// round-trip. Verified flag matters because the badge is a trust signal.
type OrgSummary struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Verified bool   `json:"verified"`
	State    string `json:"state,omitempty"`
	LGA      string `json:"lga,omitempty"`
}

// Announcement is a read-only view of the shared announcements table
// (owned by organization-service). We declare it locally so we can
// SELECT from it without importing the org-service module. Same
// shared-DB pattern as the audit + notifications tables.
type Announcement struct {
	ID             string     `json:"id" gorm:"type:uuid;primaryKey"`
	OrganizationID string     `json:"organizationId" gorm:"type:uuid;not null"`
	Title          string     `json:"title"`
	Body           string     `json:"body"`
	Status         string     `json:"status"`
	PublishedAt    *time.Time `json:"publishedAt,omitempty"`
	AuthorID       string     `json:"authorId" gorm:"type:uuid"`
	AuthorName     string     `json:"authorName"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

func (Announcement) TableName() string { return "announcements" }

// Project is the read-only view of the shared projects table. Same
// rationale as Announcement above.
type Project struct {
	ID              string     `json:"id" gorm:"type:uuid;primaryKey"`
	OrganizationID  string     `json:"organizationId" gorm:"type:uuid;not null"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	Status          string     `json:"status"`
	StartDate       *time.Time `json:"startDate,omitempty"`
	ExpectedEndDate *time.Time `json:"expectedEndDate,omitempty"`
	BudgetKobo      *int64     `json:"budgetKobo,omitempty"`
	CommunityID     *string    `json:"communityId,omitempty" gorm:"type:uuid"`
	CreatedByID     string     `json:"createdById" gorm:"type:uuid"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

func (Project) TableName() string { return "projects" }

// Consultation is the read-only view of the shared consultations
// table. Only the fields needed for the citizen feed card are
// declared — question rows, responses, and outcomes are fetched via
// the dedicated /consultations endpoints when the citizen clicks in.
type Consultation struct {
	ID             string     `json:"id" gorm:"type:uuid;primaryKey"`
	OrganizationID string     `json:"organizationId" gorm:"type:uuid;not null"`
	CommunityID    *string    `json:"communityId,omitempty" gorm:"type:uuid"`
	Title          string     `json:"title"`
	Summary        string     `json:"summary"`
	Status         string     `json:"status"`
	OpensAt        *time.Time `json:"opensAt,omitempty"`
	ClosesAt       *time.Time `json:"closesAt,omitempty"`
	ResponseCount  int        `json:"responseCount"`
	AuthorID       string     `json:"authorId" gorm:"type:uuid"`
	AuthorName     string     `json:"authorName"`
	PublishedAt    *time.Time `json:"publishedAt,omitempty"`
	ClosedAt       *time.Time `json:"closedAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

func (Consultation) TableName() string { return "consultations" }

// organization is the internal read-only view of orgs, used only for
// tier resolution (state/lga fields) and OrgSummary hydration. Not
// exposed in the response; the response uses OrgSummary.
type organization struct {
	ID       string `gorm:"type:uuid;primaryKey"`
	Name     string
	Slug     string
	State    *string
	LGA      *string
	Verified bool
}

func (organization) TableName() string { return "organizations" }

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
	kind := strings.ToLower(c.Query("kind"))
	// Treat any value outside the known kinds as no filter so a typo on
	// the client falls back to the full feed rather than silently empty.
	switch kind {
	case "issue", "petition", "announcement", "project", "consultation":
		// known — keep as-is
	default:
		kind = ""
	}
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

	result, err := h.svc.Feed(c.Query("communityId"), tier, kind, limit, offset)
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
//   - When tierFilter is empty, the response is the curated grouped view: top
//     results per tier up to totalLimit, no pagination cursor.
//   - When tierFilter is set, items are filtered to that tier and paginated via
//     offset/limit, with NextOffset != nil when more items exist.
//   - kindFilter ("issue" | "petition" | "") narrows to a single entity kind.
//     Skips fetching the other entity entirely so we don't waste a SQL round trip.
//
// userCommunityID may be empty — in that case base is nil and every item
// resolves to TierCountry.
func (s *Service) Feed(userCommunityID string, tierFilter Tier, kindFilter string, limit, offset int) (FeedResult, error) {
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

	wantIssues := kindFilter == "" || kindFilter == "issue"
	wantPetitions := kindFilter == "" || kindFilter == "petition"
	wantAnnouncements := kindFilter == "" || kindFilter == "announcement"
	wantProjects := kindFilter == "" || kindFilter == "project"
	wantConsultations := kindFilter == "" || kindFilter == "consultation"

	var issues []domain.Issue
	if wantIssues {
		issues, err = s.recentIssues(scanLimit)
		if err != nil {
			return FeedResult{}, err
		}
	}
	var petitions []domain.Petition
	if wantPetitions {
		petitions, err = s.recentPetitions(scanLimit)
		if err != nil {
			return FeedResult{}, err
		}
	}
	var announcements []Announcement
	if wantAnnouncements {
		announcements, err = s.recentAnnouncements(scanLimit)
		if err != nil {
			return FeedResult{}, err
		}
	}
	var projects []Project
	if wantProjects {
		projects, err = s.recentProjects(scanLimit)
		if err != nil {
			return FeedResult{}, err
		}
	}
	var consultations []Consultation
	if wantConsultations {
		consultations, err = s.recentConsultations(scanLimit)
		if err != nil {
			return FeedResult{}, err
		}
	}

	// Only load orgs if we'll need them for attribution/tiering. Avoids
	// a needless SELECT when the caller filtered to issue/petition only.
	var orgs map[string]*organization
	if wantAnnouncements || wantProjects || wantConsultations {
		orgs, err = s.loadOrganizations()
		if err != nil {
			return FeedResult{}, err
		}
	}

	all := make([]FeedItem, 0, len(issues)+len(petitions)+len(announcements)+len(projects)+len(consultations))
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
	// Announcements — attributed to their org, tiered by the org's
	// state/lga since announcements themselves aren't community-anchored.
	for i := range announcements {
		a := announcements[i]
		org := orgs[a.OrganizationID]
		all = append(all, FeedItem{
			Kind:         "announcement",
			Tier:         tierForOrg(base, org),
			CreatedAt:    firstNonZero(a.PublishedAt, a.CreatedAt),
			Organization: summaryOfOrg(org),
			Announcement: &a,
		})
	}
	// Projects — tier by project.CommunityID if the project is anchored
	// to one, otherwise fall back to the org's state/lga.
	for i := range projects {
		p := projects[i]
		org := orgs[p.OrganizationID]
		item := FeedItem{
			Kind:         "project",
			CreatedAt:    p.CreatedAt,
			Organization: summaryOfOrg(org),
			Project:      &p,
		}
		if p.CommunityID != nil {
			comm := communities[*p.CommunityID]
			item.Tier = tierFor(base, comm)
			item.CommunityID = *p.CommunityID
			item.Community = summaryOf(comm)
		} else {
			item.Tier = tierForOrg(base, org)
		}
		all = append(all, item)
	}
	// Consultations — same tier rules as projects: community anchor if
	// set, else fall back to the org's state/lga. Sort by publishedAt
	// when we have it so newly published consultations lead the feed.
	for i := range consultations {
		c := consultations[i]
		org := orgs[c.OrganizationID]
		item := FeedItem{
			Kind:         "consultation",
			CreatedAt:    firstNonZero(c.PublishedAt, c.CreatedAt),
			Organization: summaryOfOrg(org),
			Consultation: &c,
		}
		if c.CommunityID != nil {
			comm := communities[*c.CommunityID]
			item.Tier = tierFor(base, comm)
			item.CommunityID = *c.CommunityID
			item.Community = summaryOf(comm)
		} else {
			item.Tier = tierForOrg(base, org)
		}
		all = append(all, item)
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

// recentAnnouncements returns published announcements newest first.
// Drafts and archived items are excluded — the citizen Discover feed
// only surfaces content that's currently public.
func (s *Service) recentAnnouncements(limit int) ([]Announcement, error) {
	var list []Announcement
	err := s.db.Where("status = ?", "PUBLISHED").
		Order("COALESCE(published_at, created_at) desc").
		Limit(limit).Find(&list).Error
	return list, err
}

// recentProjects returns projects newest first. Cancelled projects
// are excluded — completed ones are kept because a "here's what we
// built" record is worth citizens seeing.
func (s *Service) recentProjects(limit int) ([]Project, error) {
	var list []Project
	err := s.db.Where("status <> ?", "CANCELLED").
		Order("created_at desc").
		Limit(limit).Find(&list).Error
	return list, err
}

// recentConsultations returns published + closed consultations newest
// first. Drafts are excluded — they're private to the org until
// publish. Closed consultations stay in the feed because their
// outcome pages are the whole point of the "close the loop" primitive.
func (s *Service) recentConsultations(limit int) ([]Consultation, error) {
	var list []Consultation
	err := s.db.Where("status IN ?", []string{"PUBLISHED", "CLOSED"}).
		Order("COALESCE(published_at, created_at) desc").
		Limit(limit).Find(&list).Error
	return list, err
}

// loadOrganizations loads every org so we can attribute announcements
// and projects with the org summary + resolve tiering via org state/lga.
// Same shape as loadCommunities. Small tables at MVP scale.
func (s *Service) loadOrganizations() (map[string]*organization, error) {
	var list []organization
	if err := s.db.Find(&list).Error; err != nil {
		return nil, err
	}
	out := make(map[string]*organization, len(list))
	for i := range list {
		out[list[i].ID] = &list[i]
	}
	return out, nil
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

// tierForOrg resolves an org's proximity to the caller's community
// using the org's state + LGA columns. Announcements and un-scoped
// projects use this. If the org has no state/lga (e.g. NATIONAL
// jurisdiction), we can't beat COUNTRY tier from location alone.
func tierForOrg(base *domain.Community, org *organization) Tier {
	if base == nil || org == nil || org.State == nil {
		return TierCountry
	}
	if *org.State == base.State {
		if org.LGA != nil && *org.LGA == base.LGA {
			return TierLGA
		}
		return TierState
	}
	return TierCountry
}

func summaryOfOrg(o *organization) *OrgSummary {
	if o == nil {
		return nil
	}
	s := OrgSummary{
		ID:       o.ID,
		Name:     o.Name,
		Slug:     o.Slug,
		Verified: o.Verified,
	}
	if o.State != nil {
		s.State = *o.State
	}
	if o.LGA != nil {
		s.LGA = *o.LGA
	}
	return &s
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

// firstNonZero returns *primary if non-nil and non-zero, else fallback.
// Announcements sort by publishedAt when set but fall back to createdAt
// so drafts (with nil publishedAt) don't wedge on time.Time{}.
func firstNonZero(primary *time.Time, fallback time.Time) time.Time {
	if primary != nil && !primary.IsZero() {
		return *primary
	}
	return fallback
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
