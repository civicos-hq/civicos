package organizations

import (
	"errors"
	"net/http"
	"strings"

	"github.com/civicos/organization-service/pkg/response"
	"github.com/gin-gonic/gin"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Service() *Service { return h.svc }

// Roles allowed to create a top-level org from scratch. Once created, the
// org's internal role (OWNER/ADMIN) governs edits — not the JWT role.
var orgCreatorRoles = []string{"GOVERNMENT_ADMIN", "PLATFORM_ADMIN", "NGO"}

var _ = orgCreatorRoles // referenced from main.go via RegisterRoutes callers

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, auth, requireCreator gin.HandlerFunc) {
	rg.GET("", h.list)
	rg.GET("/:id", h.get)
	rg.POST("", auth, requireCreator, h.create)
	rg.PATCH("/:id", auth, h.update)

	rg.GET("/:id/members", h.listMembers)
	rg.POST("/:id/members", auth, h.addMember)
	rg.PATCH("/:id/members/:userId", auth, h.updateMember)
	rg.DELETE("/:id/members/:userId", auth, h.removeMember)
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

func (h *Handler) create(c *gin.Context) {
	var input CreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	userID, _ := c.Get("userID")
	userName, _ := c.Get("userName")
	userRole, _ := c.Get("userRole")
	item, err := h.svc.Create(input, userID.(string), asString(userName), asString(userRole))
	if handled := handleAppErr(c, err); handled {
		return
	}
	response.Success(c, http.StatusCreated, gin.H{"organization": item})
}

func (h *Handler) update(c *gin.Context) {
	orgID := c.Param("id")
	userID, _ := c.Get("userID")
	userRole, _ := c.Get("userRole")
	if err := h.svc.CanAdmin(orgID, userID.(string), asString(userRole)); err != nil {
		handleAppErr(c, err)
		return
	}
	var input UpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	item, err := h.svc.Update(orgID, input)
	if handled := handleAppErr(c, err); handled {
		return
	}
	response.Success(c, http.StatusOK, gin.H{"organization": item})
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
