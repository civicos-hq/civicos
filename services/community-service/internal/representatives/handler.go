package representatives

import (
	"errors"
	"net/http"

	"github.com/civicos/community-service/pkg/response"
	"github.com/gin-gonic/gin"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, auth, requireRole gin.HandlerFunc) {
	rg.GET("", h.list)
	rg.GET("/:id", h.get)
	rg.POST("", auth, requireRole, h.create)
	rg.POST("/:id/follow", auth, h.follow)
	rg.DELETE("/:id/follow", auth, h.unfollow)
}

// RegisterMeRoutes mounts user-scoped follow lookups under /me on the parent router.
func (h *Handler) RegisterMeRoutes(rg *gin.RouterGroup, auth gin.HandlerFunc) {
	rg.GET("/follows/representatives", auth, h.followedIDs)
}

func (h *Handler) list(c *gin.Context) {
	items, err := h.svc.List(c.Query("communityId"))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch representatives")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"representatives": items})
}

func (h *Handler) get(c *gin.Context) {
	item, err := h.svc.Get(c.Param("id"))
	if err != nil {
		var appErr *AppError
		if errors.As(err, &appErr) {
			response.Error(c, appErr.Status, appErr.Code, appErr.Message)
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch representative")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"representative": item})
}

func (h *Handler) follow(c *gin.Context) {
	userID, _ := c.Get("userID")
	if err := h.svc.Follow(c.Param("id"), userID.(string)); err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to follow")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"following": true})
}

func (h *Handler) unfollow(c *gin.Context) {
	userID, _ := c.Get("userID")
	if err := h.svc.Unfollow(c.Param("id"), userID.(string)); err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to unfollow")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"following": false})
}

func (h *Handler) followedIDs(c *gin.Context) {
	userID, _ := c.Get("userID")
	ids, err := h.svc.FollowedIDs(userID.(string))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch follows")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"representativeIds": ids})
}

func (h *Handler) create(c *gin.Context) {
	var input CreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	userID, _ := c.Get("userID")
	item, err := h.svc.Create(input, userID.(string))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create representative")
		return
	}
	response.Success(c, http.StatusCreated, gin.H{"representative": item})
}
