package milestones

import (
	"errors"
	"net/http"

	"github.com/civicos/organization-service/internal/domain"
	"github.com/civicos/organization-service/internal/notifications"
	"github.com/civicos/organization-service/internal/organizations"
	"github.com/civicos/organization-service/pkg/response"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc       *Service
	orgs      *organizations.Service
	notifier  Notifier
	audience  Audience
	campaigns CampaignLookup
}

func NewHandler(svc *Service, orgs *organizations.Service) *Handler {
	return &Handler{svc: svc, orgs: orgs}
}

// Notifier is the slice of notifications this package needs.
type Notifier interface {
	EmitMany(userIDs []string, t notifications.NotificationType, title, body string, linkURL *string)
}

// Audience answers who has a stake in a campaign.
type Audience interface {
	Donors(campaignID string) []string
	Stakeholders(campaignID, orgID string) []string
}

// CampaignLookup resolves the campaign a milestone belongs to, so a
// notification can name it and link to it.
type CampaignLookup interface {
	Get(campaignID string) (*domain.Campaign, error)
}

// WithNotifications attaches fan-out for milestone completion. Optional.
func (h *Handler) WithNotifications(n Notifier, a Audience, c CampaignLookup) *Handler {
	h.notifier = n
	h.audience = a
	h.campaigns = c
	return h
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
	before, _ := h.svc.Get(item)
	updated, err := h.svc.Update(item, input)
	if handleAppErr(c, err) {
		return
	}

	// Announce only the CROSSING into COMPLETED. Re-saving an
	// already-complete milestone — fixing a typo in its title, say — must
	// not tell every donor the work was finished again.
	if updated.Status == domain.MilestoneCompleted &&
		(before == nil || before.Status != domain.MilestoneCompleted) {
		h.announceCompleted(updated)
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

// announceCompleted tells the people who funded a milestone that it is done.
// This is the spend plan turning into evidence, which is the whole point of
// publishing one.
func (h *Handler) announceCompleted(m *domain.Milestone) {
	if h.notifier == nil || h.audience == nil || h.campaigns == nil {
		return
	}
	camp, err := h.campaigns.Get(m.CampaignID)
	if err != nil || camp == nil {
		return
	}
	link := "/campaigns/" + camp.Slug
	h.notifier.EmitMany(h.audience.Stakeholders(camp.ID, camp.OrganizationID),
		notifications.TypeMilestoneCompleted,
		"Milestone completed: "+m.Title,
		camp.Title+" has completed a step of its spend plan.",
		&link)
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
