package auditlogs

import (
	"net/http"
	"strconv"
	"time"

	"github.com/civicos/identity-service/pkg/response"
	"github.com/gin-gonic/gin"
)

type Handler struct{ repo *Repository }

func NewHandler(repo *Repository) *Handler { return &Handler{repo: repo} }

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, auth, requireAdmin gin.HandlerFunc) {
	rg.GET("", auth, requireAdmin, h.list)
}

func (h *Handler) list(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))

	var since, until *time.Time
	if s := c.Query("since"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			since = &t
		}
	}
	if u := c.Query("until"); u != "" {
		if t, err := time.Parse(time.RFC3339, u); err == nil {
			until = &t
		}
	}

	entries, total, err := h.repo.Find(ListFilters{
		ActorID:    c.Query("actorId"),
		Action:     c.Query("action"),
		TargetType: c.Query("targetType"),
		TargetID:   c.Query("targetId"),
		Since:      since,
		Until:      until,
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list audit logs")
		return
	}
	response.Success(c, http.StatusOK, gin.H{
		"auditLogs": entries,
		"total":     total,
	})
}
