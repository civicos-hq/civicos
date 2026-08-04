package analytics

import (
	"net/http"
	"strconv"

	"github.com/civicos/organization-service/pkg/response"
	"github.com/gin-gonic/gin"
)

// OrgAccess is the slice of organizations.Service this package needs, taken
// as an interface so analytics does not depend on the whole thing.
type OrgAccess interface {
	CanReadInternal(orgID, userID, userRole string) error
}

type Handler struct {
	svc  *Service
	orgs OrgAccess
}

func NewHandler(svc *Service, orgs OrgAccess) *Handler {
	return &Handler{svc: svc, orgs: orgs}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, auth gin.HandlerFunc) {
	// Org-scoped. Authorised by CanReadInternal against the owning org — the
	// same gate the org's own campaign list uses, so an organization sees its
	// numbers and nobody else's.
	rg.GET("/organizations/:id/funding-analytics", auth, h.org)
	// Platform-wide. Money across every organization on the platform, so
	// PLATFORM_ADMIN and nothing less.
	rg.GET("/admin/funding-analytics", auth, h.platform)
}

func (h *Handler) org(c *gin.Context) {
	orgID := c.Param("id")
	userID, _ := c.Get("userID")
	userRole, _ := c.Get("userRole")
	if err := h.orgs.CanReadInternal(orgID, asString(userID), asString(userRole)); err != nil {
		// Mirrors how the org's other internal reads answer: the caller
		// either may read this organization or gets the same refusal
		// everywhere.
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "You do not have access to this organization")
		return
	}
	out, err := h.svc.ForOrg(orgID, weeksParam(c))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not build analytics")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"analytics": out})
}

func (h *Handler) platform(c *gin.Context) {
	role, _ := c.Get("userRole")
	if asString(role) != "PLATFORM_ADMIN" {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Platform admins only")
		return
	}
	out, err := h.svc.ForPlatform(weeksParam(c))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not build analytics")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"analytics": out})
}

// weeksParam reads the trend window. An unparseable value falls back to the
// default rather than erroring — a bad query string should not deny an
// operator their dashboard.
func weeksParam(c *gin.Context) int {
	n, err := strconv.Atoi(c.Query("weeks"))
	if err != nil {
		return defaultWeeks
	}
	return n
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}
