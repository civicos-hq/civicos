package adminmetrics

import (
	"net/http"

	"github.com/civicos/identity-service/pkg/response"
	"github.com/gin-gonic/gin"
)

type Handler struct{ repo *Repository }

func NewHandler(repo *Repository) *Handler { return &Handler{repo: repo} }

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, auth, requireAdmin gin.HandlerFunc) {
	rg.GET("/metrics", auth, requireAdmin, h.snapshot)
	rg.GET("/communities/:id/stats", auth, requireAdmin, h.communityStats)
}

func (h *Handler) snapshot(c *gin.Context) {
	m, err := h.repo.Snapshot()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to compute metrics")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"metrics": m})
}

func (h *Handler) communityStats(c *gin.Context) {
	s, err := h.repo.CommunityStats(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to compute community stats")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"stats": s})
}
