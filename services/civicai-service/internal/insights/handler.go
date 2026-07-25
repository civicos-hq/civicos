package insights

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/civicos/civicai-service/pkg/response"
	"github.com/gin-gonic/gin"
)

// staffRoles mirrors summarize + draft. Community-wide insights are a
// staff decision-support surface, not a citizen-facing feature.
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
	rg.GET("/community-insights", h.generate)
}

func (h *Handler) generate(c *gin.Context) {
	role, _ := c.Get("userRole")
	roleStr, _ := role.(string)
	if _, ok := staffRoles[roleStr]; !ok {
		response.Error(c, http.StatusForbidden, "FORBIDDEN",
			"CivicAI insights are available to representatives and organization staff.")
		return
	}

	communityID := strings.TrimSpace(c.Query("communityId"))
	if communityID == "" {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "communityId is required")
		return
	}

	bearer := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")

	out, err := h.svc.Generate(c.Request.Context(), communityID, bearer)
	if err != nil {
		var srcErr *SourceError
		if errors.As(err, &srcErr) {
			log.Printf("[insights] upstream error: %v", srcErr)
			response.Error(c, srcErr.Status, "UPSTREAM_ERROR", srcErr.Message)
			return
		}
		log.Printf("[insights] gemini call failed: %v", err)
		response.Error(c, http.StatusBadGateway, "AI_UNAVAILABLE",
			"CivicAI could not generate community insights right now. Try again shortly.")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"insights": out})
}
