package uploads

import (
	"errors"
	"mime/multipart"
	"net/http"

	"github.com/civicos/community-service/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler owns image uploads and serving for every stored object (images and
// videos alike — they share one namespace and one Storage).
type Handler struct {
	store Storage
}

func NewHandler(store Storage) *Handler { return &Handler{store: store} }

// RegisterRoutes mounts the image upload and the shared serve route. The video
// upload route lives on VideoHandler and is registered separately.
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
	if verr := validateImage(file.Size, file.Filename, file.Header.Get("Content-Type")); verr != nil {
		response.Error(c, verr.Status, verr.Code, verr.Message)
		return
	}

	name, err := storeUpload(h.store, file)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to save file")
		return
	}

	response.Success(c, http.StatusCreated, gin.H{"filename": name})
}

// serve streams a stored object. http.ServeContent (rather than a plain copy)
// gives us conditional requests and — the reason it matters here — Range
// support, so a browser can seek within a video without pulling the whole file.
func (h *Handler) serve(c *gin.Context) {
	name := c.Param("filename")
	if !safeStoredName(name) {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	f, err := h.store.Open(name)
	if errors.Is(err, ErrNotFound) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// ServeContent only sniffs when Content-Type is unset, so setting it here
	// wins for the extensions Go's MIME table doesn't know (see contentTypeFor).
	if ct := contentTypeFor[extOf(name)]; ct != "" {
		c.Header("Content-Type", ct)
	}
	http.ServeContent(c.Writer, c.Request, name, info.ModTime(), f)
}

// storeUpload generates the storage name and streams the multipart part into
// Storage. The caller's filename is used only for its extension — never as a
// path component.
func storeUpload(store Storage, file *multipart.FileHeader) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	name := uuid.New().String() + extOf(file.Filename)
	if err := store.Save(name, src); err != nil {
		return "", err
	}
	return name, nil
}
