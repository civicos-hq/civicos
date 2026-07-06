package representatives

import (
	"errors"
	"log"
	"net/http"

	"github.com/civicos/community-service/internal/domain"
	"github.com/civicos/community-service/internal/middleware"
	"github.com/civicos/community-service/pkg/response"
	"github.com/gin-gonic/gin"
)

// Notifier is the minimal interface this package needs to emit notifications,
// satisfied by notifications.Service. Import via interface to avoid an import
// cycle.
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

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, auth, verified, requireRole gin.HandlerFunc) {
	rg.GET("", h.list)
	rg.GET("/:id", h.get)
	rg.POST("", auth, requireRole, h.create)
	rg.PATCH("/:id", auth, requireRole, h.update)
	rg.POST("/:id/follow", auth, verified, h.follow)
	rg.DELETE("/:id/follow", auth, h.unfollow)
	rg.GET("/:id/comments", h.listComments)
	rg.POST("/:id/comments", auth, verified, h.addComment)
}

// RegisterMeRoutes mounts user-scoped follow lookups under /me on the parent router.
func (h *Handler) RegisterMeRoutes(rg *gin.RouterGroup, auth gin.HandlerFunc) {
	rg.GET("/follows/representatives", auth, h.followedIDs)
}

func (h *Handler) list(c *gin.Context) {
	items, err := h.svc.List(c.Query("communityId"))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch representatives")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"representatives": items})
}

func (h *Handler) get(c *gin.Context) {
	item, err := h.svc.Get(c.Param("id"))
	if err != nil {
		var appErr *AppError
		if errors.As(err, &appErr) {
			response.Error(c, appErr.Status, appErr.Code, appErr.Message)
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch representative")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"representative": item})
}

func (h *Handler) follow(c *gin.Context) {
	userID, _ := c.Get("userID")
	rep, err := h.svc.Get(c.Param("id"))
	if err != nil {
		var appErr *AppError
		if errors.As(err, &appErr) {
			response.Error(c, appErr.Status, appErr.Code, appErr.Message)
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch representative")
		return
	}
	if !middleware.RequireActiveCommunityMatch(c, rep.CommunityID) {
		return
	}
	if err := h.svc.Follow(c.Param("id"), userID.(string)); err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to follow")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"following": true})
}

func (h *Handler) unfollow(c *gin.Context) {
	userID, _ := c.Get("userID")
	if err := h.svc.Unfollow(c.Param("id"), userID.(string)); err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to unfollow")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"following": false})
}

func (h *Handler) followedIDs(c *gin.Context) {
	userID, _ := c.Get("userID")
	ids, err := h.svc.FollowedIDs(userID.(string))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch follows")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"representativeIds": ids})
}

func (h *Handler) update(c *gin.Context) {
	var input UpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	item, err := h.svc.Update(c.Param("id"), input)
	if err != nil {
		var appErr *AppError
		if errors.As(err, &appErr) {
			response.Error(c, appErr.Status, appErr.Code, appErr.Message)
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update representative")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"representative": item})
}

func (h *Handler) create(c *gin.Context) {
	var input CreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	if !middleware.RequireActiveCommunityMatch(c, input.CommunityID) {
		return
	}
	userID, _ := c.Get("userID")
	item, err := h.svc.Create(input, userID.(string))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create representative")
		return
	}
	response.Success(c, http.StatusCreated, gin.H{"representative": item})
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
	repID := c.Param("id")
	rep, err := h.svc.Get(repID)
	if err != nil {
		var appErr *AppError
		if errors.As(err, &appErr) {
			response.Error(c, appErr.Status, appErr.Code, appErr.Message)
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch representative")
		return
	}
	if !middleware.RequireActiveCommunityMatch(c, rep.CommunityID) {
		return
	}

	item, err := h.svc.AddComment(repID, userID.(string), name, role, input.Content)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to add comment")
		return
	}

	// Fan out a notification to every follower (excluding the author) only on
	// an official response — citizens don't get pinged for every other citizen
	// posting on a rep's wall.
	if h.notifier != nil && item.IsOfficialResponse {
		followers, ferr := h.svc.FollowerIDs(repID)
		if ferr != nil {
			log.Printf("notify rep response: fetch followers: %v", ferr)
		} else {
			link := "/representatives/" + repID + "#comments"
			body := name + " responded on " + rep.Name + "'s page."
			for _, fid := range followers {
				if fid == userID.(string) {
					continue
				}
				if nerr := h.notifier.Emit(
					fid,
					domain.NotificationRepresentativeResponse,
					"New response from "+rep.Name,
					body,
					&link,
				); nerr != nil {
					log.Printf("notify rep response: %v", nerr)
				}
			}
		}
	}

	response.Success(c, http.StatusCreated, gin.H{"comment": item})
}
