package issues

import (
	"errors"
	"log"
	"net/http"

	"github.com/civicos/community-service/internal/domain"
	"github.com/civicos/community-service/pkg/response"
	"github.com/gin-gonic/gin"
)

// Notifier is the minimal interface this package needs to emit notifications.
// notifications.Service satisfies it without an import cycle.
type Notifier interface {
	Emit(userID string, t domain.NotificationType, title, body string, linkURL *string) error
}

type Handler struct {
	svc      *Service
	notifier Notifier
}

func NewHandler(svc *Service, notifier Notifier) *Handler {
	return &Handler{svc: svc, notifier: notifier}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, auth gin.HandlerFunc) {
	rg.GET("", h.list)
	rg.GET("/:id", h.get)
	rg.POST("", auth, h.create)
	rg.POST("/:id/upvote", auth, h.upvote)
	rg.PATCH("/:id/status", auth, h.updateStatus)
	rg.GET("/:id/comments", h.listComments)
	rg.POST("/:id/comments", auth, h.addComment)
}

func (h *Handler) list(c *gin.Context) {
	items, err := h.svc.List(c.Query("communityId"), c.Query("status"), c.Query("category"))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch issues")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"issues": items})
}

func (h *Handler) get(c *gin.Context) {
	item, err := h.svc.Get(c.Param("id"))
	if err != nil {
		var appErr *AppError
		if errors.As(err, &appErr) {
			response.Error(c, appErr.Status, appErr.Code, appErr.Message)
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch issue")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"issue": item})
}

func (h *Handler) create(c *gin.Context) {
	var input CreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	userID, _ := c.Get("userID")
	item, err := h.svc.Create(input, userID.(string))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create issue")
		return
	}
	response.Success(c, http.StatusCreated, gin.H{"issue": item})
}

func (h *Handler) upvote(c *gin.Context) {
	if err := h.svc.Upvote(c.Param("id")); err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to upvote")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"upvoted": true})
}

func (h *Handler) listComments(c *gin.Context) {
	items, err := h.svc.ListComments(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch comments")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"comments": items})
}

func (h *Handler) addComment(c *gin.Context) {
	var input CommentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	userID, _ := c.Get("userID")
	userName, _ := c.Get("userName")
	userRole, _ := c.Get("userRole")
	name, _ := userName.(string)
	role, _ := userRole.(string)
	if name == "" {
		name = "Anonymous"
	}
	if role == "" {
		role = "CITIZEN"
	}
	issueID := c.Param("id")
	item, err := h.svc.AddComment(issueID, userID.(string), name, role, input.Content)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to add comment")
		return
	}

	if h.notifier != nil {
		if issue, gerr := h.svc.Get(issueID); gerr == nil && issue.ReportedByID != "" && issue.ReportedByID != userID.(string) {
			link := "/issues/" + issueID + "#comments"
			if nerr := h.notifier.Emit(
				issue.ReportedByID,
				domain.NotificationIssueUpdate,
				"New comment on your issue",
				name+" commented on \""+issue.Title+"\"",
				&link,
			); nerr != nil {
				log.Printf("notify issue comment: %v", nerr)
			}
		}
	}

	response.Success(c, http.StatusCreated, gin.H{"comment": item})
}

func (h *Handler) updateStatus(c *gin.Context) {
	var body struct {
		Status domain.IssueStatus `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	issueID := c.Param("id")
	if err := h.svc.UpdateStatus(issueID, body.Status); err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update status")
		return
	}

	if h.notifier != nil {
		actorID, _ := c.Get("userID")
		if issue, gerr := h.svc.Get(issueID); gerr == nil && issue.ReportedByID != "" && issue.ReportedByID != actorID {
			link := "/issues/" + issueID
			if nerr := h.notifier.Emit(
				issue.ReportedByID,
				domain.NotificationIssueUpdate,
				"Issue status updated",
				"\""+issue.Title+"\" is now "+string(body.Status),
				&link,
			); nerr != nil {
				log.Printf("notify issue status: %v", nerr)
			}
		}
	}

	response.Success(c, http.StatusOK, gin.H{"updated": true})
}
