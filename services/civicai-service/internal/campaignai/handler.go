package campaignai

import (
	"errors"
	"log"
	"net/http"

	"github.com/civicos/civicai-service/pkg/response"
	"github.com/gin-gonic/gin"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// staffRoles mirrors the existing draft / summarize / insights handlers and
// the FE STAFF_ROLES set. These are the platform's actual roles — an
// organization admin carries NGO, not a per-org role in the JWT.
//
// This gate is coarse on purpose: it only keeps citizens from burning Gemini
// calls. The surfaces that read a real campaign are authorised properly
// upstream, where organization-service applies CanReadInternal against the
// owning org — so an NGO user cannot pull another organization's campaign
// through here just by holding the right role.
var staffRoles = map[string]struct{}{
	"REPRESENTATIVE":   {},
	"GOVERNMENT_ADMIN": {},
	"PLATFORM_ADMIN":   {},
	"NGO":              {},
	"MODERATOR":        {},
}

// riskRoles is PLATFORM_ADMIN alone.
//
// Not org admins, and the difference matters: assess-campaign-risk reads a
// campaign and returns fraud signals about the organization running it. An
// organization able to point that at a rival's flood appeal would have been
// handed a weapon. The source read is also authorised upstream by
// CanReadInternal, so this is the second of two gates, not the only one.
var riskRoles = map[string]struct{}{
	"PLATFORM_ADMIN": {},
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/classify-campaign", h.staff(h.classify))
	rg.POST("/draft-campaign", h.staff(h.draft))
	rg.POST("/summarize-campaign-impact", h.staff(h.impact))
	rg.POST("/draft-donor-update", h.staff(h.donorUpdate))
	rg.POST("/draft-completion-report", h.staff(h.completionReport))
	rg.POST("/assess-campaign-risk", h.admin(h.assessRisk))
}

func (h *Handler) staff(next gin.HandlerFunc) gin.HandlerFunc {
	return h.requireRole(staffRoles, "Organization staff only", next)
}

func (h *Handler) admin(next gin.HandlerFunc) gin.HandlerFunc {
	return h.requireRole(riskRoles, "Platform admins only", next)
}

func (h *Handler) requireRole(allowed map[string]struct{}, msg string, next gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("userRole")
		roleStr, _ := role.(string)
		if _, ok := allowed[roleStr]; !ok {
			response.Error(c, http.StatusForbidden, "FORBIDDEN", msg)
			return
		}
		next(c)
	}
}

// bearer returns the caller's Authorization header verbatim, for forwarding
// to organization-service so the campaign read is authorised as them.
func bearer(c *gin.Context) string { return c.GetHeader("Authorization") }

// fail maps an error to a response.
//
// A SourceError is the caller's problem (they cannot read that campaign, or
// it does not exist) and keeps its own status. Anything else is Gemini, and
// gets 502 with a message that tells the user they can still do the task by
// hand — principle 4, fail open: AI is additive, never load-bearing.
func fail(c *gin.Context, op string, err error, fallback string) {
	var se *SourceError
	if errors.As(err, &se) {
		response.Error(c, se.Status, se.Code, se.Message)
		return
	}
	log.Printf("[campaignai/%s] failed: %v", op, err)
	response.Error(c, http.StatusBadGateway, "AI_UNAVAILABLE", fallback)
}

func (h *Handler) classify(c *gin.Context) {
	var in ClassifyInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	out, err := h.svc.Classify(c.Request.Context(), in)
	if err != nil {
		fail(c, "classify", err, "CivicAI could not suggest a category. Please pick one yourself.")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"classification": out})
}

func (h *Handler) draft(c *gin.Context) {
	var in DraftInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	out, err := h.svc.Draft(c.Request.Context(), in)
	if err != nil {
		fail(c, "draft", err, "CivicAI could not draft this campaign. You can still write it yourself.")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"draft": out})
}

func (h *Handler) impact(c *gin.Context) {
	var in ImpactInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	out, err := h.svc.Impact(c.Request.Context(), in.CampaignID, bearer(c))
	if err != nil {
		fail(c, "impact", err, "CivicAI could not summarise this campaign right now.")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"impact": out})
}

func (h *Handler) donorUpdate(c *gin.Context) {
	var in DonorUpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	out, err := h.svc.DonorUpdate(c.Request.Context(), in, bearer(c))
	if err != nil {
		fail(c, "donor-update", err, "CivicAI could not draft this update. You can still write it yourself.")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"update": out})
}

func (h *Handler) completionReport(c *gin.Context) {
	var in CompletionReportInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	out, err := h.svc.CompletionReport(c.Request.Context(), in, bearer(c))
	if err != nil {
		fail(c, "completion-report", err, "CivicAI could not draft this report. You can still write it yourself.")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"report": out})
}

func (h *Handler) assessRisk(c *gin.Context) {
	var in RiskInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	out, err := h.svc.AssessRisk(c.Request.Context(), in.CampaignID, bearer(c))
	if err != nil {
		fail(c, "assess-risk", err, "CivicAI could not assess this campaign. Review it manually.")
		return
	}
	// Nothing is written to the campaign here — no risk score persisted, no
	// status changed, no notification sent. The reviewer acts through the
	// ordinary review and pause endpoints, which carry their own audit trail.
	response.Success(c, http.StatusOK, gin.H{"assessment": out})
}
