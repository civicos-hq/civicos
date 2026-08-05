package repannouncements

import (
	"errors"
	"net/http"

	"github.com/civicos/community-service/pkg/response"
	"github.com/gin-gonic/gin"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// RegisterRoutes mounts under /representatives.
//
// The public list is unauthenticated: a constituent should be able to read
// what their representative said without an account, the same as every other
// public record on CivicOS.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, auth, verified gin.HandlerFunc) {
	rg.GET("/:id/announcements", h.listPublic)
	rg.GET("/:id/announcements/manage", auth, h.listMine)
	rg.POST("/:id/announcements", auth, h.create)
	rg.PATCH("/:id/announcements/:annId", auth, h.update)
	rg.POST("/:id/announcements/:annId/publish", auth, h.publish)
	rg.POST("/:id/announcements/:annId/archive", auth, h.archive)
	rg.DELETE("/:id/announcements/:annId", auth, h.remove)
	// The thread under a published announcement. Reading is open to anyone
	// who can read the announcement; posting needs a verified account, the
	// same bar as every other thread on CivicOS.
	rg.GET("/:id/announcements/:annId/comments", h.listComments)
	rg.POST("/:id/announcements/:annId/comments", auth, verified, h.addComment)
}

func fail(c *gin.Context, err error, fallback string) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		response.Error(c, appErr.Status, appErr.Code, appErr.Message)
		return
	}
	response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", fallback)
}

func actor(c *gin.Context) (id, name string) {
	uid, _ := c.Get("userID")
	uname, _ := c.Get("userName")
	s, _ := uid.(string)
	n, _ := uname.(string)
	return s, n
}

func (h *Handler) listPublic(c *gin.Context) {
	items, err := h.svc.ListPublic(c.Param("id"))
	if err != nil {
		fail(c, err, "Failed to load announcements")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"announcements": items})
}

func (h *Handler) listMine(c *gin.Context) {
	uid, _ := actor(c)
	items, err := h.svc.ListMine(c.Param("id"), uid)
	if err != nil {
		fail(c, err, "Failed to load announcements")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"announcements": items})
}

func (h *Handler) create(c *gin.Context) {
	var in CreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	uid, uname := actor(c)
	item, err := h.svc.Create(c.Param("id"), uid, uname, in)
	if err != nil {
		fail(c, err, "Failed to create announcement")
		return
	}
	response.Success(c, http.StatusCreated, gin.H{"announcement": item})
}

func (h *Handler) update(c *gin.Context) {
	var in UpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	uid, _ := actor(c)
	item, err := h.svc.Update(c.Param("id"), c.Param("annId"), uid, in)
	if err != nil {
		fail(c, err, "Failed to update announcement")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"announcement": item})
}

func (h *Handler) publish(c *gin.Context) {
	uid, _ := actor(c)
	item, err := h.svc.Publish(c.Param("id"), c.Param("annId"), uid)
	if err != nil {
		fail(c, err, "Failed to publish announcement")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"announcement": item})
}

func (h *Handler) archive(c *gin.Context) {
	uid, _ := actor(c)
	item, err := h.svc.Archive(c.Param("id"), c.Param("annId"), uid)
	if err != nil {
		fail(c, err, "Failed to archive announcement")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"announcement": item})
}

func (h *Handler) remove(c *gin.Context) {
	uid, _ := actor(c)
	if err := h.svc.Delete(c.Param("id"), c.Param("annId"), uid); err != nil {
		fail(c, err, "Failed to delete announcement")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) listComments(c *gin.Context) {
	items, err := h.svc.ListComments(c.Param("id"), c.Param("annId"))
	if err != nil {
		fail(c, err, "Failed to load comments")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"comments": items})
}

func (h *Handler) addComment(c *gin.Context) {
	var in CommentInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	uid, uname := actor(c)
	role, _ := c.Get("userRole")
	roleStr, _ := role.(string)
	if uname == "" {
		uname = "Anonymous"
	}
	item, err := h.svc.AddComment(c.Param("id"), c.Param("annId"), uid, uname, roleStr, in.Content)
	if err != nil {
		fail(c, err, "Failed to post comment")
		return
	}
	response.Success(c, http.StatusCreated, gin.H{"comment": item})
}
