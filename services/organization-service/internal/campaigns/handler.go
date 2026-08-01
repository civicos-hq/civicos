package campaigns

import (
	"errors"
	"net/http"
	"strings"

	"github.com/civicos/organization-service/internal/audit"
	"github.com/civicos/organization-service/internal/domain"
	"github.com/civicos/organization-service/internal/notifications"
	"github.com/civicos/organization-service/internal/organizations"
	"github.com/civicos/organization-service/pkg/response"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc      *Service
	orgs     *organizations.Service
	auditor  *audit.Auditor
	notifier Notifier
	audience Audience
}

func NewHandler(svc *Service, orgs *organizations.Service, auditor *audit.Auditor) *Handler {
	return &Handler{svc: svc, orgs: orgs, auditor: auditor}
}

// Notifier is the slice of notifications this package needs. Declared here
// rather than imported wholesale so the campaign lifecycle does not depend
// on how notifications are delivered.
type Notifier interface {
	EmitMany(userIDs []string, t notifications.NotificationType, title, body string, linkURL *string)
}

// Audience answers "who has a stake in this campaign".
type Audience interface {
	OrgMembers(orgID string) []string
	Donors(campaignID string) []string
	Stakeholders(campaignID, orgID string) []string
}

// WithNotifications attaches notification fan-out. Optional: a campaign must
// still be reviewable and completable when notifications are unavailable.
func (h *Handler) WithNotifications(n Notifier, a Audience) *Handler {
	h.notifier = n
	h.audience = a
	return h
}

// notify fans out, tolerating a missing notifier. A notification miss must
// never fail the lifecycle action that triggered it.
func (h *Handler) notify(userIDs []string, t notifications.NotificationType, title, body, link string) {
	if h.notifier == nil || len(userIDs) == 0 {
		return
	}
	h.notifier.EmitMany(userIDs, t, title, body, &link)
}

// RegisterRoutes mounts on v1 because campaign URLs span two resource roots
// (organization-scoped creation, campaign-scoped everything else) — same
// reason projects and announcements mount here.
//
// Phase 2 adds the first PUBLIC routes: a citizen can browse published
// campaigns and read one by slug, seeing the goal and the spend plan with
// ₦0 raised. That ordering is deliberate — the transparency surface exists
// before any money can move through it. Phase 4 extends the same pages with
// the full funds-flow dashboard (received / withdrawn / remaining, reports).
//
// Public reads go through a DTO (see public.go), never the domain model, so
// the review trail cannot leak. Everything else still requires membership of
// the owning org or platform admin.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, auth gin.HandlerFunc) {
	// Public, unauthenticated.
	rg.GET("/campaigns", h.listPublic)
	rg.GET("/campaigns/slug/:slug", h.getPublic)

	rg.GET("/organizations/:id/campaigns", auth, h.listByOrg)
	rg.POST("/organizations/:id/campaigns", auth, h.create)

	rg.GET("/campaigns/:campaignId", auth, h.get)
	rg.PATCH("/campaigns/:campaignId", auth, h.update)
	rg.DELETE("/campaigns/:campaignId", auth, h.delete)

	// Org-driven lifecycle.
	rg.POST("/campaigns/:campaignId/submit", auth, h.submit)
	rg.POST("/campaigns/:campaignId/publish", auth, h.publish)
	rg.POST("/campaigns/:campaignId/complete", auth, h.complete)
	rg.POST("/campaigns/:campaignId/report", auth, h.report)

	// Platform-admin review + governance.
	rg.GET("/admin/campaigns", auth, h.adminList)
	rg.POST("/campaigns/:campaignId/review", auth, h.review)
	rg.POST("/campaigns/:campaignId/pause", auth, h.pause)
	rg.POST("/campaigns/:campaignId/resume", auth, h.resume)
	rg.POST("/campaigns/:campaignId/archive", auth, h.archive)
}

// ─── Public reads ───────────────────────────────────────────────────────

func (h *Handler) listPublic(c *gin.Context) {
	items, err := h.svc.ListPublic(ListFilters{
		Category:      strings.ToUpper(c.Query("category")),
		CommunityID:   c.Query("communityId"),
		State:         c.Query("state"),
		LGA:           c.Query("lga"),
		OrgID:         c.Query("organizationId"),
		EmergencyOnly: c.Query("emergency") == "true",
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list campaigns")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"campaigns": items})
}

func (h *Handler) getPublic(c *gin.Context) {
	item, err := h.svc.GetPublicBySlug(c.Param("slug"))
	if handleAppErr(c, err) {
		return
	}
	response.Success(c, http.StatusOK, gin.H{"campaign": item})
}

// ─── Member reads ───────────────────────────────────────────────────────

func (h *Handler) listByOrg(c *gin.Context) {
	orgID := c.Param("id")
	userID, userRole := actorFrom(c)
	// CanReadInternal, not CanAdmin: STAFF should be able to see their org's
	// campaigns without being able to change them.
	if err := h.orgs.CanReadInternal(orgID, userID, userRole); err != nil {
		handleAppErr(c, err)
		return
	}
	items, err := h.svc.List(ListFilters{
		OrgID:    orgID,
		Status:   strings.ToUpper(c.Query("status")),
		Category: strings.ToUpper(c.Query("category")),
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list campaigns")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"campaigns": items})
}

// adminList is the platform review queue. `emergency=true` surfaces the
// spec's fast path for campaigns that cannot wait days for review.
func (h *Handler) adminList(c *gin.Context) {
	_, userRole := actorFrom(c)
	if userRole != "PLATFORM_ADMIN" {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Platform admins only")
		return
	}
	status := strings.ToUpper(c.Query("status"))
	if status == "" {
		status = string(domain.CampaignPendingReview)
	}
	items, err := h.svc.List(ListFilters{
		Status:        status,
		Category:      strings.ToUpper(c.Query("category")),
		EmergencyOnly: c.Query("emergency") == "true",
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list campaigns")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"campaigns": items})
}

func (h *Handler) get(c *gin.Context) {
	item, err := h.svc.Get(c.Param("campaignId"))
	if handleAppErr(c, err) {
		return
	}
	userID, userRole := actorFrom(c)
	if err := h.orgs.CanReadInternal(item.OrganizationID, userID, userRole); err != nil {
		handleAppErr(c, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"campaign": item})
}

// ─── Writes ─────────────────────────────────────────────────────────────

func (h *Handler) create(c *gin.Context) {
	orgID := c.Param("id")
	userID, userRole := actorFrom(c)
	if err := h.orgs.CanAdmin(orgID, userID, userRole); err != nil {
		handleAppErr(c, err)
		return
	}
	var input CreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	userName, _ := c.Get("userName")
	item, err := h.svc.Create(orgID, input, userID, asString(userName))
	if handleAppErr(c, err) {
		return
	}
	h.record(c, "campaign.created", item.ID, map[string]any{
		"organizationId": orgID,
		"title":          item.Title,
		"goalMinor":      item.GoalMinor,
		"currency":       item.Currency,
	})
	response.Success(c, http.StatusCreated, gin.H{"campaign": item})
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
	updated, err := h.svc.Update(item.ID, input)
	if handleAppErr(c, err) {
		return
	}
	response.Success(c, http.StatusOK, gin.H{"campaign": updated})
}

func (h *Handler) delete(c *gin.Context) {
	item, ok := h.loadForOrgAdmin(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(item.ID); handleAppErr(c, err) {
		return
	}
	h.record(c, "campaign.deleted", item.ID, map[string]any{"title": item.Title})
	response.Success(c, http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) submit(c *gin.Context) {
	item, ok := h.loadForOrgAdmin(c)
	if !ok {
		return
	}
	updated, err := h.svc.Submit(item.ID)
	if handleAppErr(c, err) {
		return
	}
	h.record(c, "campaign.submitted", item.ID, map[string]any{"title": item.Title})
	response.Success(c, http.StatusOK, gin.H{"campaign": updated})
}

func (h *Handler) publish(c *gin.Context) {
	item, ok := h.loadForOrgAdmin(c)
	if !ok {
		return
	}
	updated, err := h.svc.Publish(item.ID)
	if handleAppErr(c, err) {
		return
	}
	h.record(c, "campaign.published", item.ID, map[string]any{"title": item.Title})
	response.Success(c, http.StatusOK, gin.H{"campaign": updated})
}

func (h *Handler) complete(c *gin.Context) {
	item, ok := h.loadForOrgAdmin(c)
	if !ok {
		return
	}
	updated, err := h.svc.Transition(item.ID, domain.CampaignCompleted, ActorOrg)
	if handleAppErr(c, err) {
		return
	}
	h.record(c, "campaign.completed", item.ID, nil)

	// Donors first, then the org — Stakeholders preserves that order, so the
	// people who paid for the work are the ones this is really addressed to.
	if h.audience != nil {
		h.notify(h.audience.Stakeholders(updated.ID, updated.OrganizationID),
			notifications.TypeCampaignCompleted,
			"Campaign completed: "+updated.Title,
			"The organization has marked this campaign complete. Its final report is due before it can be archived.",
			"/campaigns/"+updated.Slug)
	}
	response.Success(c, http.StatusOK, gin.H{"campaign": updated})
}

func (h *Handler) report(c *gin.Context) {
	item, ok := h.loadForOrgAdmin(c)
	if !ok {
		return
	}
	var input ReportInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	updated, err := h.svc.FileReport(item.ID, input)
	if handleAppErr(c, err) {
		return
	}
	// The shortfall at filing time is audited too: "who said the work was
	// finished, when, and how much was still unexplained" needs to be
	// answerable without reading the campaign row later.
	h.record(c, "campaign.reported", item.ID, map[string]any{
		"attachments":              len(updated.FinalReportURLs),
		"unaccountedAtReportMinor": updated.UnaccountedAtReportMinor,
	})

	// Everyone who funded it should hear that the account is closed.
	if h.audience != nil {
		h.notify(h.audience.Stakeholders(updated.ID, updated.OrganizationID),
			notifications.TypeCampaignCompleted,
			"Final report published: "+updated.Title,
			"The organization has published its account of what this campaign delivered.",
			"/campaigns/"+updated.Slug)
	}
	response.Success(c, http.StatusOK, gin.H{"campaign": updated})
}

// ─── Platform admin ─────────────────────────────────────────────────────

type reviewInput struct {
	Decision string  `json:"decision" binding:"required"`
	Note     *string `json:"note"`
}

func (h *Handler) review(c *gin.Context) {
	userID, ok := h.requirePlatformAdmin(c)
	if !ok {
		return
	}
	var input reviewInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	updated, err := h.svc.Review(c.Param("campaignId"), input.Decision, input.Note, userID)
	if handleAppErr(c, err) {
		return
	}
	h.record(c, "campaign.reviewed", updated.ID, map[string]any{
		"decision": strings.ToUpper(input.Decision),
		"status":   updated.Status,
	})

	// Only an approval is announced, and only to the organization.
	// NEEDS_CHANGES and REJECTED carry the reviewer's note, which is a
	// private conversation between the platform and the org — the same
	// reasoning that keeps reviewNote out of the public DTO. The org learns
	// those outcomes in its own console, not in a fan-out.
	if updated.Status == domain.CampaignApproved && h.audience != nil {
		h.notify(h.audience.OrgMembers(updated.OrganizationID),
			notifications.TypeCampaignApproved,
			"Campaign approved: "+updated.Title,
			"Your campaign passed review and can now be published.",
			"/campaigns/"+updated.Slug)
	}
	response.Success(c, http.StatusOK, gin.H{"campaign": updated})
}

type pauseInput struct {
	ReasonCode string `json:"reasonCode" binding:"required"`
	Note       string `json:"note" binding:"required"`
}

func (h *Handler) pause(c *gin.Context) {
	if _, ok := h.requirePlatformAdmin(c); !ok {
		return
	}
	var input pauseInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	updated, err := h.svc.Pause(c.Param("campaignId"), input.ReasonCode, input.Note)
	if handleAppErr(c, err) {
		return
	}
	// The code goes in the audit metadata so pauses are countable by
	// reason without parsing free text.
	h.record(c, "campaign.paused", updated.ID, map[string]any{
		"reasonCode": strings.ToUpper(input.ReasonCode),
		"note":       input.Note,
	})
	response.Success(c, http.StatusOK, gin.H{"campaign": updated})
}

func (h *Handler) resume(c *gin.Context) {
	if _, ok := h.requirePlatformAdmin(c); !ok {
		return
	}
	updated, err := h.svc.Resume(c.Param("campaignId"))
	if handleAppErr(c, err) {
		return
	}
	h.record(c, "campaign.resumed", updated.ID, nil)
	response.Success(c, http.StatusOK, gin.H{"campaign": updated})
}

func (h *Handler) archive(c *gin.Context) {
	if _, ok := h.requirePlatformAdmin(c); !ok {
		return
	}
	updated, err := h.svc.Transition(c.Param("campaignId"), domain.CampaignArchived, ActorPlatform)
	if handleAppErr(c, err) {
		return
	}
	h.record(c, "campaign.archived", updated.ID, nil)
	response.Success(c, http.StatusOK, gin.H{"campaign": updated})
}

// ─── Helpers ────────────────────────────────────────────────────────────

// loadForOrgAdmin fetches the campaign and checks the caller is an
// OWNER/ADMIN of the owning org. Returns ok=false having already written
// the error response.
func (h *Handler) loadForOrgAdmin(c *gin.Context) (*domain.Campaign, bool) {
	item, err := h.svc.Get(c.Param("campaignId"))
	if handleAppErr(c, err) {
		return nil, false
	}
	userID, userRole := actorFrom(c)
	if err := h.orgs.CanAdmin(item.OrganizationID, userID, userRole); err != nil {
		handleAppErr(c, err)
		return nil, false
	}
	return item, true
}

func (h *Handler) requirePlatformAdmin(c *gin.Context) (string, bool) {
	userID, userRole := actorFrom(c)
	if userRole != "PLATFORM_ADMIN" {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Platform admins only")
		return "", false
	}
	return userID, true
}

// record writes an audit entry. Every state change on a campaign is
// audited: this is the object that will later move money, so "who changed
// what, when" needs to be answerable without reading application logs.
func (h *Handler) record(c *gin.Context, action, targetID string, meta map[string]any) {
	if h.auditor == nil {
		return
	}
	h.auditor.Log(audit.Entry{
		Actor:      audit.FromContext(c),
		Action:     action,
		TargetType: "CAMPAIGN",
		TargetID:   targetID,
		Metadata:   meta,
		Request:    c.Request,
	})
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
