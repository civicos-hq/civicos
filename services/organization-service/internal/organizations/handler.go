package organizations

import (
	"errors"
	"net/http"
	"strings"

	"github.com/civicos/organization-service/internal/audit"
	"github.com/civicos/organization-service/pkg/response"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc     *Service
	auditor *audit.Auditor
}

func NewHandler(svc *Service, auditor *audit.Auditor) *Handler {
	return &Handler{svc: svc, auditor: auditor}
}

func (h *Handler) Service() *Service { return h.svc }

// Organizations are created only by approving an
// OrganizationApplication in identity-service — there is no admin-side
// direct create. Admins can still PATCH existing rows to fix data or
// promote/demote members.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, auth gin.HandlerFunc) {
	rg.GET("", h.list)
	rg.GET("/:id", h.get)
	rg.PATCH("/:id", auth, h.update)

	rg.GET("/:id/members", h.listMembers)
	rg.POST("/:id/members", auth, h.addMember)
	rg.PATCH("/:id/members/:userId", auth, h.updateMember)
	rg.DELETE("/:id/members/:userId", auth, h.removeMember)
}

// RegisterMeRoutes mounts caller-scoped org endpoints on /me. Separate
// from RegisterRoutes because /me isn't a subpath of /organizations —
// it's mounted on the v1 root by main.go.
func (h *Handler) RegisterMeRoutes(rg *gin.RouterGroup, auth gin.HandlerFunc) {
	rg.GET("/organizations", auth, h.listMyMemberships)
}

func (h *Handler) listMyMemberships(c *gin.Context) {
	userID, _ := c.Get("userID")
	uid, _ := userID.(string)
	items, err := h.svc.ListMyMemberships(uid)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list your organizations")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"memberships": items})
}

func (h *Handler) list(c *gin.Context) {
	items, err := h.svc.List(ListFilters{
		Kind:         strings.ToUpper(c.Query("kind")),
		Jurisdiction: strings.ToUpper(c.Query("jurisdiction")),
		State:        c.Query("state"),
		LGA:          c.Query("lga"),
		Search:       strings.ToLower(c.Query("q")),
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list organizations")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"organizations": items})
}

func (h *Handler) get(c *gin.Context) {
	item, err := h.svc.Get(c.Param("id"))
	if handled := handleAppErr(c, err); handled {
		return
	}
	response.Success(c, http.StatusOK, gin.H{"organization": item})
}

func (h *Handler) update(c *gin.Context) {
	orgID := c.Param("id")
	userID, _ := c.Get("userID")
	userRole, _ := c.Get("userRole")
	if err := h.svc.CanAdmin(orgID, userID.(string), asString(userRole)); err != nil {
		handleAppErr(c, err)
		return
	}
	// Capture the pre-change verified state so we can log a distinct
	// org.verified / org.unverified action separate from the plain
	// org.updated. The verify badge is a citizen-facing trust signal,
	// so its flip is worth its own action name in the audit log.
	previous, _ := h.svc.Get(orgID)
	var input UpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	item, err := h.svc.Update(orgID, input)
	if handled := handleAppErr(c, err); handled {
		return
	}
	actor := audit.FromContext(c)
	if input.Verified != nil && previous != nil && previous.Verified != *input.Verified {
		action := "org.verified"
		if !*input.Verified {
			action = "org.unverified"
		}
		h.auditor.Log(audit.Entry{
			Actor:      actor,
			Action:     action,
			TargetType: "ORGANIZATION",
			TargetID:   orgID,
			Metadata: map[string]any{
				"name": item.Name,
				"slug": item.Slug,
			},
			Request: c.Request,
		})
	} else {
		h.auditor.Log(audit.Entry{
			Actor:      actor,
			Action:     "org.updated",
			TargetType: "ORGANIZATION",
			TargetID:   orgID,
			Metadata: map[string]any{
				"name":            item.Name,
				"fieldsSubmitted": listChangedFields(input),
			},
			Request: c.Request,
		})
	}
	response.Success(c, http.StatusOK, gin.H{"organization": item})
}

// listChangedFields returns the names of the UpdateInput fields the
// caller supplied — a lightweight summary of "what was actually
// touched" so the audit log doesn't dump the full request body.
func listChangedFields(in UpdateInput) []string {
	fields := []string{}
	if in.Name != nil {
		fields = append(fields, "name")
	}
	if in.Kind != nil {
		fields = append(fields, "kind")
	}
	if in.Jurisdiction != nil {
		fields = append(fields, "jurisdiction")
	}
	if in.State != nil {
		fields = append(fields, "state")
	}
	if in.LGA != nil {
		fields = append(fields, "lga")
	}
	if in.Description != nil {
		fields = append(fields, "description")
	}
	if in.LogoURL != nil {
		fields = append(fields, "logoUrl")
	}
	if in.Email != nil {
		fields = append(fields, "email")
	}
	if in.Phone != nil {
		fields = append(fields, "phone")
	}
	if in.Website != nil {
		fields = append(fields, "website")
	}
	if in.Verified != nil {
		fields = append(fields, "verified")
	}
	return fields
}

func (h *Handler) listMembers(c *gin.Context) {
	items, err := h.svc.ListMembers(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list members")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"members": items})
}

func (h *Handler) addMember(c *gin.Context) {
	orgID := c.Param("id")
	userID, _ := c.Get("userID")
	userRole, _ := c.Get("userRole")
	if err := h.svc.CanAdmin(orgID, userID.(string), asString(userRole)); err != nil {
		handleAppErr(c, err)
		return
	}
	var input AddMemberInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	m, err := h.svc.AddMember(orgID, input)
	if handled := handleAppErr(c, err); handled {
		return
	}
	h.auditor.Log(audit.Entry{
		Actor:      audit.FromContext(c),
		Action:     "org.member_added",
		TargetType: "ORGANIZATION",
		TargetID:   orgID,
		Metadata: map[string]any{
			"memberUserId": input.UserID,
			"memberEmail":  input.UserName,
			"role":         input.Role,
		},
		Request: c.Request,
	})
	response.Success(c, http.StatusCreated, gin.H{"member": m})
}

func (h *Handler) updateMember(c *gin.Context) {
	orgID := c.Param("id")
	targetUserID := c.Param("userId")
	userID, _ := c.Get("userID")
	userRole, _ := c.Get("userRole")
	if err := h.svc.CanAdmin(orgID, userID.(string), asString(userRole)); err != nil {
		handleAppErr(c, err)
		return
	}
	var input UpdateMemberInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	if err := h.svc.UpdateMember(orgID, targetUserID, input); handleAppErr(c, err) {
		return
	}
	h.auditor.Log(audit.Entry{
		Actor:      audit.FromContext(c),
		Action:     "org.member_role_changed",
		TargetType: "ORGANIZATION",
		TargetID:   orgID,
		Metadata: map[string]any{
			"memberUserId": targetUserID,
			"newRole":      input.Role,
		},
		Request: c.Request,
	})
	response.Success(c, http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) removeMember(c *gin.Context) {
	orgID := c.Param("id")
	targetUserID := c.Param("userId")
	userID, _ := c.Get("userID")
	userRole, _ := c.Get("userRole")
	if err := h.svc.CanAdmin(orgID, userID.(string), asString(userRole)); err != nil {
		handleAppErr(c, err)
		return
	}
	if err := h.svc.RemoveMember(orgID, targetUserID); handleAppErr(c, err) {
		return
	}
	h.auditor.Log(audit.Entry{
		Actor:      audit.FromContext(c),
		Action:     "org.member_removed",
		TargetType: "ORGANIZATION",
		TargetID:   orgID,
		Metadata:   map[string]any{"memberUserId": targetUserID},
		Request:    c.Request,
	})
	response.Success(c, http.StatusOK, gin.H{"ok": true})
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
