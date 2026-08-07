package invitations

import (
	"errors"
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

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, auth gin.HandlerFunc) {
	rg.GET("/organizations/:id/invitations", auth, h.list)
	rg.POST("/organizations/:id/invitations", auth, h.create)
	rg.DELETE("/invitations/:invitationId", auth, h.revoke)

	// Unauthenticated: the invitee reads this before they have an account,
	// which is the whole point. Returns only the organization name, role
	// and invited address — see Preview.
	rg.GET("/invitations/:invitationId", h.preview)
	rg.POST("/invitations/:invitationId/accept", auth, h.accept)
}

// The token travels in the same path position as an ID on the
// authenticated routes, so Gin's router sees one parameter name. Naming it
// `invitationId` keeps the tree consistent; the public routes treat the
// value as an opaque token rather than a database identifier.
func token(c *gin.Context) string { return c.Param("invitationId") }

func actor(c *gin.Context) (id, name, role string) {
	uid, _ := c.Get("userID")
	uname, _ := c.Get("userName")
	urole, _ := c.Get("userRole")
	s, _ := uid.(string)
	n, _ := uname.(string)
	r, _ := urole.(string)
	return s, n, r
}

func fail(c *gin.Context, err error, fallback string) bool {
	if err == nil {
		return false
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		response.Error(c, appErr.Status, appErr.Code, appErr.Message)
		return true
	}
	response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", fallback)
	return true
}

func (h *Handler) list(c *gin.Context) {
	orgID := c.Param("id")
	uid, _, urole := actor(c)
	// Any member may see who is pending — knowing who has been asked to
	// join is not an admin privilege, and it stops two admins inviting the
	// same person twice.
	if err := h.orgs.CanReadInternal(orgID, uid, urole); err != nil {
		fail(c, err, "Failed to list invitations")
		return
	}
	items, err := h.svc.ListPending(orgID)
	if fail(c, err, "Failed to list invitations") {
		return
	}
	response.Success(c, http.StatusOK, gin.H{"invitations": items})
}

func (h *Handler) create(c *gin.Context) {
	orgID := c.Param("id")
	uid, uname, urole := actor(c)
	// Inviting decides who can act in the organization's name, so it sits
	// with the same people who can publish and run campaigns.
	if err := h.orgs.CanAdmin(orgID, uid, urole); err != nil {
		fail(c, err, "Failed to create invitation")
		return
	}

	var in CreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	inv, err := h.svc.Create(orgID, in, uid, uname)
	if fail(c, err, "Failed to create invitation") {
		return
	}

	h.auditor.Log(audit.Entry{
		Actor:      audit.FromContext(c),
		Action:     "org.member_invited",
		TargetType: "ORGANIZATION",
		TargetID:   orgID,
		Metadata: map[string]any{
			"invitationId": inv.ID,
			"email":        inv.Email,
			"role":         inv.Role,
			"title":        inv.Title,
		},
		Request: c.Request,
	})
	response.Success(c, http.StatusCreated, gin.H{"invitation": inv})
}

func (h *Handler) revoke(c *gin.Context) {
	invID := c.Param("invitationId")
	uid, _, urole := actor(c)

	orgID, err := h.svc.OrganizationOf(invID)
	if fail(c, err, "Failed to revoke invitation") {
		return
	}
	if err := h.orgs.CanAdmin(orgID, uid, urole); err != nil {
		fail(c, err, "Failed to revoke invitation")
		return
	}
	if fail(c, h.svc.Revoke(invID, uid), "Failed to revoke invitation") {
		return
	}

	h.auditor.Log(audit.Entry{
		Actor:      audit.FromContext(c),
		Action:     "org.invitation_revoked",
		TargetType: "ORGANIZATION",
		TargetID:   orgID,
		Metadata:   map[string]any{"invitationId": invID},
		Request:    c.Request,
	})
	response.Success(c, http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) preview(c *gin.Context) {
	p, err := h.svc.Preview(token(c))
	if fail(c, err, "Failed to load invitation") {
		return
	}
	response.Success(c, http.StatusOK, gin.H{"invitation": p})
}

func (h *Handler) accept(c *gin.Context) {
	uid, _, _ := actor(c)
	if uid == "" {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Sign in required")
		return
	}

	member, err := h.svc.Accept(token(c), uid)
	if fail(c, err, "Failed to accept invitation") {
		return
	}

	h.auditor.Log(audit.Entry{
		Actor:      audit.FromContext(c),
		Action:     "org.invitation_accepted",
		TargetType: "ORGANIZATION",
		TargetID:   member.OrganizationID,
		Metadata:   map[string]any{"userId": member.UserID, "role": member.Role},
		Request:    c.Request,
	})
	response.Success(c, http.StatusOK, gin.H{"member": member})
}
