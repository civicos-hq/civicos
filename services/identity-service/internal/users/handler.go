package users

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/civicos/identity-service/internal/audit"
	"github.com/civicos/identity-service/internal/domain"
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

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, auth, requireAdmin gin.HandlerFunc) {
	rg.GET("", auth, requireAdmin, h.list)
	rg.GET("/:id", auth, requireAdmin, h.get)
	rg.PATCH("/:id/role", auth, requireAdmin, h.changeRole)
	rg.POST("/:id/ban", auth, requireAdmin, h.ban)
	rg.POST("/:id/unban", auth, requireAdmin, h.unban)
}

func (h *Handler) list(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	users, total, err := h.svc.List(ListFilters{
		Search: c.Query("q"),
		Role:   c.Query("role"),
		Banned: c.Query("banned"),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list users")
		return
	}
	// Sanitize — strip password hashes and reset tokens before returning.
	public := make([]any, len(users))
	for i, u := range users {
		public[i] = gin.H{
			"id":                 u.ID,
			"email":              u.Email,
			"name":               u.Name,
			"role":               u.Role,
			"emailVerified":      u.EmailVerified,
			"activeCommunityId":  u.ActiveCommunityID,
			"primaryCommunityId": u.PrimaryCommunityID,
			"memberships":        domain.ToPublicMemberships(u.Memberships),
			"avatarUrl":          u.AvatarURL,
			"bannedAt":           u.BannedAt,
			"banReason":          u.BanReason,
			"bannedById":         u.BannedByID,
			"createdAt":          u.CreatedAt,
		}
	}
	response.Success(c, http.StatusOK, gin.H{"users": public, "total": total})
}

func (h *Handler) get(c *gin.Context) {
	u, err := h.svc.Get(c.Param("id"))
	if handleAppErr(c, err) {
		return
	}
	response.Success(c, http.StatusOK, gin.H{"user": u.ToPublic()})
}

func (h *Handler) changeRole(c *gin.Context) {
	id := c.Param("id")
	var input RoleUpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	previous, err := h.svc.Get(id)
	if handleAppErr(c, err) {
		return
	}
	u, err := h.svc.ChangeRole(id, input)
	if handleAppErr(c, err) {
		return
	}
	h.auditor.Log(audit.Entry{
		Actor:      audit.FromContext(c),
		Action:     "user.role_changed",
		TargetType: "USER",
		TargetID:   id,
		Metadata: map[string]any{
			"previousRole": previous.Role,
			"newRole":      input.Role,
			"email":        u.Email,
		},
		Request: c.Request,
	})
	response.Success(c, http.StatusOK, gin.H{"user": u.ToPublic()})
}

func (h *Handler) ban(c *gin.Context) {
	id := c.Param("id")
	var input BanInput
	_ = c.ShouldBindJSON(&input) // reason is optional
	actor := audit.FromContext(c)
	u, err := h.svc.Ban(id, input, actor.ID)
	if handleAppErr(c, err) {
		return
	}
	h.auditor.Log(audit.Entry{
		Actor:      actor,
		Action:     "user.banned",
		TargetType: "USER",
		TargetID:   id,
		Metadata: map[string]any{
			"reason": input.Reason,
			"email":  u.Email,
		},
		Request: c.Request,
	})
	response.Success(c, http.StatusOK, gin.H{"user": u.ToPublic()})
}

func (h *Handler) unban(c *gin.Context) {
	id := c.Param("id")
	actor := audit.FromContext(c)
	u, err := h.svc.Unban(id)
	if handleAppErr(c, err) {
		return
	}
	h.auditor.Log(audit.Entry{
		Actor:      actor,
		Action:     "user.unbanned",
		TargetType: "USER",
		TargetID:   id,
		Metadata:   map[string]any{"email": u.Email},
		Request:    c.Request,
	})
	response.Success(c, http.StatusOK, gin.H{"user": u.ToPublic()})
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
