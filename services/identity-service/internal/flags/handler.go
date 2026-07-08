package flags

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/civicos/identity-service/internal/audit"
	"github.com/civicos/identity-service/internal/domain"
	"github.com/civicos/identity-service/pkg/response"
	"github.com/gin-gonic/gin"
)

// domainUser adapts an audit.Actor into the domain.User shape the
// flags service expects for DirectHide. Only ID + Name are read.
func domainUser(a audit.Actor) domain.User {
	return domain.User{ID: a.ID, Name: a.Name}
}

type Handler struct {
	svc     *Service
	auditor *audit.Auditor
}

func NewHandler(svc *Service, auditor *audit.Auditor) *Handler {
	return &Handler{svc: svc, auditor: auditor}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, auth, requireVerified, requireAdmin gin.HandlerFunc) {
	// Citizens: file a flag. Verified email required so throwaway accounts
	// can't drown the queue.
	rg.POST("", auth, requireVerified, h.create)

	// Moderators: read the queue, resolve, view counters.
	rg.GET("", auth, requireAdmin, h.list)
	rg.GET("/counts", auth, requireAdmin, h.counts)
	rg.GET("/:id", auth, requireAdmin, h.get)
	rg.PATCH("/:id", auth, requireAdmin, h.resolve)

	// Admin's proactive-moderation shortcut: create+hide in one call.
	rg.POST("/direct-hide", auth, requireAdmin, h.directHide)
}

// directHide creates a flag and immediately resolves it as HIDDEN, with
// the admin as both reporter and resolver. Writes an audit entry with
// the "flag.direct_hide" action so it's distinguishable in the audit
// log from citizen-filed flags that were later resolved.
func (h *Handler) directHide(c *gin.Context) {
	var input DirectHideInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	actor := audit.FromContext(c)
	// The service expects a domain.User for the actor; only ID and Name
	// are actually read.
	f, err := h.svc.DirectHide(input, domainUser(actor))
	if handleAppErr(c, err) {
		return
	}
	h.auditor.Log(audit.Entry{
		Actor:      actor,
		Action:     "flag.direct_hide",
		TargetType: "CONTENT_FLAG",
		TargetID:   f.ID,
		Metadata: map[string]any{
			"contentType":    input.ContentType,
			"contentId":      input.ContentID,
			"reason":         input.Reason,
			"resolutionNote": input.ResolutionNote,
		},
		Request: c.Request,
	})
	response.Success(c, http.StatusCreated, gin.H{"flag": f})
}

func (h *Handler) create(c *gin.Context) {
	var input CreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	userID, _ := c.Get("userID")
	userName, _ := c.Get("userName")
	f, err := h.svc.Create(input, userID.(string), asString(userName))
	if handleAppErr(c, err) {
		return
	}
	response.Success(c, http.StatusCreated, gin.H{"flag": f})
}

func (h *Handler) list(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	items, err := h.svc.List(ListFilters{
		Status:      strings.ToUpper(c.Query("status")),
		ContentType: strings.ToUpper(c.Query("contentType")),
		Reason:      strings.ToUpper(c.Query("reason")),
		ReporterID:  c.Query("reporterId"),
		Search:      strings.TrimSpace(c.Query("q")),
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list flags")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"flags": items})
}

func (h *Handler) counts(c *gin.Context) {
	counts, err := h.svc.Counts()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to count flags")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"counts": counts})
}

func (h *Handler) get(c *gin.Context) {
	f, err := h.svc.Get(c.Param("id"))
	if handleAppErr(c, err) {
		return
	}
	response.Success(c, http.StatusOK, gin.H{"flag": f})
}

func (h *Handler) resolve(c *gin.Context) {
	id := c.Param("id")
	var input ResolveInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	actor := audit.FromContext(c)
	f, err := h.svc.Resolve(id, input, actor)
	if handleAppErr(c, err) {
		return
	}

	// Audit every resolution so the moderator's decision is reviewable.
	// Kept OUT of the service layer so the service stays request-agnostic.
	h.auditor.Log(audit.Entry{
		Actor:      actor,
		Action:     "flag.resolved",
		TargetType: "CONTENT_FLAG",
		TargetID:   id,
		Metadata: map[string]any{
			"newStatus":      strings.ToUpper(input.Status),
			"contentType":    f.ContentType,
			"contentId":      f.ContentID,
			"resolutionNote": input.ResolutionNote,
		},
		Request: c.Request,
	})

	response.Success(c, http.StatusOK, gin.H{"flag": f})
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
	response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	return true
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
