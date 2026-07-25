package narrate

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/civicos/civicai-service/pkg/response"
	"github.com/gin-gonic/gin"
)

// Analytics narration is tighter than the other CivicAI surfaces:
// PLATFORM_ADMIN only. Community narration (scope=community) would open
// to the wider staff set, but that scope isn't implemented yet.
var platformNarratorRoles = map[string]struct{}{
	"PLATFORM_ADMIN": {},
}

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/narrate-metrics", h.narrate)
}

func (h *Handler) narrate(c *gin.Context) {
	scope := strings.TrimSpace(c.Query("scope"))
	if scope == "" {
		scope = "platform"
	}
	if scope != "platform" {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "unsupported scope; only 'platform' is implemented")
		return
	}

	role, _ := c.Get("userRole")
	roleStr, _ := role.(string)
	if _, ok := platformNarratorRoles[roleStr]; !ok {
		response.Error(c, http.StatusForbidden, "FORBIDDEN",
			"Platform narration is available to platform administrators only.")
		return
	}

	bearer := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")

	out, err := h.svc.NarratePlatform(c.Request.Context(), bearer)
	if err != nil {
		var srcErr *SourceError
		if errors.As(err, &srcErr) {
			log.Printf("[narrate] upstream error: %v", srcErr)
			response.Error(c, srcErr.Status, "UPSTREAM_ERROR", srcErr.Message)
			return
		}
		log.Printf("[narrate] gemini call failed: %v", err)
		response.Error(c, http.StatusBadGateway, "AI_UNAVAILABLE",
			"CivicAI could not narrate metrics right now. Try again shortly.")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"narration": out})
}
