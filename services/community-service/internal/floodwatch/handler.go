package floodwatch

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/civicos/community-service/pkg/response"
	"github.com/gin-gonic/gin"
)

// staleAfter is how long a forecast stays displayable without the upstream
// reconfirming it.
//
// Six hours against an hourly poll: long enough to ride out a transient
// outage or a rate-limit blip, short enough that a warning cannot linger
// most of a day after the source stopped saying it. When rows age out the
// banner disappears — which reads as "no current forecast", never as "you
// are safe".
const staleAfter = 6 * time.Hour

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	// Public. A flood forecast is exactly the thing someone should be able
	// to read without an account, and every other public record on CivicOS
	// works the same way.
	rg.GET("/:id/flood-alerts", h.list)
}

func (h *Handler) list(c *gin.Context) {
	alerts, err := h.svc.ActiveForCommunity(c.Param("id"), staleAfter)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to load flood forecasts")
		return
	}
	response.Success(c, http.StatusOK, gin.H{
		"alerts": alerts,
		// Attribution travels with the data, not just in the UI, so any
		// consumer of this endpoint carries it too.
		"attribution": gin.H{
			"source": "Google Flood Hub",
			"url":    "https://sites.research.google/floods/",
			"disclaimer": "Third-party forecast. Not a CivicOS prediction and not an official warning. " +
				"Follow NEMA and NiMet for official guidance.",
		},
	})
}

// StartPoller runs the sweep on a ticker until ctx is cancelled.
//
// Mirrors the donation reconciliation sweep in organization-service: an
// interval of 0 disables it entirely. That is the kill switch — Google
// state the Flood Forecasting API is in pilot and that breaking changes
// should be expected, so an operator must be able to stop consuming it
// without a deploy.
func StartPoller(ctx context.Context, svc *Service, every time.Duration) {
	if every <= 0 {
		log.Printf("floodwatch: DISABLED by FLOOD_POLL_INTERVAL_MINUTES=0 — no flood forecasts will be fetched or shown")
		return
	}
	log.Printf("floodwatch: polling Google Flood Hub every %s (region %s, %.0fkm match radius)",
		every, svc.region, svc.radiusKm)

	run := func() {
		// Bounded per-run context so a hung upstream cannot stall the next
		// tick indefinitely.
		runCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()

		n, err := svc.Poll(runCtx)
		if err != nil {
			log.Printf("floodwatch: poll failed: %v", err)
			return
		}
		if err := svc.NotifyEscalations(); err != nil {
			log.Printf("floodwatch: notify failed: %v", err)
		}
		if n > 0 {
			log.Printf("floodwatch: %d community/gauge pairings alerting", n)
		}
	}

	go func() {
		// One sweep at startup so a fresh deploy is not blind until the
		// first tick — during a flood that gap is the whole point.
		run()
		t := time.NewTicker(every)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				run()
			}
		}
	}()
}
