package draft

import (
	"log"
	"net/http"

	"github.com/civicos/civicai-service/pkg/response"
	"github.com/gin-gonic/gin"
)

// staffRoles mirrors the summarize handler + the FE STAFF_ROLES set.
// Citizens shouldn't be able to burn Gemini calls drafting announcements
// they can't publish.
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
	rg.POST("/draft-announcement", h.draft)
}

func (h *Handler) draft(c *gin.Context) {
	role, _ := c.Get("userRole")
	roleStr, _ := role.(string)
	if _, ok := staffRoles[roleStr]; !ok {
		response.Error(c, http.StatusForbidden, "FORBIDDEN",
			"CivicAI drafting is available to representatives and organization staff.")
		return
	}

	var input DraftInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	out, err := h.svc.Draft(c.Request.Context(), input)
	if err != nil {
		log.Printf("[draft] gemini call failed: %v", err)
		response.Error(c, http.StatusBadGateway, "AI_UNAVAILABLE",
			"CivicAI could not draft the announcement right now. Try again in a moment.")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"draft": out})
}
