package flags

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/civicos/identity-service/internal/audit"
	"github.com/civicos/identity-service/pkg/response"
	"github.com/gin-gonic/gin"
)

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
