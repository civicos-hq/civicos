package summarize

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/civicos/civicai-service/pkg/response"
	"github.com/gin-gonic/gin"
)

// staffRoles mirrors the FE STAFF_ROLES set. Only these can burn a Gemini
// call to summarize a thread — citizens don't get the button. The gateway
// gates auth; this handler gates the role.
var staffRoles = map[string]struct{}{
	"REPRESENTATIVE":   {},
	"GOVERNMENT_ADMIN": {},
	"PLATFORM_ADMIN":   {},
	"NGO":              {},
	"MODERATOR":        {},
}

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/summarize", h.summarize)
}

func (h *Handler) summarize(c *gin.Context) {
	role, _ := c.Get("userRole")
	roleStr, _ := role.(string)
	if _, ok := staffRoles[roleStr]; !ok {
		response.Error(c, http.StatusForbidden, "FORBIDDEN",
			"CivicAI summaries are available to representatives and organization staff.")
		return
	}

	var input SummarizeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	bearer := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")

	out, err := h.svc.Summarize(c.Request.Context(), input, bearer)
	if err != nil {
		var srcErr *SourceError
		if errors.As(err, &srcErr) {
			// Propagate meaningful upstream statuses (404 for missing, 403
			// for unauthorized reads) rather than hiding everything behind
			// AI_UNAVAILABLE — the FE messaging is different.
			code := "UPSTREAM_ERROR"
			if srcErr.Status == http.StatusNotFound {
				code = "RESOURCE_NOT_FOUND"
			} else if srcErr.Status == http.StatusForbidden {
				code = "RESOURCE_FORBIDDEN"
			}
			response.Error(c, srcErr.Status, code, srcErr.Message)
			return
		}
		log.Printf("[summarize] gemini call failed: %v", err)
		response.Error(c, http.StatusBadGateway, "AI_UNAVAILABLE",
			"CivicAI could not summarize this thread right now. Try again in a moment.")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"summary": out})
}
