package consultations

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/civicos/organization-service/internal/audit"
	"github.com/civicos/organization-service/internal/notifications"
	"github.com/civicos/organization-service/internal/organizations"
	"github.com/civicos/organization-service/pkg/response"
	"github.com/gin-gonic/gin"
)

// Notifier is the minimal interface the handler needs, satisfied by
// notifications.DBNotifier. Kept as an interface here to keep the
// handler decoupled from the concrete writer.
type Notifier interface {
	Emit(userID string, t notifications.NotificationType, title, body string, linkURL *string) error
	EmitMany(userIDs []string, t notifications.NotificationType, title, body string, linkURL *string)
}

type Handler struct {
	svc      *Service
	orgs     *organizations.Service
	auditor  *audit.Auditor
	notifier Notifier
}

func NewHandler(svc *Service, orgs *organizations.Service, auditor *audit.Auditor, notifier Notifier) *Handler {
	return &Handler{svc: svc, orgs: orgs, auditor: auditor, notifier: notifier}
}

// RegisterRoutes mounts every consultation URL under the /v1 router.
// URL shapes span multiple roots (org, consultation, question,
// response, outcome) so they're mounted on v1 rather than a single
// resource-scoped group — same convention as announcements and
// progress updates.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, auth, verified gin.HandlerFunc) {
	// Public reads.
	rg.GET("/consultations", h.list)
	rg.GET("/consultations/:id", h.get)
	rg.GET("/consultations/:id/questions", h.listQuestions)
	rg.GET("/consultations/:id/outcome", h.getOutcome)

	// Verified-user actions.
	rg.POST("/consultations/:id/responses", auth, verified, h.submitResponse)
	rg.GET("/me/consultations/responses", auth, h.myResponses)

	// Org-admin actions.
	rg.POST("/organizations/:id/consultations", auth, h.create)
	rg.PATCH("/consultations/:id", auth, h.update)
	rg.DELETE("/consultations/:id", auth, h.delete)
	rg.POST("/consultations/:id/publish", auth, h.publish)
	rg.POST("/consultations/:id/close", auth, h.close)

	rg.POST("/consultations/:id/questions", auth, h.addQuestion)
	rg.PATCH("/consultation-questions/:questionId", auth, h.updateQuestion)
	rg.DELETE("/consultation-questions/:questionId", auth, h.deleteQuestion)
	rg.PATCH("/consultations/:id/questions/reorder", auth, h.reorderQuestions)

	// Admin-only reads.
	rg.GET("/consultations/:id/responses", auth, h.listResponses)
	rg.GET("/consultations/:id/analytics", auth, h.analytics)

	// Outcome publishing (closes the loop).
	rg.POST("/consultations/:id/outcome", auth, h.publishOutcome)
}

// ── Public reads ─────────────────────────────────────────────────

func (h *Handler) list(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	items, err := h.svc.List(ListFilters{
		OrganizationID: c.Query("organizationId"),
		CommunityID:    c.Query("communityId"),
		Status:         strings.ToUpper(c.Query("status")),
		Limit:          limit,
		Offset:         offset,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list consultations")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"consultations": items})
}

func (h *Handler) get(c *gin.Context) {
	item, err := h.svc.Get(c.Param("id"))
	if handleAppErr(c, err) {
		return
	}
	response.Success(c, http.StatusOK, gin.H{"consultation": item})
}

func (h *Handler) listQuestions(c *gin.Context) {
	items, err := h.svc.ListQuestions(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch questions")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"questions": items})
}

func (h *Handler) getOutcome(c *gin.Context) {
	item, err := h.svc.GetOutcome(c.Param("id"))
	if handleAppErr(c, err) {
		return
	}
	response.Success(c, http.StatusOK, gin.H{"outcome": item})
}

// ── Response submission ──────────────────────────────────────────

func (h *Handler) submitResponse(c *gin.Context) {
	var in SubmitResponseInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	userID, _ := c.Get("userID")
	resp, err := h.svc.SubmitResponse(c.Param("id"), userID.(string), in)
	if handleAppErr(c, err) {
		return
	}
	response.Success(c, http.StatusCreated, gin.H{"response": resp})
}

func (h *Handler) myResponses(c *gin.Context) {
	userID, _ := c.Get("userID")
	ids, err := h.svc.MyResponseIDs(userID.(string))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to load responses")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"consultationIds": ids})
}

// ── Org-admin lifecycle ──────────────────────────────────────────

func (h *Handler) create(c *gin.Context) {
	orgID := c.Param("id")
	if err := h.requireAdmin(c, orgID); err != nil {
		handleAppErr(c, err)
		return
	}
	var in CreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	actor := audit.FromContext(c)
	item, err := h.svc.Create(orgID, in, actor.ID, actor.Name)
	if handleAppErr(c, err) {
		return
	}
	h.auditor.Log(audit.Entry{
		Actor:      actor,
		Action:     "consultation.created",
		TargetType: "CONSULTATION",
		TargetID:   item.ID,
		Metadata:   map[string]any{"orgId": orgID, "title": item.Title, "communityId": item.CommunityID},
		Request:    c.Request,
	})
	response.Success(c, http.StatusCreated, gin.H{"consultation": item})
}

func (h *Handler) update(c *gin.Context) {
	id := c.Param("id")
	cur, err := h.svc.Get(id)
	if handleAppErr(c, err) {
		return
	}
	if err := h.requireAdmin(c, cur.OrganizationID); err != nil {
		handleAppErr(c, err)
		return
	}
	var in UpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	item, err := h.svc.Update(id, in)
	if handleAppErr(c, err) {
		return
	}
	response.Success(c, http.StatusOK, gin.H{"consultation": item})
}

func (h *Handler) delete(c *gin.Context) {
	id := c.Param("id")
	cur, err := h.svc.Get(id)
	if handleAppErr(c, err) {
		return
	}
	if err := h.requireAdmin(c, cur.OrganizationID); err != nil {
		handleAppErr(c, err)
		return
	}
	if err := h.svc.Delete(id); handleAppErr(c, err) {
		return
	}
	h.auditor.Log(audit.Entry{
		Actor:      audit.FromContext(c),
		Action:     "consultation.deleted",
		TargetType: "CONSULTATION",
		TargetID:   id,
		Metadata:   map[string]any{"orgId": cur.OrganizationID, "title": cur.Title},
		Request:    c.Request,
	})
	response.Success(c, http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) publish(c *gin.Context) {
	id := c.Param("id")
	cur, err := h.svc.Get(id)
	if handleAppErr(c, err) {
		return
	}
	if err := h.requireAdmin(c, cur.OrganizationID); err != nil {
		handleAppErr(c, err)
		return
	}
	item, err := h.svc.Publish(id)
	if handleAppErr(c, err) {
		return
	}
	h.auditor.Log(audit.Entry{
		Actor:      audit.FromContext(c),
		Action:     "consultation.published",
		TargetType: "CONSULTATION",
		TargetID:   id,
		Metadata:   map[string]any{"orgId": item.OrganizationID, "title": item.Title, "communityId": item.CommunityID},
		Request:    c.Request,
	})

	// Fan out a notification to org members (whole-org audience) or to
	// the community (single-community audience). Community-scoped fan-out
	// requires cross-service data (community memberships live in
	// identity-service); MVP fans out to org members only. Broader
	// fan-out lands in v1 alongside community-membership lookups.
	if h.notifier != nil {
		members, mErr := h.orgs.ListMembers(item.OrganizationID)
		if mErr != nil {
			log.Printf("notify consultation.published: list members: %v", mErr)
		} else {
			userIDs := make([]string, 0, len(members))
			for _, m := range members {
				userIDs = append(userIDs, m.UserID)
			}
			link := "/consultations/" + item.ID
			h.notifier.EmitMany(userIDs,
				notifications.TypeConsultationUpdate,
				"New consultation: "+item.Title,
				item.Summary,
				&link,
			)
		}
	}

	response.Success(c, http.StatusOK, gin.H{"consultation": item})
}

func (h *Handler) close(c *gin.Context) {
	id := c.Param("id")
	cur, err := h.svc.Get(id)
	if handleAppErr(c, err) {
		return
	}
	// Close is the emergency lever — platform admins retain this even
	// without org membership so a bad consultation can be frozen
	// without escalating to full moderation.
	if err := h.requireCloseAuthority(c, cur.OrganizationID); err != nil {
		handleAppErr(c, err)
		return
	}
	item, err := h.svc.Close(id)
	if handleAppErr(c, err) {
		return
	}
	h.auditor.Log(audit.Entry{
		Actor:      audit.FromContext(c),
		Action:     "consultation.closed",
		TargetType: "CONSULTATION",
		TargetID:   id,
		Metadata:   map[string]any{"orgId": item.OrganizationID, "title": item.Title, "responseCount": item.ResponseCount},
		Request:    c.Request,
	})

	// Notify responders that the consultation closed so they can watch
	// for the outcome.
	if h.notifier != nil {
		responderIDs, rErr := h.svc.ResponderIDs(id)
		if rErr == nil && len(responderIDs) > 0 {
			link := "/consultations/" + id
			h.notifier.EmitMany(responderIDs,
				notifications.TypeConsultationUpdate,
				"Consultation closed: "+item.Title,
				"Responses are no longer being collected. Watch for the outcome.",
				&link,
			)
		}
	}

	response.Success(c, http.StatusOK, gin.H{"consultation": item})
}

// ── Questions ────────────────────────────────────────────────────

func (h *Handler) addQuestion(c *gin.Context) {
	id := c.Param("id")
	cur, err := h.svc.Get(id)
	if handleAppErr(c, err) {
		return
	}
	if err := h.requireAdmin(c, cur.OrganizationID); err != nil {
		handleAppErr(c, err)
		return
	}
	var in QuestionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	q, err := h.svc.AddQuestion(id, in)
	if handleAppErr(c, err) {
		return
	}
	response.Success(c, http.StatusCreated, gin.H{"question": q})
}

func (h *Handler) updateQuestion(c *gin.Context) {
	qid := c.Param("questionId")
	q, err := h.svc.getQuestion(qid)
	if handleAppErr(c, err) {
		return
	}
	cur, err := h.svc.Get(q.ConsultationID)
	if handleAppErr(c, err) {
		return
	}
	if err := h.requireAdmin(c, cur.OrganizationID); err != nil {
		handleAppErr(c, err)
		return
	}
	var in QuestionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	updated, err := h.svc.UpdateQuestion(qid, in)
	if handleAppErr(c, err) {
		return
	}
	response.Success(c, http.StatusOK, gin.H{"question": updated})
}

func (h *Handler) deleteQuestion(c *gin.Context) {
	qid := c.Param("questionId")
	q, err := h.svc.getQuestion(qid)
	if handleAppErr(c, err) {
		return
	}
	cur, err := h.svc.Get(q.ConsultationID)
	if handleAppErr(c, err) {
		return
	}
	if err := h.requireAdmin(c, cur.OrganizationID); err != nil {
		handleAppErr(c, err)
		return
	}
	if err := h.svc.DeleteQuestion(qid); handleAppErr(c, err) {
		return
	}
	response.Success(c, http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) reorderQuestions(c *gin.Context) {
	id := c.Param("id")
	cur, err := h.svc.Get(id)
	if handleAppErr(c, err) {
		return
	}
	if err := h.requireAdmin(c, cur.OrganizationID); err != nil {
		handleAppErr(c, err)
		return
	}
	var in ReorderInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	if err := h.svc.ReorderQuestions(id, in); handleAppErr(c, err) {
		return
	}
	response.Success(c, http.StatusOK, gin.H{"ok": true})
}

// ── Admin reads ──────────────────────────────────────────────────

func (h *Handler) listResponses(c *gin.Context) {
	id := c.Param("id")
	cur, err := h.svc.Get(id)
	if handleAppErr(c, err) {
		return
	}
	if err := h.requireInternalRead(c, cur.OrganizationID); err != nil {
		handleAppErr(c, err)
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	items, total, err := h.svc.ListResponses(id, limit, offset)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list responses")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"responses": items, "total": total})
}

func (h *Handler) analytics(c *gin.Context) {
	id := c.Param("id")
	cur, err := h.svc.Get(id)
	if handleAppErr(c, err) {
		return
	}
	if err := h.requireInternalRead(c, cur.OrganizationID); err != nil {
		handleAppErr(c, err)
		return
	}
	agg, err := h.svc.Analytics(id)
	if handleAppErr(c, err) {
		return
	}
	response.Success(c, http.StatusOK, gin.H{
		"consultation":  cur,
		"responseCount": cur.ResponseCount,
		"questions":     agg,
	})
}

// ── Outcome ──────────────────────────────────────────────────────

func (h *Handler) publishOutcome(c *gin.Context) {
	id := c.Param("id")
	cur, err := h.svc.Get(id)
	if handleAppErr(c, err) {
		return
	}
	if err := h.requireAdmin(c, cur.OrganizationID); err != nil {
		handleAppErr(c, err)
		return
	}
	var in OutcomeInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	actor := audit.FromContext(c)
	item, err := h.svc.PublishOutcome(id, in, actor.ID, actor.Name)
	if handleAppErr(c, err) {
		return
	}
	h.auditor.Log(audit.Entry{
		Actor:      actor,
		Action:     "consultation.outcome_published",
		TargetType: "CONSULTATION",
		TargetID:   id,
		Metadata:   map[string]any{"orgId": cur.OrganizationID, "title": cur.Title},
		Request:    c.Request,
	})

	// Notify every responder that the outcome is out — this is the
	// close-the-loop moment.
	if h.notifier != nil {
		responderIDs, rErr := h.svc.ResponderIDs(id)
		if rErr == nil && len(responderIDs) > 0 {
			link := "/consultations/" + id + "#outcome"
			h.notifier.EmitMany(responderIDs,
				notifications.TypeConsultationUpdate,
				"Outcome published: "+cur.Title,
				"The organization has posted findings and next steps.",
				&link,
			)
		}
	}

	response.Success(c, http.StatusCreated, gin.H{"outcome": item})
}

// ── Auth helpers ─────────────────────────────────────────────────

// requireAdmin gates content-authorship write paths. Strict: the caller
// must be an OWNER or ADMIN member of the target org. Platform admins
// who aren't in the org are refused — the consultation's authorship
// belongs to the org itself. See organizations.CanAdmin for the
// underlying rule.
func (h *Handler) requireAdmin(c *gin.Context, orgID string) error {
	userID, _ := c.Get("userID")
	userRole, _ := c.Get("userRole")
	uid, _ := userID.(string)
	role, _ := userRole.(string)
	if uid == "" {
		return &AppError{Code: "UNAUTHORIZED", Message: "Sign in required", Status: http.StatusUnauthorized}
	}
	return h.orgs.CanAdmin(orgID, uid, role)
}

// requireCloseAuthority gates emergency-close actions. Same rules as
// requireAdmin plus PLATFORM_ADMIN is allowed even without org
// membership — the platform must be able to freeze a bad consultation
// without joining the org first.
func (h *Handler) requireCloseAuthority(c *gin.Context, orgID string) error {
	userID, _ := c.Get("userID")
	userRole, _ := c.Get("userRole")
	uid, _ := userID.(string)
	role, _ := userRole.(string)
	if uid == "" {
		return &AppError{Code: "UNAUTHORIZED", Message: "Sign in required", Status: http.StatusUnauthorized}
	}
	return h.orgs.CanClose(orgID, uid, role)
}

// requireInternalRead gates admin-only reads (response list, analytics).
// Any org member (including STAFF) can view, plus PLATFORM_ADMIN.
func (h *Handler) requireInternalRead(c *gin.Context, orgID string) error {
	userID, _ := c.Get("userID")
	userRole, _ := c.Get("userRole")
	uid, _ := userID.(string)
	role, _ := userRole.(string)
	if uid == "" {
		return &AppError{Code: "UNAUTHORIZED", Message: "Sign in required", Status: http.StatusUnauthorized}
	}
	return h.orgs.CanReadInternal(orgID, uid, role)
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
	// Bubble org-service errors (from CanAdmin) with their own code/status.
	var orgErr *organizations.AppError
	if errors.As(err, &orgErr) {
		response.Error(c, orgErr.Status, orgErr.Code, orgErr.Message)
		return true
	}
	response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	return true
}
