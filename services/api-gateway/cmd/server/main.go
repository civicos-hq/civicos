package main

import (
	"context"
	"log"
	"time"

	"github.com/civicos/api-gateway/internal/middleware"
	"github.com/civicos/api-gateway/internal/proxy"
	"github.com/civicos/api-gateway/pkg/config"
	"github.com/civicos/api-gateway/pkg/ratelimit"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()

	// Rate-limit dependencies. Failing to reach Redis at boot is not fatal
	// — the limiter falls back to no-op — but we log it loudly so an
	// operator can see the gap in defence.
	var limiter *ratelimit.Limiter
	if cfg.RedisURL != "" {
		bootCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		rdb, err := ratelimit.Connect(bootCtx, cfg.RedisURL)
		cancel()
		if err != nil {
			log.Printf("⚠️  rate limiter disabled — could not connect to Redis: %v", err)
		} else {
			limiter = ratelimit.New(rdb)
			log.Printf("🛡️  rate limiter active via %s", cfg.RedisURL)
		}
	} else {
		log.Printf("⚠️  rate limiter disabled — REDIS_URL not set")
	}

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	// Add CORS middleware that sets headers on all responses
	r.Use(func(c *gin.Context) {
		// Allow frontend origins
		origin := c.Request.Header.Get("Origin")
		if origin == "http://localhost:5173" || origin == "http://localhost:5174" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		} else if origin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "false")
		}

		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "api-gateway"})
	})

	authMiddleware := middleware.JWTAuth(cfg)

	// Rate-limit tiers — see internal/middleware/ratelimit.go for the budgets.
	// Convention: apply the limit BEFORE the auth middleware for public routes
	// (key by IP), AFTER auth for protected routes (key by userID).
	limitStrict := middleware.Limit(limiter, middleware.Strict)
	limitStandard := middleware.Limit(limiter, middleware.Standard)
	limitLenient := middleware.Limit(limiter, middleware.Lenient)

	// --- Identity Service ---
	identityPublic := proxy.NewReverseProxy(cfg.IdentityServiceURL, "/api")
	identityProtected := proxy.NewReverseProxy(cfg.IdentityServiceURL, "/api")

	// Auth endpoints are the most-attacked surface — Strict is deliberately
	// low. Refresh/logout use Lenient because the SPA hits refresh on every
	// 401 during a burst of concurrent 401s (interceptor coalesces, but
	// timing across tabs isn't guaranteed).
	r.POST("/api/v1/auth/register", limitStrict, identityPublic)
	r.POST("/api/v1/auth/login", limitStrict, identityPublic)
	r.POST("/api/v1/auth/refresh", limitLenient, identityPublic)
	r.POST("/api/v1/auth/logout", limitLenient, identityPublic)
	r.POST("/api/v1/auth/verify-email", limitStandard, identityPublic)
	r.POST("/api/v1/auth/resend-verification", authMiddleware, limitStrict, identityProtected)
	r.POST("/api/v1/auth/forgot-password", limitStrict, identityPublic)
	r.POST("/api/v1/auth/reset-password", limitStandard, identityPublic)
	r.GET("/api/v1/auth/me", authMiddleware, identityProtected)
	r.PATCH("/api/v1/auth/me", authMiddleware, limitStandard, identityProtected)
	r.POST("/api/v1/auth/me/community", authMiddleware, limitStandard, identityProtected)

	// --- Community Service ---
	communityProxy := proxy.NewReverseProxy(cfg.CommunityServiceURL, "/api")

	r.GET("/api/v1/communities", communityProxy)
	r.GET("/api/v1/communities/:id", communityProxy)
	r.POST("/api/v1/communities", authMiddleware, limitStandard, communityProxy)

	r.GET("/api/v1/issues", communityProxy)
	r.GET("/api/v1/issues/:id", communityProxy)
	r.POST("/api/v1/issues", authMiddleware, limitStandard, communityProxy)
	r.POST("/api/v1/issues/:id/upvote", authMiddleware, limitStandard, communityProxy)
	r.PATCH("/api/v1/issues/:id/status", authMiddleware, limitStandard, communityProxy)
	r.GET("/api/v1/issues/:id/comments", communityProxy)
	r.POST("/api/v1/issues/:id/comments", authMiddleware, limitStandard, communityProxy)

	// Petitions
	r.GET("/api/v1/petitions", communityProxy)
	r.GET("/api/v1/petitions/:id", communityProxy)
	r.POST("/api/v1/petitions", authMiddleware, limitStandard, communityProxy)
	r.POST("/api/v1/petitions/:id/sign", authMiddleware, limitStandard, communityProxy)
	r.GET("/api/v1/petitions/:id/comments", communityProxy)
	r.POST("/api/v1/petitions/:id/comments", authMiddleware, limitStandard, communityProxy)

	// Representatives
	r.GET("/api/v1/representatives", communityProxy)
	r.GET("/api/v1/representatives/:id", communityProxy)
	r.POST("/api/v1/representatives", authMiddleware, limitStandard, communityProxy)
	r.PATCH("/api/v1/representatives/:id", authMiddleware, limitStandard, communityProxy)
	r.POST("/api/v1/representatives/:id/follow", authMiddleware, limitStandard, communityProxy)
	r.DELETE("/api/v1/representatives/:id/follow", authMiddleware, limitStandard, communityProxy)
	r.GET("/api/v1/representatives/:id/comments", communityProxy)
	r.POST("/api/v1/representatives/:id/comments", authMiddleware, limitStandard, communityProxy)
	r.GET("/api/v1/me/follows/representatives", authMiddleware, communityProxy)
	r.GET("/api/v1/me/upvotes/issues", authMiddleware, communityProxy)

	// Uploads (POST is auth-protected; GET is public so images render in <img>)
	r.POST("/api/v1/uploads", authMiddleware, limitStandard, communityProxy)
	r.GET("/api/v1/uploads/:filename", communityProxy)

	// Search
	r.GET("/api/v1/search", communityProxy)

	// Discover
	r.GET("/api/v1/discover/feed", authMiddleware, communityProxy)

	// Notifications
	notificationsStream := proxy.NewStreamingProxy(cfg.CommunityServiceURL, "/api")
	r.GET("/api/v1/notifications", authMiddleware, communityProxy)
	r.GET("/api/v1/notifications/unread-count", authMiddleware, communityProxy)
	r.GET("/api/v1/notifications/stream", authMiddleware, notificationsStream)
	r.PATCH("/api/v1/notifications/:id/read", authMiddleware, communityProxy)
	r.POST("/api/v1/notifications/read-all", authMiddleware, communityProxy)

	// Handle unmatched OPTIONS requests with CORS headers
	r.NoRoute(func(c *gin.Context) {
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.JSON(404, gin.H{"error": "not found"})
	})

	addr := ":" + cfg.Port
	log.Printf("api-gateway listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
