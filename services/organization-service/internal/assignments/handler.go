package assignments

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
	rg.GET("/organizations/:id/assignments", auth, h.listByOrg)
	rg.GET("/issues/:issueId/assignments", h.listByIssue)
	rg.POST("/organizations/:id/assignments", auth, h.create)
	rg.PATCH("/assignments/:assignmentId", auth, h.updateStatus)
	rg.DELETE("/assignments/:assignmentId", auth, h.delete)
}

// listByOrg is member-only — the caller must belong to the org they're
// querying. Prevents a curious user from enumerating another org's inbox.
func (h *Handler) listByOrg(c *gin.Context) {
	orgID := c.Param("id")
	userID, _ := c.Get("userID")
	userRole, _ := c.Get("userRole")
	if userRole.(string) != "PLATFORM_ADMIN" {
		if _, err := h.orgs.IsMember(orgID, userID.(string)); err != nil {
			handleAppErr(c, err)
			return
		}
	}
	items, err := h.svc.ListByOrg(orgID, strings.ToUpper(c.Query("status")))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list assignments")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"assignments": items})
}

// listByIssue is public read — anyone viewing an issue should be able to
// see which orgs have claimed it, so citizens know who's responsible.
func (h *Handler) listByIssue(c *gin.Context) {
	items, err := h.svc.ListByIssue(c.Param("issueId"))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list assignments")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"assignments": items})
}

func (h *Handler) create(c *gin.Context) {
	orgID := c.Param("id")
	userID, _ := c.Get("userID")
	userName, _ := c.Get("userName")
	userRole, _ := c.Get("userRole")
	// Two paths in: an org admin claims an issue for their org (self-assign),
	// or a PLATFORM_ADMIN routes an issue on someone's behalf.
	if userRole.(string) != "PLATFORM_ADMIN" {
		if err := h.orgs.CanAdmin(orgID, userID.(string), asString(userRole)); err != nil {
			handleAppErr(c, err)
			return
		}
	}
	var input AssignInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	item, err := h.svc.Create(orgID, input, userID.(string), asString(userName))
	if handleAppErr(c, err) {
		return
	}
	response.Success(c, http.StatusCreated, gin.H{"assignment": item})
}

func (h *Handler) updateStatus(c *gin.Context) {
	id := c.Param("assignmentId")
	a, err := h.svc.Get(id)
	if handleAppErr(c, err) {
		return
	}
	userID, _ := c.Get("userID")
	userRole, _ := c.Get("userRole")
	if err := h.orgs.CanAdmin(a.OrganizationID, userID.(string), asString(userRole)); err != nil {
		handleAppErr(c, err)
		return
	}
	var input StatusInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	item, err := h.svc.UpdateStatus(id, input)
	if handleAppErr(c, err) {
		return
	}
	response.Success(c, http.StatusOK, gin.H{"assignment": item})
}

func (h *Handler) delete(c *gin.Context) {
	id := c.Param("assignmentId")
	a, err := h.svc.Get(id)
	if handleAppErr(c, err) {
		return
	}
	userID, _ := c.Get("userID")
	userRole, _ := c.Get("userRole")
	if err := h.orgs.CanAdmin(a.OrganizationID, userID.(string), asString(userRole)); err != nil {
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
