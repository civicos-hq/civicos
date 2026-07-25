package classify

import (
	"log"
	"net/http"

	"github.com/civicos/civicai-service/pkg/response"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// RegisterRoutes mounts the classification endpoint. Route auth is applied
// by the parent router — this handler assumes JWTAuth has already run.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/classify-issue", h.classifyIssue)
}

func (h *Handler) classifyIssue(c *gin.Context) {
	var input ClassifyInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	out, err := h.svc.Classify(c.Request.Context(), input)
	if err != nil {
		// Log the full error for operators; return a generic message to the
		// client so we don't leak Gemini quota / auth details.
		log.Printf("[classify] gemini call failed: %v", err)
		response.Error(c, http.StatusBadGateway, "AI_UNAVAILABLE",
			"CivicAI could not classify this issue right now. Please pick a category manually.")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"classification": out})
}
