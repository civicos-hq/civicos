package projects

import (
	"errors"
	"net/http"
	"strings"

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
	rg.GET("/projects", h.list)
	rg.GET("/organizations/:id/projects", h.listByOrg)
	rg.GET("/projects/:projectId", h.get)

	rg.POST("/organizations/:id/projects", auth, h.create)
	rg.PATCH("/projects/:projectId", auth, h.update)
	rg.DELETE("/projects/:projectId", auth, h.delete)
}

func (h *Handler) list(c *gin.Context) {
	items, err := h.svc.List(ListFilters{
		OrgID:       c.Query("organizationId"),
		CommunityID: c.Query("communityId"),
		Status:      strings.ToUpper(c.Query("status")),
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list projects")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"projects": items})
}

func (h *Handler) listByOrg(c *gin.Context) {
	items, err := h.svc.List(ListFilters{
		OrgID:  c.Param("id"),
		Status: strings.ToUpper(c.Query("status")),
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list projects")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"projects": items})
}

func (h *Handler) get(c *gin.Context) {
	item, err := h.svc.Get(c.Param("projectId"))
	if handleAppErr(c, err) {
		return
	}
	response.Success(c, http.StatusOK, gin.H{"project": item})
}

func (h *Handler) create(c *gin.Context) {
	orgID := c.Param("id")
	userID, _ := c.Get("userID")
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
	item, err := h.svc.Create(orgID, input, userID.(string))
	if handleAppErr(c, err) {
		return
	}
	response.Success(c, http.StatusCreated, gin.H{"project": item})
}

func (h *Handler) update(c *gin.Context) {
	id := c.Param("projectId")
	p, err := h.svc.Get(id)
	if handleAppErr(c, err) {
		return
	}
	userID, _ := c.Get("userID")
	userRole, _ := c.Get("userRole")
	if err := h.orgs.CanAdmin(p.OrganizationID, userID.(string), asString(userRole)); err != nil {
		handleAppErr(c, err)
		return
	}
	var input UpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	item, err := h.svc.Update(id, input)
	if handleAppErr(c, err) {
		return
	}
	response.Success(c, http.StatusOK, gin.H{"project": item})
}

func (h *Handler) delete(c *gin.Context) {
	id := c.Param("projectId")
	p, err := h.svc.Get(id)
	if handleAppErr(c, err) {
		return
	}
	userID, _ := c.Get("userID")
	userRole, _ := c.Get("userRole")
	if err := h.orgs.CanAdmin(p.OrganizationID, userID.(string), asString(userRole)); err != nil {
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
