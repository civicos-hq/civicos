package notifications

import (
	"net/http"
	"strconv"

	"github.com/civicos/community-service/pkg/response"
	"github.com/gin-gonic/gin"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, auth gin.HandlerFunc) {
	rg.GET("", auth, h.list)
	rg.GET("/unread-count", auth, h.unreadCount)
	rg.PATCH("/:id/read", auth, h.markRead)
	rg.POST("/read-all", auth, h.markAllRead)
}

func (h *Handler) list(c *gin.Context) {
	userID, _ := c.Get("userID")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	items, err := h.svc.List(userID.(string), limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch notifications")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"notifications": items})
}

func (h *Handler) unreadCount(c *gin.Context) {
	userID, _ := c.Get("userID")
	n, err := h.svc.UnreadCount(userID.(string))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to count notifications")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"count": n})
}

func (h *Handler) markRead(c *gin.Context) {
	userID, _ := c.Get("userID")
	if err := h.svc.MarkRead(c.Param("id"), userID.(string)); err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to mark as read")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"read": true})
}

func (h *Handler) markAllRead(c *gin.Context) {
	userID, _ := c.Get("userID")
	if err := h.svc.MarkAllRead(userID.(string)); err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to mark all as read")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"read": true})
}
