package petitions

import (
	"log"
	"net/http"

	"github.com/civicos/community-service/internal/domain"
	"github.com/civicos/community-service/pkg/response"
	"github.com/gin-gonic/gin"
)

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
	rg.POST("/:id/sign", auth, h.sign)
	rg.GET("/:id/comments", h.listComments)
	rg.POST("/:id/comments", auth, h.addComment)
}

func (h *Handler) list(c *gin.Context) {
	items, err := h.svc.List(c.Query("communityId"), c.Query("status"))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch petitions")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"petitions": items})
}

func (h *Handler) get(c *gin.Context) {
	item, err := h.svc.Get(c.Param("id"))
	if err != nil {
		if appErr, ok := err.(*AppError); ok {
			response.Error(c, appErr.Status, appErr.Code, appErr.Message)
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch petition")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"petition": item})
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
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create petition")
		return
	}
	response.Success(c, http.StatusCreated, gin.H{"petition": item})
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
	petitionID := c.Param("id")
	item, err := h.svc.AddComment(petitionID, userID.(string), name, role, input.Content)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to add comment")
		return
	}

	if h.notifier != nil {
		if p, gerr := h.svc.Get(petitionID); gerr == nil && p.CreatedByID != "" && p.CreatedByID != userID.(string) {
			link := "/petitions/" + petitionID
			if nerr := h.notifier.Emit(
				p.CreatedByID,
				domain.NotificationPetitionUpdate,
				"New comment on your petition",
				name+" commented on \""+p.Title+"\"",
				&link,
			); nerr != nil {
				log.Printf("notify petition comment: %v", nerr)
			}
		}
	}

	response.Success(c, http.StatusCreated, gin.H{"comment": item})
}

func (h *Handler) sign(c *gin.Context) {
	userID, _ := c.Get("userID")
	if err := h.svc.Sign(c.Param("id"), userID.(string)); err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to sign petition")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"signed": true})
}
