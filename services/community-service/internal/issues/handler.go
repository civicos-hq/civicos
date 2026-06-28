package issues

import (
	"errors"
	"net/http"

	"github.com/civicos/community-service/internal/domain"
	"github.com/civicos/community-service/pkg/response"
	"github.com/gin-gonic/gin"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, auth gin.HandlerFunc) {
	rg.GET("", h.list)
	rg.GET("/:id", h.get)
	rg.POST("", auth, h.create)
	rg.POST("/:id/upvote", auth, h.upvote)
	rg.PATCH("/:id/status", auth, h.updateStatus)
}

func (h *Handler) list(c *gin.Context) {
	items, err := h.svc.List(c.Query("communityId"), c.Query("status"))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch issues")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"issues": items})
}

func (h *Handler) get(c *gin.Context) {
	item, err := h.svc.Get(c.Param("id"))
	if err != nil {
		var appErr *AppError
		if errors.As(err, &appErr) {
			response.Error(c, appErr.Status, appErr.Code, appErr.Message)
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch issue")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"issue": item})
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
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create issue")
		return
	}
	response.Success(c, http.StatusCreated, gin.H{"issue": item})
}

func (h *Handler) upvote(c *gin.Context) {
	if err := h.svc.Upvote(c.Param("id")); err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to upvote")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"upvoted": true})
}

func (h *Handler) updateStatus(c *gin.Context) {
	var body struct {
		Status domain.IssueStatus `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	if err := h.svc.UpdateStatus(c.Param("id"), body.Status); err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update status")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"updated": true})
}
