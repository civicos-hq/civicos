package uploads

import (
	"net/http"

	"github.com/civicos/community-service/pkg/response"
	"github.com/gin-gonic/gin"
)

// VideoHandler owns POST /v1/uploads/video.
//
// It is a separate handler from the image one on purpose: the limits (10MB vs
// 5MB), the allowlists, and the rate-limit budget at the gateway all differ,
// and folding them into one endpoint would mean branching on the uploaded
// file's own claimed type to decide how strictly to check it. Serving is still
// shared — both kinds of object live in the same Storage and come back out
// through Handler.serve.
type VideoHandler struct {
	store Storage
}

func NewVideoHandler(store Storage) *VideoHandler { return &VideoHandler{store: store} }

func (h *VideoHandler) RegisterRoutes(rg *gin.RouterGroup, auth, verified gin.HandlerFunc) {
	rg.POST("/video", auth, verified, h.upload)
}

func (h *VideoHandler) upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "FILE_REQUIRED", "Attach a file under the 'file' field")
		return
	}
	if verr := validateVideo(file.Size, file.Filename, file.Header.Get("Content-Type")); verr != nil {
		response.Error(c, verr.Status, verr.Code, verr.Message)
		return
	}

	name, err := storeUpload(h.store, file)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to save file")
		return
	}

	// sizeBytes comes back so the client can persist it on the issue and show
	// "12.4 MB" next to the player — people on metered data decide whether to
	// press play before a byte of video moves.
	response.Success(c, http.StatusCreated, gin.H{"filename": name, "sizeBytes": file.Size})
}
