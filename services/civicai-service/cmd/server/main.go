package main

import (
	"context"
	"log"
	"time"

	"github.com/civicos/civicai-service/internal/campaignai"
	"github.com/civicos/civicai-service/internal/classify"
	"github.com/civicos/civicai-service/internal/draft"
	"github.com/civicos/civicai-service/internal/gemini"
	"github.com/civicos/civicai-service/internal/insights"
	"github.com/civicos/civicai-service/internal/middleware"
	"github.com/civicos/civicai-service/internal/narrate"
	"github.com/civicos/civicai-service/internal/summarize"
	"github.com/civicos/civicai-service/pkg/config"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := config.Load()

	// Boot-time init: catch bad API keys immediately, not on the first
	// citizen click. The gateway won't route to us until we're healthy.
	aiClient, err := gemini.New(context.Background(), cfg.GeminiAPIKey, cfg.GeminiModel)
	if err != nil {
		log.Fatalf("gemini init failed: %v", err)
	}
	log.Printf("gemini client ready — model=%s", cfg.GeminiModel)

	classifySvc := classify.NewService(aiClient)
	classifyHandler := classify.NewHandler(classifySvc)

	// Summarize: pulls source data from community-service using the caller's
	// forwarded JWT, then caches the result in Redis. Cache is optional —
	// if REDIS_URL is unset, summaries just aren't cached (dev-only).
	var rdb *redis.Client
	if cfg.RedisURL != "" {
		opts, err := redis.ParseURL(cfg.RedisURL)
		if err != nil {
			log.Printf("civicai: bad REDIS_URL, summary cache disabled: %v", err)
		} else {
			rdb = redis.NewClient(opts)
			bootCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			if err := rdb.Ping(bootCtx).Err(); err != nil {
				log.Printf("civicai: redis unreachable, summary cache disabled: %v", err)
				rdb = nil
			} else {
				log.Printf("civicai: summary cache active via %s", cfg.RedisURL)
			}
			cancel()
		}
	}
	summarizeSource := summarize.NewSourceClient(cfg.CommunityServiceURL, cfg.OrganizationServiceURL)
	summarizeCache := summarize.NewCache(rdb, 30*time.Minute)
	summarizeSvc := summarize.NewService(aiClient, summarizeSource, summarizeCache)
	summarizeHandler := summarize.NewHandler(summarizeSvc)

	// Draft: creative one-shot, no external data pulls, no cache
	// (announcement drafts are personal to a brief — a second call with
	// the same brief should give the admin a fresh variation to compare).
	draftSvc := draft.NewService(aiClient)
	draftHandler := draft.NewHandler(draftSvc)

	// Insights: community-scoped aggregate digest. Shares the same
	// community-service source pattern as summarize, but fans out across
	// many resources. Cached in Redis (1h TTL) — cache-miss cost is
	// dominated by the fan-out reads, not just Gemini.
	insightsSource := insights.NewSourceClient(cfg.CommunityServiceURL)
	insightsSvc := insights.NewService(aiClient, insightsSource, rdb)
	insightsHandler := insights.NewHandler(insightsSvc)

	// Analytics Narrator: platform-scoped digest of admin metrics. Uses the
	// same Redis handle for a short (15min) cache — the numbers change
	// slowly enough that operators re-clicking within that window get an
	// instant response.
	narrateSvc := narrate.NewService(aiClient, cfg.IdentityServiceURL, rdb)
	narrateHandler := narrate.NewHandler(narrateSvc)

	// Community Funding surfaces. All six are advisory and none writes
	// anything: they read a campaign through organization-service with the
	// caller's own token, so authorization cascades rather than being
	// re-implemented here.
	campaignAISvc := campaignai.NewService(aiClient, campaignai.NewSourceClient(cfg.OrganizationServiceURL), campaignai.NewCache(rdb))
	campaignAIHandler := campaignai.NewHandler(campaignAISvc)

	authMiddleware := middleware.JWTAuth(cfg)

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:5174"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "civicai-service", "model": cfg.GeminiModel})
	})

	v1 := r.Group("/v1")
	ai := v1.Group("/ai", authMiddleware)
	classifyHandler.RegisterRoutes(ai)
	summarizeHandler.RegisterRoutes(ai)
	draftHandler.RegisterRoutes(ai)
	insightsHandler.RegisterRoutes(ai)
	narrateHandler.RegisterRoutes(ai)
	campaignAIHandler.RegisterRoutes(ai)

	addr := ":" + cfg.Port
	log.Printf("civicai-service listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
