package spend

import (
	"errors"
	"net/http"

	"github.com/civicos/organization-service/internal/audit"
	"github.com/civicos/organization-service/internal/domain"
	"github.com/civicos/organization-service/internal/organizations"
	"github.com/civicos/organization-service/pkg/response"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc     *Service
	orgs    *organizations.Service
	auditor *audit.Auditor
}

func NewHandler(svc *Service, orgs *organizations.Service, auditor *audit.Auditor) *Handler {
	return &Handler{svc: svc, orgs: orgs, auditor: auditor}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, auth gin.HandlerFunc) {
	// PUBLIC. The whole point of publishing spend is that anyone can read
	// it without an account — a donor answering "where did my money go?"
	// should not have to log in to find out.
	rg.GET("/campaigns/:campaignId/spend", h.listPublic)

	// Org admin publishes and maintains the account.
	rg.POST("/campaigns/:campaignId/spend", auth, h.create)
	rg.PATCH("/spend/:spendId", auth, h.update)
	rg.DELETE("/spend/:spendId", auth, h.delete)
}

// PublicSpend is what a citizen sees. An explicit allow-list rather than the
// domain model, matching campaigns/public.go: new internal fields stay
// private until somebody deliberately publishes them.
type PublicSpend struct {
	ID          string  `json:"id"`
	MilestoneID string  `json:"milestoneId"`
	AmountMinor int64   `json:"amountMinor"`
	Currency    string  `json:"currency"`
	Description string  `json:"description"`
	SpentAt     string  `json:"spentAt"`
	ReceiptURL  *string `json:"receiptUrl,omitempty"`
	PublishedBy string  `json:"publishedBy"`
	PublishedAt string  `json:"publishedAt"`
}

func toPublic(rs []domain.SpendRecord) []PublicSpend {
	out := make([]PublicSpend, 0, len(rs))
	for _, r := range rs {
		out = append(out, PublicSpend{
			ID:          r.ID,
			MilestoneID: r.MilestoneID,
			AmountMinor: r.AmountMinor,
			Currency:    r.Currency,
			Description: r.Description,
			SpentAt:     r.SpentAt.UTC().Format("2006-01-02T15:04:05Z"),
			ReceiptURL:  r.ReceiptURL,
			// The person's name, not their user id: attribution is for
			// readers, and exposing ids invites correlation across surfaces.
			PublishedBy: r.PublishedByName,
			PublishedAt: r.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		})
	}
	return out
}

func (h *Handler) listPublic(c *gin.Context) {
	records, err := h.svc.ListForCampaign(c.Param("campaignId"))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list spend")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"spend": toPublic(records)})
}

func (h *Handler) create(c *gin.Context) {
	campaignID := c.Param("campaignId")
	camp, err := h.svc.repo.Campaign(campaignID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "CAMPAIGN_NOT_FOUND", "Campaign not found")
		return
	}
	userID, userRole := actorFrom(c)
	if err := h.orgs.CanAdmin(camp.OrganizationID, userID, userRole); err != nil {
		handleAppErr(c, err)
		return
	}

	var in CreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	userName, _ := c.Get("userName")
	rec, err := h.svc.Create(campaignID, in, userID, asString(userName))
	if handleAppErr(c, err) {
		return
	}

	h.record(c, "campaign.spend_reported", rec.ID, map[string]any{
		"campaignId":  rec.CampaignID,
		"milestoneId": rec.MilestoneID,
		"amountMinor": rec.AmountMinor,
	})
	response.Success(c, http.StatusCreated, gin.H{"spend": rec})
}

func (h *Handler) update(c *gin.Context) {
	rec, ok := h.loadForOrgAdmin(c)
	if !ok {
		return
	}
	var in UpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	before := rec.AmountMinor
	updated, err := h.svc.Update(rec, in)
	if handleAppErr(c, err) {
		return
	}

	// Every edit is audited with the old and new figure. A published spend
	// record is an accountability claim; changing one silently would let an
	// organization rewrite what it told donors after the fact.
	h.record(c, "campaign.spend_amended", updated.ID, map[string]any{
		"campaignId":        updated.CampaignID,
		"amountMinorBefore": before,
		"amountMinorAfter":  updated.AmountMinor,
	})
	response.Success(c, http.StatusOK, gin.H{"spend": updated})
}

func (h *Handler) delete(c *gin.Context) {
	rec, ok := h.loadForOrgAdmin(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(rec.ID); err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete spend record")
		return
	}
	// Withdrawing a published claim is the most sensitive of these actions,
	// so the record's content is preserved in the audit entry.
	h.record(c, "campaign.spend_withdrawn", rec.ID, map[string]any{
		"campaignId":  rec.CampaignID,
		"milestoneId": rec.MilestoneID,
		"amountMinor": rec.AmountMinor,
		"description": rec.Description,
	})
	response.Success(c, http.StatusOK, gin.H{"ok": true})
}

// loadForOrgAdmin resolves the record, walks up to its campaign, and checks
// the caller administers the owning org.
func (h *Handler) loadForOrgAdmin(c *gin.Context) (*domain.SpendRecord, bool) {
	rec, err := h.svc.Get(c.Param("spendId"))
	if handleAppErr(c, err) {
		return nil, false
	}
	userID, userRole := actorFrom(c)
	if err := h.orgs.CanAdmin(rec.OrganizationID, userID, userRole); err != nil {
		handleAppErr(c, err)
		return nil, false
	}
	return rec, true
}

func (h *Handler) record(c *gin.Context, action, targetID string, meta map[string]any) {
	if h.auditor == nil {
		return
	}
	h.auditor.Log(audit.Entry{
		Actor:      audit.FromContext(c),
		Action:     action,
		TargetType: "SPEND_RECORD",
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

func asString(v any) string {
	s, _ := v.(string)
	return s
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
	response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Something went wrong")
	return true
}
