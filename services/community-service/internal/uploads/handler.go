package uploads

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/civicos/community-service/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	maxBytes    = 5 * 1024 * 1024
	maxFileType = "image/"
)

var allowedExts = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webp": true,
}

type Handler struct {
	dir string
}

func NewHandler(dir string) *Handler { return &Handler{dir: dir} }

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, auth, verified gin.HandlerFunc) {
	rg.POST("", auth, verified, h.upload)
	rg.GET("/:filename", h.serve)
}

func (h *Handler) upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "FILE_REQUIRED", "Attach a file under the 'file' field")
		return
	}
	if file.Size > maxBytes {
		response.Error(c, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", "Files must be 5MB or smaller")
		return
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowedExts[ext] {
		response.Error(c, http.StatusUnsupportedMediaType, "INVALID_FILE_TYPE", "Only JPG, PNG, GIF, and WEBP images are allowed")
		return
	}
	if ct := file.Header.Get("Content-Type"); !strings.HasPrefix(ct, maxFileType) {
		response.Error(c, http.StatusUnsupportedMediaType, "INVALID_FILE_TYPE", "Only image files are allowed")
		return
	}

	name := uuid.New().String() + ext
	path := filepath.Join(h.dir, name)
	if err := c.SaveUploadedFile(file, path); err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to save file")
		return
	}

	response.Success(c, http.StatusCreated, gin.H{"filename": name})
}

func (h *Handler) serve(c *gin.Context) {
	name := c.Param("filename")
	if strings.ContainsAny(name, "/\\") || strings.Contains(name, "..") {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	c.File(filepath.Join(h.dir, name))
}
