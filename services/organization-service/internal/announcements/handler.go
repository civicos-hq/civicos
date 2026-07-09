package announcements

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/civicos/organization-service/internal/audit"
	"github.com/civicos/organization-service/internal/notifications"
	"github.com/civicos/organization-service/internal/organizations"
	"github.com/civicos/organization-service/pkg/response"
	"github.com/gin-gonic/gin"
)

// Notifier is the minimal surface this handler needs. notifications.DBNotifier
// satisfies it — kept as an interface so tests can substitute a fake.
type Notifier interface {
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

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, auth gin.HandlerFunc) {
	// Anyone can read published feed.
	rg.GET("/announcements", h.listGlobal)
	rg.GET("/organizations/:id/announcements", h.listByOrg)
	rg.GET("/announcements/:announcementId", h.get)

	// Writes require org membership (admin/owner).
	rg.POST("/organizations/:id/announcements", auth, h.create)
	rg.PATCH("/announcements/:announcementId", auth, h.update)
	rg.POST("/announcements/:announcementId/publish", auth, h.publish)
	rg.POST("/announcements/:announcementId/archive", auth, h.archive)
	rg.DELETE("/announcements/:announcementId", auth, h.delete)
}

func (h *Handler) listGlobal(c *gin.Context) {
	limit := 20
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 && l <= 100 {
		limit = l
	}
	items, err := h.svc.ListPublishedGlobal(limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list announcements")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"announcements": items})
}

func (h *Handler) listByOrg(c *gin.Context) {
	orgID := c.Param("id")
	includeDrafts := false
	// Only members see drafts. If the caller isn't authed at all, they only
	// see published — we don't 401 on read.
	if userID, ok := c.Get("userID"); ok {
		if _, err := h.orgs.IsMember(orgID, userID.(string)); err == nil {
			includeDrafts = true
		}
	}
	items, err := h.svc.ListByOrg(orgID, includeDrafts)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list announcements")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"announcements": items})
}

func (h *Handler) get(c *gin.Context) {
	item, err := h.svc.Get(c.Param("announcementId"))
	if handleAppErr(c, err) {
		return
	}
	response.Success(c, http.StatusOK, gin.H{"announcement": item})
}

func (h *Handler) create(c *gin.Context) {
	orgID := c.Param("id")
	userID, _ := c.Get("userID")
	userName, _ := c.Get("userName")
	userRole, _ := c.Get("userRole")
	if err := h.orgs.CanAdmin(orgID, userID.(string), asString(userRole)); err != nil {
		handleAppErr(c, err)
		return
	}
	var input CreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	item, err := h.svc.Create(orgID, input, userID.(string), asString(userName))
	if handleAppErr(c, err) {
		return
	}
	// Immediate-publish path shares the fan-out helper with the dedicated
	// /publish endpoint so both entry points produce identical notifications.
	if input.Publish {
		h.notifyPublished(item.OrganizationID, item.ID, item.Title, item.Body)
	}
	response.Success(c, http.StatusCreated, gin.H{"announcement": item})
}

func (h *Handler) update(c *gin.Context) {
	id := c.Param("announcementId")
	a, err := h.svc.Get(id)
	if handleAppErr(c, err) {
		return
	}
	userID, _ := c.Get("userID")
	userRole, _ := c.Get("userRole")
	if err := h.orgs.CanAdmin(a.OrganizationID, userID.(string), asString(userRole)); err != nil {
		handleAppErr(c, err)
		return
	}
	var input UpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	item, err := h.svc.Update(id, input)
	if handleAppErr(c, err) {
		return
	}
	response.Success(c, http.StatusOK, gin.H{"announcement": item})
}

func (h *Handler) publish(c *gin.Context) {
	id := c.Param("announcementId")
	a, err := h.svc.Get(id)
	if handleAppErr(c, err) {
		return
	}
	userID, _ := c.Get("userID")
	userRole, _ := c.Get("userRole")
	if err := h.orgs.CanAdmin(a.OrganizationID, userID.(string), asString(userRole)); err != nil {
		handleAppErr(c, err)
		return
	}
	item, err := h.svc.Publish(id)
	if handleAppErr(c, err) {
		return
	}
	h.auditor.Log(audit.Entry{
		Actor:      audit.FromContext(c),
		Action:     "announcement.published",
		TargetType: "ANNOUNCEMENT",
		TargetID:   id,
		Metadata:   map[string]any{"orgId": a.OrganizationID, "title": a.Title},
		Request:    c.Request,
	})
	h.notifyPublished(item.OrganizationID, item.ID, item.Title, item.Body)
	response.Success(c, http.StatusOK, gin.H{"announcement": item})
}

// notifyPublished fans out to every member of the org that an announcement
// went live. Kept as a helper so both the publish endpoint and the
// create-with-publish path in `create` can share the emit logic.
//
// Failures inside the notifier are logged by the underlying DBNotifier
// but never bubbled up — a publish that succeeds shouldn't fail because
// notification delivery hiccupped.
func (h *Handler) notifyPublished(orgID, announcementID, title, body string) {
	if h.notifier == nil {
		return
	}
	members, err := h.orgs.ListMembers(orgID)
	if err != nil {
		log.Printf("notify announcement.published: list members: %v", err)
		return
	}
	userIDs := make([]string, 0, len(members))
	for _, m := range members {
		userIDs = append(userIDs, m.UserID)
	}
	link := "/announcements/" + announcementID
	preview := body
	if len(preview) > 200 {
		preview = preview[:200] + "…"
	}
	h.notifier.EmitMany(userIDs,
		notifications.TypeAnnouncementUpdate,
		"New announcement: "+title,
		preview,
		&link,
	)
}

func (h *Handler) archive(c *gin.Context) {
	id := c.Param("announcementId")
	a, err := h.svc.Get(id)
	if handleAppErr(c, err) {
		return
	}
	userID, _ := c.Get("userID")
	userRole, _ := c.Get("userRole")
	// Archive is the emergency lever — platform admins retain this even
	// without org membership so a bad announcement can be pulled from
	// the public feed. See organizations.CanClose.
	if err := h.orgs.CanClose(a.OrganizationID, userID.(string), asString(userRole)); err != nil {
		handleAppErr(c, err)
		return
	}
	item, err := h.svc.Archive(id)
	if handleAppErr(c, err) {
		return
	}
	h.auditor.Log(audit.Entry{
		Actor:      audit.FromContext(c),
		Action:     "announcement.archived",
		TargetType: "ANNOUNCEMENT",
		TargetID:   id,
		Metadata:   map[string]any{"orgId": a.OrganizationID, "title": a.Title},
		Request:    c.Request,
	})
	response.Success(c, http.StatusOK, gin.H{"announcement": item})
}

func (h *Handler) delete(c *gin.Context) {
	id := c.Param("announcementId")
	a, err := h.svc.Get(id)
	if handleAppErr(c, err) {
		return
	}
	userID, _ := c.Get("userID")
	userRole, _ := c.Get("userRole")
	if err := h.orgs.CanAdmin(a.OrganizationID, userID.(string), asString(userRole)); err != nil {
		handleAppErr(c, err)
		return
	}
	if err := h.svc.Delete(id); handleAppErr(c, err) {
		return
	}
	h.auditor.Log(audit.Entry{
		Actor:      audit.FromContext(c),
		Action:     "announcement.deleted",
		TargetType: "ANNOUNCEMENT",
		TargetID:   id,
		Metadata:   map[string]any{"orgId": a.OrganizationID, "title": a.Title},
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
	// Also handle org-service errors that bubble up through CanAdmin/IsMember.
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
