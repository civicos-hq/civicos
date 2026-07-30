package milestones

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

// RegisterRoutes mounts milestones under their campaign. As with campaigns,
// Phase 1 exposes no unauthenticated route — the public spend plan ships in
// Phase 4 alongside the transparency dashboard.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, auth gin.HandlerFunc) {
	rg.GET("/campaigns/:campaignId/milestones", auth, h.list)
	rg.POST("/campaigns/:campaignId/milestones", auth, h.create)
	rg.PATCH("/milestones/:milestoneId", auth, h.update)
	rg.DELETE("/milestones/:milestoneId", auth, h.delete)
}

func (h *Handler) list(c *gin.Context) {
	campaignID := c.Param("campaignId")
	campaign, err := h.svc.Campaign(campaignID)
	if handleAppErr(c, err) {
		return
	}
	userID, userRole := actorFrom(c)
	if err := h.orgs.CanReadInternal(campaign.OrganizationID, userID, userRole); err != nil {
		handleAppErr(c, err)
		return
	}
	items, err := h.svc.List(campaignID)
	if handleAppErr(c, err) {
		return
	}
	response.Success(c, http.StatusOK, gin.H{"milestones": items})
}

func (h *Handler) create(c *gin.Context) {
	campaignID := c.Param("campaignId")
	campaign, err := h.svc.Campaign(campaignID)
	if handleAppErr(c, err) {
		return
	}
	userID, userRole := actorFrom(c)
	if err := h.orgs.CanAdmin(campaign.OrganizationID, userID, userRole); err != nil {
		handleAppErr(c, err)
		return
	}
	var input CreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	item, err := h.svc.Create(campaignID, input)
	if handleAppErr(c, err) {
		return
	}
	response.Success(c, http.StatusCreated, gin.H{"milestone": item})
}

func (h *Handler) update(c *gin.Context) {
	item, ok := h.loadForOrgAdmin(c)
	if !ok {
		return
	}
	var input UpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	updated, err := h.svc.Update(item, input)
	if handleAppErr(c, err) {
		return
	}
	response.Success(c, http.StatusOK, gin.H{"milestone": updated})
}

func (h *Handler) delete(c *gin.Context) {
	item, ok := h.loadForOrgAdmin(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(item); handleAppErr(c, err) {
		return
	}
	response.Success(c, http.StatusOK, gin.H{"ok": true})
}

// loadForOrgAdmin resolves the milestone, walks up to its campaign, and
// checks the caller administers the owning org. Returns the milestone ID
// and ok=false having already written the error response.
func (h *Handler) loadForOrgAdmin(c *gin.Context) (string, bool) {
	id := c.Param("milestoneId")
	m, err := h.svc.Get(id)
	if handleAppErr(c, err) {
		return "", false
	}
	campaign, err := h.svc.Campaign(m.CampaignID)
	if handleAppErr(c, err) {
		return "", false
	}
	userID, userRole := actorFrom(c)
	if err := h.orgs.CanAdmin(campaign.OrganizationID, userID, userRole); err != nil {
		handleAppErr(c, err)
		return "", false
	}
	return id, true
}

func actorFrom(c *gin.Context) (userID, userRole string) {
	id, _ := c.Get("userID")
	role, _ := c.Get("userRole")
	return asString(id), asString(role)
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
