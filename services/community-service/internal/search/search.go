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
	Issues          []domain.Issue          `json:"issues"`
	Petitions       []domain.Petition       `json:"petitions"`
	Representatives []domain.Representative `json:"representatives"`
	Organizations   []Organization          `json:"organizations"`
	Consultations   []Consultation          `json:"consultations"`
	Announcements   []Announcement          `json:"announcements"`
	Projects        []Project               `json:"projects"`
}

func emptyResult() gin.H {
	return gin.H{
		"issues":          []domain.Issue{},
		"petitions":       []domain.Petition{},
		"representatives": []domain.Representative{},
		"organizations":   []Organization{},
		"consultations":   []Consultation{},
		"announcements":   []Announcement{},
		"projects":        []Project{},
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
		"issues":          res.Issues,
		"petitions":       res.Petitions,
		"representatives": res.Representatives,
		"organizations":   res.Organizations,
		"consultations":   res.Consultations,
		"announcements":   res.Announcements,
		"projects":        res.Projects,
	})
}

// Search runs seven case-insensitive LIKE queries. ILIKE is good enough
// for the dataset sizes this catalog will see before we need pg_trgm or
// full-text indexing (tracked on the roadmap as "Full-text search").
//
// Visibility rules match the citizen browse pages:
//   - Consultations: DRAFT is hidden (author-only visibility).
//   - Announcements: only PUBLISHED (drafts + archived are not
//     citizen-facing).
//   - Projects: all statuses render on the citizen browse, so search
//     matches all statuses too.
//   - Organizations: no status field; the registry is fully public.
func (s *Service) Search(q string) (*Result, error) {
	like := "%" + strings.ReplaceAll(strings.ReplaceAll(q, `\`, `\\`), `%`, `\%`) + "%"

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

	return &Result{
		Issues:          issues,
		Petitions:       petitions,
		Representatives: reps,
		Organizations:   orgs,
		Consultations:   consultations,
		Announcements:   announcements,
		Projects:        projects,
	}, nil
}
