package search

import (
	"net/http"
	"strings"
	"time"

	"github.com/civicos/community-service/internal/domain"
	"github.com/civicos/community-service/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const perBucketLimit = 8

// Announcement / Project / Consultation / Organization are read-only
// views of tables owned by organization-service. Same shared-DB pattern
// used by the Discover feed — declaring the shape here keeps search
// self-contained without needing to import the org-service module.

type Announcement struct {
	ID             string    `json:"id" gorm:"type:uuid;primaryKey"`
	OrganizationID string    `json:"organizationId" gorm:"type:uuid;not null"`
	Title          string    `json:"title"`
	Body           string    `json:"body"`
	Status         string    `json:"status"`
	AuthorName     string    `json:"authorName"`
	CreatedAt      time.Time `json:"createdAt"`
}

func (Announcement) TableName() string { return "announcements" }

type Project struct {
	ID             string    `json:"id" gorm:"type:uuid;primaryKey"`
	OrganizationID string    `json:"organizationId" gorm:"type:uuid;not null"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"createdAt"`
}

func (Project) TableName() string { return "projects" }

type Consultation struct {
	ID             string    `json:"id" gorm:"type:uuid;primaryKey"`
	OrganizationID string    `json:"organizationId" gorm:"type:uuid;not null"`
	Title          string    `json:"title"`
	Summary        string    `json:"summary"`
	Status         string    `json:"status"`
	ResponseCount  int       `json:"responseCount"`
	CreatedAt      time.Time `json:"createdAt"`
}

func (Consultation) TableName() string { return "consultations" }

type Organization struct {
	ID           string    `json:"id" gorm:"type:uuid;primaryKey"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	Kind         string    `json:"kind"`
	Jurisdiction string    `json:"jurisdiction"`
	Description  *string   `json:"description,omitempty"`
	Verified     bool      `json:"verified"`
	CreatedAt    time.Time `json:"createdAt"`
}

func (Organization) TableName() string { return "organizations" }

// Campaign carries slug, cover image and progress because a search result
// for a fundraiser is close to useless without them — "how far along is it"
// is the first thing anyone wants to know.
type Campaign struct {
	ID             string    `json:"id" gorm:"type:uuid;primaryKey"`
	OrganizationID string    `json:"organizationId" gorm:"type:uuid;not null"`
	Slug           string    `json:"slug"`
	Title          string    `json:"title"`
	Summary        string    `json:"summary"`
	Category       string    `json:"category"`
	Status         string    `json:"status"`
	Currency       string    `json:"currency"`
	GoalMinor      int64     `json:"goalMinor"`
	RaisedMinor    int64     `json:"raisedMinor"`
	DonorCount     int       `json:"donorCount"`
	CoverImageURL  *string   `json:"coverImageUrl,omitempty"`
	IsEmergency    bool      `json:"isEmergency"`
	State          *string   `json:"state,omitempty"`
	LGA            *string   `json:"lga,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
}

func (Campaign) TableName() string { return "campaigns" }

// RepAnnouncement carries the representative's name so a result is
// attributable without a second lookup — "who said this" is most of what
// makes it worth finding.
type RepAnnouncement struct {
	ID                 string     `json:"id"`
	RepresentativeID   string     `json:"representativeId"`
	RepresentativeName string     `json:"representativeName"`
	Title              string     `json:"title"`
	Body               string     `json:"body"`
	CommunityID        string     `json:"communityId"`
	CommentCount       int        `json:"commentCount"`
	PublishedAt        *time.Time `json:"publishedAt,omitempty"`
}

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("", h.search)
}

// Result is a flat per-entity search payload. Frontend renders one lane
// per bucket by the map key without needing entity-specific shapes.
type Result struct {
	// Communities lead the struct because they are the one result kind that
	// is an entry point rather than a destination: someone searching
	// "University of Abuja" is almost always trying to join it, not to read
	// a specific issue inside it.
	Communities      []domain.Community      `json:"communities"`
	Issues           []domain.Issue          `json:"issues"`
	Petitions        []domain.Petition       `json:"petitions"`
	Representatives  []domain.Representative `json:"representatives"`
	Organizations    []Organization          `json:"organizations"`
	Consultations    []Consultation          `json:"consultations"`
	Announcements    []Announcement          `json:"announcements"`
	Projects         []Project               `json:"projects"`
	Campaigns        []Campaign              `json:"campaigns"`
	RepAnnouncements []RepAnnouncement       `json:"repAnnouncements"`
}

func emptyResult() gin.H {
	return gin.H{
		"communities":      []domain.Community{},
		"issues":           []domain.Issue{},
		"petitions":        []domain.Petition{},
		"representatives":  []domain.Representative{},
		"organizations":    []Organization{},
		"consultations":    []Consultation{},
		"announcements":    []Announcement{},
		"projects":         []Project{},
		"campaigns":        []Campaign{},
		"repAnnouncements": []RepAnnouncement{},
	}
}

func (h *Handler) search(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	if len(q) < 2 {
		response.Success(c, http.StatusOK, emptyResult())
		return
	}

	res, err := h.svc.Search(q)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Search failed")
		return
	}
	response.Success(c, http.StatusOK, gin.H{
		"communities":      res.Communities,
		"issues":           res.Issues,
		"petitions":        res.Petitions,
		"representatives":  res.Representatives,
		"organizations":    res.Organizations,
		"consultations":    res.Consultations,
		"announcements":    res.Announcements,
		"projects":         res.Projects,
		"campaigns":        res.Campaigns,
		"repAnnouncements": res.RepAnnouncements,
	})
}

// Search runs ten case-insensitive LIKE queries. ILIKE is good enough
// for the dataset sizes this catalog will see before we need pg_trgm or
// full-text indexing (tracked on the roadmap as "Full-text search").
//
// Visibility rules match the citizen browse pages:
//   - Communities: no status field; civic geography is fully public, and
//     a community nobody can find is a community nobody can join.
//   - Consultations: DRAFT is hidden (author-only visibility).
//   - Announcements: only PUBLISHED (drafts + archived are not
//     citizen-facing).
//   - Projects: all statuses render on the citizen browse, so search
//     matches all statuses too.
//   - Organizations: no status field; the registry is fully public.
//   - Campaigns: exactly the statuses organization-service treats as
//     citizen-visible (its `publicStatuses` allow-list). DRAFT,
//     PENDING_REVIEW and REJECTED have no public page, and surfacing the
//     title would leak that an organization asked for money and was
//     refused. PAUSED is excluded for a different reason: its page 404s
//     too, so returning it here would be a result that goes nowhere.
//     If that list ever changes, change it there first — this mirrors it.
//   - Representative announcements: PUBLISHED only. A draft was never public
//     and an archived one has been withdrawn by the person who said it —
//     surfacing either would put words back in front of people that the
//     representative has not, or no longer has, stood behind.
func (s *Service) Search(q string) (*Result, error) {
	like := "%" + strings.ReplaceAll(strings.ReplaceAll(q, `\`, `\\`), `%`, `\%`) + "%"

	// Communities match on place names as well as their own name, so
	// searching "Gwagwalada" surfaces both the LGA community and the
	// university sitting inside it. Prefix matches lead, for the same
	// reason they do on the browse endpoint.
	var communities []domain.Community
	if err := s.db.
		Where("name ILIKE ? OR description ILIKE ? OR lga ILIKE ? OR state ILIKE ?", like, like, like, like).
		Order(gorm.Expr("CASE WHEN name ILIKE ? THEN 0 ELSE 1 END", strings.ReplaceAll(strings.ReplaceAll(q, `\`, `\\`), `%`, `\%`)+"%")).
		Order("name asc").
		Limit(perBucketLimit).
		Find(&communities).Error; err != nil {
		return nil, err
	}
	if err := s.attachCommunityMemberCounts(communities); err != nil {
		return nil, err
	}

	var issues []domain.Issue
	if err := s.db.
		Where("title ILIKE ? OR description ILIKE ?", like, like).
		Order("created_at desc").
		Limit(perBucketLimit).
		Find(&issues).Error; err != nil {
		return nil, err
	}

	var petitions []domain.Petition
	if err := s.db.
		Where("title ILIKE ? OR description ILIKE ?", like, like).
		Order("created_at desc").
		Limit(perBucketLimit).
		Find(&petitions).Error; err != nil {
		return nil, err
	}

	var reps []domain.Representative
	if err := s.db.
		Where("name ILIKE ? OR position ILIKE ? OR constituency ILIKE ?", like, like, like).
		Order("name asc").
		Limit(perBucketLimit).
		Find(&reps).Error; err != nil {
		return nil, err
	}

	var orgs []Organization
	if err := s.db.
		Where("name ILIKE ? OR description ILIKE ? OR slug ILIKE ?", like, like, like).
		Order("name asc").
		Limit(perBucketLimit).
		Find(&orgs).Error; err != nil {
		return nil, err
	}

	var consultations []Consultation
	if err := s.db.
		Where("(title ILIKE ? OR summary ILIKE ?) AND status <> 'DRAFT'", like, like).
		Order("created_at desc").
		Limit(perBucketLimit).
		Find(&consultations).Error; err != nil {
		return nil, err
	}

	var announcements []Announcement
	if err := s.db.
		Where("(title ILIKE ? OR body ILIKE ?) AND status = 'PUBLISHED'", like, like).
		Order("created_at desc").
		Limit(perBucketLimit).
		Find(&announcements).Error; err != nil {
		return nil, err
	}

	var projects []Project
	if err := s.db.
		Where("title ILIKE ? OR description ILIKE ?", like, like).
		Order("created_at desc").
		Limit(perBucketLimit).
		Find(&projects).Error; err != nil {
		return nil, err
	}

	var campaigns []Campaign
	if err := s.db.
		Where("(title ILIKE ? OR summary ILIKE ? OR description ILIKE ?) AND status IN ?",
			like, like, like,
			[]string{"PUBLISHED", "FUNDED", "COMPLETED", "REPORTED"}).
		Order("created_at desc").
		Limit(perBucketLimit).
		Find(&campaigns).Error; err != nil {
		return nil, err
	}

	repAnns := []RepAnnouncement{}
	if err := s.db.Table("representative_announcements AS ra").
		Select("ra.id, ra.representative_id, r.name AS representative_name, ra.title, ra.body, ra.community_id, ra.comment_count, ra.published_at").
		Joins("JOIN representatives r ON r.id = ra.representative_id").
		Where("(ra.title ILIKE ? OR ra.body ILIKE ?) AND ra.status = ?", like, like, "PUBLISHED").
		Order("ra.published_at desc").
		Limit(perBucketLimit).
		Scan(&repAnns).Error; err != nil {
		return nil, err
	}

	return &Result{
		Communities:      communities,
		Issues:           issues,
		Petitions:        petitions,
		Representatives:  reps,
		Organizations:    orgs,
		Consultations:    consultations,
		Announcements:    announcements,
		Projects:         projects,
		Campaigns:        campaigns,
		RepAnnouncements: repAnns,
	}, nil
}

// attachCommunityMemberCounts fills the read-only MemberCount on a bucket of
// community results. Mirrors the communities repository rather than sharing
// it: search reaches the database directly and has no repository to borrow.
//
// The count is what makes a community result actionable — with four
// universities in one city, "which of these is actually in use" is the only
// thing distinguishing them at a glance.
func (s *Service) attachCommunityMemberCounts(list []domain.Community) error {
	if len(list) == 0 {
		return nil
	}
	ids := make([]string, 0, len(list))
	for i := range list {
		ids = append(ids, list[i].ID)
	}

	var rows []struct {
		CommunityID string
		Total       int
	}
	if err := s.db.
		Table("user_community_memberships").
		Select("community_id, COUNT(*) AS total").
		Where("community_id IN ?", ids).
		Group("community_id").
		Scan(&rows).Error; err != nil {
		return err
	}

	counts := make(map[string]int, len(rows))
	for _, row := range rows {
		counts[row.CommunityID] = row.Total
	}
	for i := range list {
		list[i].MemberCount = counts[list[i].ID]
	}
	return nil
}
