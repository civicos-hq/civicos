package communities

import (
	"errors"
	"net/http"

	"github.com/civicos/community-service/pkg/response"
	"github.com/gin-gonic/gin"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Roles permitted to create a community. Citizens browse and join; they don't author.
var communityCreatorRoles = []string{"GOVERNMENT_ADMIN", "PLATFORM_ADMIN", "NGO"}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, auth, requireRole gin.HandlerFunc) {
	rg.GET("", h.list)
	rg.GET("/:id", h.get)
	rg.POST("", auth, requireRole, h.create)
}

func (h *Handler) list(c *gin.Context) {
	items, err := h.svc.List()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch communities")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"communities": items})
}

func (h *Handler) get(c *gin.Context) {
	item, err := h.svc.Get(c.Param("id"))
	if err != nil {
		var appErr *AppError
		if errors.As(err, &appErr) {
			response.Error(c, appErr.Status, appErr.Code, appErr.Message)
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch community")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"community": item})
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
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create community")
		return
	}
	response.Success(c, http.StatusCreated, gin.H{"community": item})
}
