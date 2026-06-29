package notifications

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/civicos/community-service/pkg/response"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
	hub *Hub
}

func NewHandler(svc *Service, hub *Hub) *Handler { return &Handler{svc: svc, hub: hub} }

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, auth gin.HandlerFunc) {
	rg.GET("", auth, h.list)
	rg.GET("/unread-count", auth, h.unreadCount)
	rg.PATCH("/:id/read", auth, h.markRead)
	rg.POST("/read-all", auth, h.markAllRead)
	rg.GET("/stream", auth, h.stream)
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

// stream upgrades the response into a Server-Sent Events stream and pushes
// every notification created for the authenticated user. Sends a comment-only
// keep-alive every 25s so intermediaries (proxies, load balancers) don't close
// the idle connection.
func (h *Handler) stream(c *gin.Context) {
	if h.hub == nil {
		response.Error(c, http.StatusServiceUnavailable, "NO_STREAM", "Realtime stream is disabled")
		return
	}

	userID, _ := c.Get("userID")
	uid, _ := userID.(string)
	if uid == "" {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Missing user context")
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Flush()

	ch := h.hub.Subscribe(uid)
	defer h.hub.Unsubscribe(uid, ch)

	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	notify := c.Request.Context().Done()
	_, _ = c.Writer.Write([]byte(": connected\n\n"))
	c.Writer.Flush()

	for {
		select {
		case <-notify:
			return
		case <-ticker.C:
			if _, err := c.Writer.Write([]byte(": ping\n\n")); err != nil {
				return
			}
			c.Writer.Flush()
		case n, ok := <-ch:
			if !ok {
				return
			}
			payload, err := json.Marshal(n)
			if err != nil {
				continue
			}
			if _, err := c.Writer.Write([]byte("event: notification\ndata: ")); err != nil {
				return
			}
			if _, err := c.Writer.Write(payload); err != nil {
				return
			}
			if _, err := c.Writer.Write([]byte("\n\n")); err != nil {
				return
			}
			c.Writer.Flush()
		}
	}
}
