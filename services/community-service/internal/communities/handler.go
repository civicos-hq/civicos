package communities

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/civicos/community-service/pkg/response"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc      *Service
	geocoder *Geocoder
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// WithGeocoder enables the location-suggestion endpoint. Optional: without
// it the route still exists but reports that lookup is unconfigured, so
// the admin UI can hide the affordance rather than offer a button that
// always fails.
func (h *Handler) WithGeocoder(g *Geocoder) *Handler {
	h.geocoder = g
	return h
}

// Roles permitted to create a community. Citizens browse and join; they
// don't author. Orgs (NGO role) also don't — civic geography is not
// theirs to add.
var communityCreatorRoles = []string{"GOVERNMENT_ADMIN", "PLATFORM_ADMIN"}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, auth, requireRole gin.HandlerFunc) {
	rg.GET("", h.list)
	rg.GET("/:id", h.get)
	rg.POST("", auth, requireRole, h.create)
	// Same roles as create: civic geography is not citizen-editable, and
	// coordinates decide who receives a flood warning.
	rg.PATCH("/:id", auth, requireRole, h.update)
	// Suggests a point for a state/LGA while an admin fills in the create
	// form. Same roles as create — it spends an upstream geocoding quota
	// and should not be an open lookup service.
	//
	// A static segment sharing a position with "/:id" is the shape Gin's
	// router historically panicked on, and a panic here means the service
	// never starts. It resolves correctly on this version; pinned by
	// TestGeocodeRouteDoesNotConflictWithIDRoute so an upgrade cannot
	// reintroduce it silently.
	rg.GET("/geocode", auth, requireRole, h.geocode)
}

func (h *Handler) geocode(c *gin.Context) {
	if h.geocoder == nil || !h.geocoder.Enabled() {
		response.Error(c, http.StatusServiceUnavailable, "GEOCODING_UNAVAILABLE",
			"Automatic location lookup is not configured. Enter coordinates manually.")
		return
	}
	suggestion, err := h.geocoder.Lookup(c.Request.Context(), c.Query("state"), c.Query("lga"))
	if err != nil {
		var appErr *AppError
		if errors.As(err, &appErr) {
			response.Error(c, appErr.Status, appErr.Code, appErr.Message)
			return
		}
		// Upstream failures are logged server-side, never echoed: the
		// message can name the project and the key restriction.
		response.Error(c, http.StatusBadGateway, "GEOCODING_FAILED",
			"Location lookup failed. Enter coordinates manually.")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"suggestion": suggestion})
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
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update community")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"community": item})
}

func (h *Handler) list(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))

	var ids []string
	if raw := strings.TrimSpace(c.Query("ids")); raw != "" {
		for _, id := range strings.Split(raw, ",") {
			if id = strings.TrimSpace(id); id != "" {
				ids = append(ids, id)
			}
		}
	}

	result, err := h.svc.List(SearchParams{
		Query:  c.Query("q"),
		State:  c.Query("state"),
		LGA:    c.Query("lga"),
		IDs:    ids,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch communities")
		return
	}
	response.Success(c, http.StatusOK, gin.H{
		"communities": result.Communities,
		"total":       result.Total,
		"limit":       result.Limit,
		"offset":      result.Offset,
	})
}

func (h *Handler) get(c *gin.Context) {
	item, err := h.svc.Get(c.Param("id"))
	if err != nil {
		var appErr *AppError
		if errors.As(err, &appErr) {
			response.Error(c, appErr.Status, appErr.Code, appErr.Message)
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch community")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"community": item})
}

func (h *Handler) create(c *gin.Context) {
	var input CreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	userID, _ := c.Get("userID")
	item, err := h.svc.Create(input, userID.(string))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create community")
		return
	}
	response.Success(c, http.StatusCreated, gin.H{"community": item})
}
