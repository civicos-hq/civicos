package applications

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

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, auth gin.HandlerFunc) {
	rg.GET("/me", auth, h.getMe)
	rg.PUT("/me/representative", auth, h.upsertRepresentative)
	rg.PUT("/me/organization", auth, h.upsertOrganization)
}

func (h *Handler) RegisterAdminRoutes(rg *gin.RouterGroup, auth, requireAdmin gin.HandlerFunc) {
	rg.GET("", auth, requireAdmin, h.listAdmin)
	rg.GET("/:kind/:id", auth, requireAdmin, h.getAdmin)
	rg.PATCH("/:kind/:id", auth, requireAdmin, h.review)
}

func (h *Handler) getMe(c *gin.Context) {
	userID, _ := c.Get("userID")
	data, err := h.svc.GetMe(userID.(string))
	if handleAppErr(c, err) {
		return
	}
	response.Success(c, http.StatusOK, gin.H{"application": data})
}

func (h *Handler) upsertRepresentative(c *gin.Context) {
	var input RepresentativeApplicationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	userID, _ := c.Get("userID")
	app, err := h.svc.UpsertRepresentative(userID.(string), input)
	if handleAppErr(c, err) {
		return
	}
	response.Success(c, http.StatusOK, gin.H{"representativeApplication": app})
}

func (h *Handler) upsertOrganization(c *gin.Context) {
	var input OrganizationApplicationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	userID, _ := c.Get("userID")
	app, err := h.svc.UpsertOrganization(userID.(string), input)
	if handleAppErr(c, err) {
		return
	}
	response.Success(c, http.StatusOK, gin.H{"organizationApplication": app})
}

func (h *Handler) listAdmin(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	items, total, err := h.svc.ListAdmin(AdminListFilters{
		Kind:   c.Query("kind"),
		Status: c.Query("status"),
		Search: c.Query("q"),
		Limit:  limit,
		Offset: offset,
	})
	if handleAppErr(c, err) {
		return
	}
	response.Success(c, http.StatusOK, gin.H{"applications": items, "total": total})
}

func (h *Handler) getAdmin(c *gin.Context) {
	item, err := h.svc.GetAdmin(c.Param("kind"), c.Param("id"))
	if handleAppErr(c, err) {
		return
	}
	response.Success(c, http.StatusOK, gin.H{"application": item})
}

func (h *Handler) review(c *gin.Context) {
	var input ReviewInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	reviewerID, _ := c.Get("userID")
	item, err := h.svc.Review(c.Param("kind"), c.Param("id"), reviewerID.(string), input)
	if handleAppErr(c, err) {
		return
	}
	action := "application.approved"
	if input.Status == string(domain.ApprovalStatusRejected) {
		action = "application.rejected"
	}
	h.auditor.Log(audit.Entry{
		Actor:      audit.FromContext(c),
		Action:     action,
		TargetType: item.Kind + "_APPLICATION",
		TargetID:   item.ID,
		Metadata: map[string]any{
			"status":          input.Status,
			"note":            input.Note,
			"applicantUserId": item.Applicant.ID,
			"kind":            item.Kind,
		},
		Request: c.Request,
	})
	response.Success(c, http.StatusOK, gin.H{"application": item})
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
