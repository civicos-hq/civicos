package petitions

import (
	"log"
	"net/http"
	"strconv"

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
	petitionID := c.Param("id")
	res, err := h.svc.Sign(petitionID, userID.(string))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to sign petition")
		return
	}

	if res.Added && h.notifier != nil {
		if p, gerr := h.svc.Get(petitionID); gerr == nil && p.CreatedByID != "" {
			link := "/petitions/" + petitionID
			actor := userID.(string)

			if p.CreatedByID != actor {
				if nerr := h.notifier.Emit(
					p.CreatedByID,
					domain.NotificationPetitionUpdate,
					"New signature on your petition",
					"Your petition \""+p.Title+"\" reached "+itoa(res.NewCount)+" of "+itoa(p.Goal)+" signatures.",
					&link,
				); nerr != nil {
					log.Printf("notify petition sign: %v", nerr)
				}
			}

			if crossed := milestone(res.NewCount, p.Goal); crossed != "" {
				title := "Petition milestone: " + crossed
				body := "\"" + p.Title + "\" hit " + crossed + " of its goal (" + itoa(res.NewCount) + "/" + itoa(p.Goal) + ")."
				if nerr := h.notifier.Emit(
					p.CreatedByID,
					domain.NotificationPetitionUpdate,
					title,
					body,
					&link,
				); nerr != nil {
					log.Printf("notify petition milestone: %v", nerr)
				}
			}
		}
	}

	response.Success(c, http.StatusOK, gin.H{"signed": true})
}

// milestone returns a human-readable label ("25%", "50%", "100%") when newCount
// crosses one of those thresholds against goal, otherwise an empty string.
// A threshold is "crossed" when newCount-1 was below and newCount is at or above.
func milestone(newCount, goal int) string {
	if goal <= 0 {
		return ""
	}
	thresholds := []struct {
		label string
		pct   int
	}{
		{"25%", 25}, {"50%", 50}, {"100%", 100},
	}
	for _, t := range thresholds {
		need := (goal*t.pct + 99) / 100
		if newCount >= need && newCount-1 < need {
			return t.label
		}
	}
	return ""
}

func itoa(n int) string { return strconv.Itoa(n) }
