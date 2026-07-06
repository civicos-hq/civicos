package applications

import (
	"errors"
	"net/http"

	"github.com/civicos/identity-service/pkg/response"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, auth gin.HandlerFunc) {
	rg.GET("/me", auth, h.getMe)
	rg.PUT("/me/representative", auth, h.upsertRepresentative)
	rg.PUT("/me/organization", auth, h.upsertOrganization)
}

func (h *Handler) getMe(c *gin.Context) {
	userID, _ := c.Get("userID")
	data, err := h.svc.GetMe(userID.(string))
	if handleAppErr(c, err) {
		return
	}
	response.Success(c, http.StatusOK, gin.H{"application": data})
}

func (h *Handler) upsertRepresentative(c *gin.Context) {
	var input RepresentativeApplicationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	userID, _ := c.Get("userID")
	app, err := h.svc.UpsertRepresentative(userID.(string), input)
	if handleAppErr(c, err) {
		return
	}
	response.Success(c, http.StatusOK, gin.H{"representativeApplication": app})
}

func (h *Handler) upsertOrganization(c *gin.Context) {
	var input OrganizationApplicationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	userID, _ := c.Get("userID")
	app, err := h.svc.UpsertOrganization(userID.(string), input)
	if handleAppErr(c, err) {
		return
	}
	response.Success(c, http.StatusOK, gin.H{"organizationApplication": app})
}

func handleAppErr(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		response.Error(c, appErr.Status, appErr.Code, appErr.Message)
		return true
	}
	response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	return true
}
