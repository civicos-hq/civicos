package search

import (
	"net/http"
	"strings"

	"github.com/civicos/community-service/internal/domain"
	"github.com/civicos/community-service/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const perBucketLimit = 8

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("", h.search)
}

// Result is a flat per-entity search payload. Frontend renders three lanes by
// the `kind` discriminator without needing entity-specific shapes.
type Result struct {
	Issues          []domain.Issue          `json:"issues"`
	Petitions       []domain.Petition       `json:"petitions"`
	Representatives []domain.Representative `json:"representatives"`
}

func (h *Handler) search(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	if len(q) < 2 {
		response.Success(c, http.StatusOK, gin.H{
			"issues":          []domain.Issue{},
			"petitions":       []domain.Petition{},
			"representatives": []domain.Representative{},
		})
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
	})
}

// Search runs three case-insensitive LIKE queries in parallel-friendly serial
// order. ILIKE is good enough for the dataset sizes this catalog will see
// before we need pg_trgm or full-text indexing.
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

	return &Result{Issues: issues, Petitions: petitions, Representatives: reps}, nil
}
