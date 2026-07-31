package donations

import (
	"errors"
	"io"
	"net/http"
	"time"

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

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, auth gin.HandlerFunc, optionalAuth gin.HandlerFunc) {
	// PUBLIC — a donor need not have an account to give. Rate limiting at
	// the gateway does the abuse work; there is nothing to authenticate.
	// optionalAuth, not auth: a guest may give, but a signed-in donor must
	// be ATTRIBUTED, or their donation is never linked to their account and
	// they can never be notified about the campaign they funded.
	rg.POST("/campaigns/:campaignId/donation-intents", optionalAuth, h.createIntent)
	rg.GET("/campaigns/:campaignId/donations", h.publicDonations)

	// UNAUTHENTICATED BY NECESSITY. Paystack does not carry our JWTs, so
	// this route must not sit behind auth middleware — it authenticates by
	// HMAC signature over the raw body instead. Anything added here that
	// trusts the caller rather than the signature is a hole.
	rg.POST("/webhooks/paystack", h.paystackWebhook)

	// Org admin: connect the payout destination.
	rg.POST("/organizations/:id/psp-account", auth, h.connectSubaccount)

	// Platform admin: run reconciliation on demand. The job also runs on a
	// timer, but an admin investigating a specific complaint should not have
	// to wait for the next tick.
	rg.POST("/admin/donations/reconcile", auth, h.reconcile)
}

// ─── Reconciliation ─────────────────────────────────────────────────────

type reconcileInput struct {
	// Minutes a PENDING donation must have sat before it is chased.
	// Omitted uses the default; an explicit 0 means "check everything now",
	// which is what an admin chasing a specific complaint needs.
	PendingGraceMinutes *int `json:"pendingGraceMinutes"`
	// How far back to re-check already-settled rows.
	SettledWindowHours int `json:"settledWindowHours"`
	Limit              int `json:"limit"`
}

// reconcile re-reads transactions from the provider and repairs or reports.
//
// PLATFORM_ADMIN only. It can move a donation to SETTLED, which changes a
// campaign's public total — not something an org should be able to trigger
// against its own campaigns.
func (h *Handler) reconcile(c *gin.Context) {
	_, userRole := actorFrom(c)
	if userRole != "PLATFORM_ADMIN" {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Platform admins only")
		return
	}

	var in reconcileInput
	// A body is optional: an empty POST runs with the defaults.
	_ = c.ShouldBindJSON(&in)

	opts := ReconcileOptions{
		SettledWindow: time.Duration(in.SettledWindowHours) * time.Hour,
		Limit:         in.Limit,
	}
	if in.PendingGraceMinutes != nil {
		grace := time.Duration(*in.PendingGraceMinutes) * time.Minute
		opts.PendingGrace = &grace
	}

	rep, err := h.svc.Reconcile(c.Request.Context(), opts)
	if handleAppErr(c, err) {
		return
	}

	// Audited: this is a privileged action that can change public totals,
	// and "who ran it, when, and what did it find" needs to be answerable
	// without reading application logs.
	if h.auditor != nil {
		h.auditor.Log(audit.Entry{
			Actor:  audit.FromContext(c),
			Action: "donations.reconciled",
			// The run itself is the target: reconciliation is
			// platform-wide and has no single donation to point at, and
			// audit_logs.target_id is a NOT NULL uuid column.
			TargetType: "RECONCILIATION_RUN",
			TargetID:   rep.ID,
			Metadata: map[string]any{
				"pendingChecked": rep.PendingChecked,
				"settledChecked": rep.SettledChecked,
				"recovered":      rep.Recovered,
				"recoveredMinor": rep.RecoveredMinor,
				"driftCount":     len(rep.Drift),
			},
			Request: c.Request,
		})
	}
	response.Success(c, http.StatusOK, gin.H{"report": rep})
}

func actorFrom(c *gin.Context) (userID, userRole string) {
	id, _ := c.Get("userID")
	role, _ := c.Get("userRole")
	return asString(id), asString(role)
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
