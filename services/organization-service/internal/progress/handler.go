package progress

import (
	"errors"
	"net/http"

	"github.com/civicos/organization-service/internal/organizations"
	"github.com/civicos/organization-service/pkg/response"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc  *Service
	orgs *organizations.Service
}

func NewHandler(svc *Service, orgs *organizations.Service) *Handler {
	return &Handler{svc: svc, orgs: orgs}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, auth gin.HandlerFunc) {
	// Public reads — anyone can see progress on an issue or project.
	rg.GET("/issues/:issueId/progress-updates", h.listForIssue)
	rg.GET("/projects/:projectId/progress-updates", h.listForProject)

	// Writes require org admin.
	rg.POST("/organizations/:id/progress-updates", auth, h.create)
	rg.DELETE("/progress-updates/:updateId", auth, h.delete)
}

func (h *Handler) listForIssue(c *gin.Context) {
	items, err := h.svc.List(ListFilters{IssueID: c.Param("issueId"), PublicOnly: true})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list updates")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"updates": items})
}

func (h *Handler) listForProject(c *gin.Context) {
	items, err := h.svc.List(ListFilters{ProjectID: c.Param("projectId"), PublicOnly: true})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list updates")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"updates": items})
}

func (h *Handler) create(c *gin.Context) {
	orgID := c.Param("id")
	userID, _ := c.Get("userID")
	userName, _ := c.Get("userName")
	userRole, _ := c.Get("userRole")
	if err := h.orgs.CanAdmin(orgID, userID.(string), asString(userRole)); err != nil {
		handleAppErr(c, err)
		return
	}
	var input CreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	item, err := h.svc.Create(orgID, input, userID.(string), asString(userName))
	if handleAppErr(c, err) {
		return
	}
	response.Success(c, http.StatusCreated, gin.H{"update": item})
}

func (h *Handler) delete(c *gin.Context) {
	id := c.Param("updateId")
	u, err := h.svc.Get(id)
	if handleAppErr(c, err) {
		return
	}
	userID, _ := c.Get("userID")
	userRole, _ := c.Get("userRole")
	if err := h.orgs.CanAdmin(u.OrganizationID, userID.(string), asString(userRole)); err != nil {
		handleAppErr(c, err)
		return
	}
	if err := h.svc.Delete(id); handleAppErr(c, err) {
		return
	}
	response.Success(c, http.StatusOK, gin.H{"ok": true})
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
	var orgErr *organizations.AppError
	if errors.As(err, &orgErr) {
		response.Error(c, orgErr.Status, orgErr.Code, orgErr.Message)
		return true
	}
	response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	return true
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
