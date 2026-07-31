package donations

import (
	"errors"
	"io"
	"net/http"

	"github.com/civicos/organization-service/internal/audit"
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

// maxWebhookBody caps what we will read from an unauthenticated endpoint.
// Without it, anyone who knows the URL can stream an unbounded body at us.
const maxWebhookBody = 1 << 20 // 1 MiB

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, auth gin.HandlerFunc) {
	// PUBLIC — a donor need not have an account to give. Rate limiting at
	// the gateway does the abuse work; there is nothing to authenticate.
	rg.POST("/campaigns/:campaignId/donation-intents", h.createIntent)
	rg.GET("/campaigns/:campaignId/donations", h.publicDonations)

	// UNAUTHENTICATED BY NECESSITY. Paystack does not carry our JWTs, so
	// this route must not sit behind auth middleware — it authenticates by
	// HMAC signature over the raw body instead. Anything added here that
	// trusts the caller rather than the signature is a hole.
	rg.POST("/webhooks/paystack", h.paystackWebhook)

	// Org admin: connect the payout destination.
	rg.POST("/organizations/:id/psp-account", auth, h.connectSubaccount)
}

// ─── Donations ──────────────────────────────────────────────────────────

func (h *Handler) createIntent(c *gin.Context) {
	var in IntentInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	// Attribute the donation to a signed-in user when there is one, but do
	// not require it — guest giving is deliberate.
	var donorUserID *string
	if v, ok := c.Get("userID"); ok {
		if s, ok := v.(string); ok && s != "" {
			donorUserID = &s
		}
	}
	res, err := h.svc.CreateIntent(c.Request.Context(), c.Param("campaignId"), donorUserID, in)
	if handleAppErr(c, err) {
		return
	}
	response.Success(c, http.StatusCreated, gin.H{"donation": res})
}

func (h *Handler) publicDonations(c *gin.Context) {
	items, err := h.svc.PublicDonations(c.Param("campaignId"))
	if handleAppErr(c, err) {
		return
	}
	response.Success(c, http.StatusOK, gin.H{"donations": items})
}

// paystackWebhook is the only unauthenticated write path in the service.
//
// Two things it must get right beyond signature verification:
//
//  1. Read the RAW body. Gin's binding would consume and re-encode it, and
//     the HMAC is over the exact bytes Paystack sent.
//  2. Answer 200 for anything that is merely unactionable — an unknown
//     reference, a duplicate delivery, an event type we ignore. Paystack
//     retries non-2xx responses, so returning an error for "nothing to do"
//     buys a retry storm. Only a failed signature earns a non-2xx, because
//     that one should never be retried into success.
func (h *Handler) paystackWebhook(c *gin.Context) {
	raw, err := io.ReadAll(io.LimitReader(c.Request.Body, maxWebhookBody))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "UNREADABLE_BODY", "Could not read the request body")
		return
	}
	sig := c.GetHeader("x-paystack-signature")

	if err := h.svc.HandleWebhook(c.Request.Context(), raw, sig); err != nil {
		var appErr *AppError
		if errors.As(err, &appErr) {
			response.Error(c, appErr.Status, appErr.Code, appErr.Message)
			return
		}
		// A genuine server-side failure SHOULD be retried — this is the one
		// case where we want Paystack to come back.
		response.Error(c, http.StatusInternalServerError, "WEBHOOK_FAILED", "Could not process the event")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"received": true})
}

// ─── Sub-account connection ─────────────────────────────────────────────

func (h *Handler) connectSubaccount(c *gin.Context) {
	orgID := c.Param("id")
	userID, _ := c.Get("userID")
	userRole, _ := c.Get("userRole")
	if err := h.orgs.CanAdmin(orgID, asString(userID), asString(userRole)); err != nil {
		handleAppErr(c, err)
		return
	}
	var in ConnectInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	org, err := h.svc.ConnectSubaccount(c.Request.Context(), orgID, in)
	if handleAppErr(c, err) {
		return
	}
	// The account number is deliberately absent from the audit metadata —
	// it is not persisted anywhere, and an audit log is not an exception.
	if h.auditor != nil {
		h.auditor.Log(audit.Entry{
			Actor:      audit.FromContext(c),
			Action:     "org.psp_connected",
			TargetType: "ORGANIZATION",
			TargetID:   orgID,
			Metadata: map[string]any{
				"provider": org.PSPProvider,
				"bank":     org.PSPBankName,
				"last4":    org.PSPAccountLast4,
			},
			Request: c.Request,
		})
	}
	response.Success(c, http.StatusOK, gin.H{"organization": org})
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
