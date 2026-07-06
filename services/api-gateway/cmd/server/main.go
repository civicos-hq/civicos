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
		if origin == "http://localhost:5173" || origin == "http://localhost:5174" || origin == "http://localhost:5175" {
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

	// Server-side health probes for the admin console — the browser can't
	// reach the internal services, so the gateway checks on its behalf.
	r.GET("/health/identity", proxy.NewHealthProxy(cfg.IdentityServiceURL))
	r.GET("/health/community", proxy.NewHealthProxy(cfg.CommunityServiceURL))
	r.GET("/health/organization", proxy.NewHealthProxy(cfg.OrganizationServiceURL))

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
	// Citizen-initiated account deletion — soft-deletes the user (PII
	// anonymized, refresh tokens revoked). Strict-tier because a runaway
	// script that hammers DELETE would nuke accounts en masse.
	r.DELETE("/api/v1/auth/me", authMiddleware, limitStrict, identityProtected)
	r.POST("/api/v1/auth/me/community", authMiddleware, limitStandard, identityProtected)

	// --- Moderation infrastructure (identity-service owns these) ---
	// Content flags — Strict tier on the POST to prevent flag spam.
	r.POST("/api/v1/flags", authMiddleware, limitStrict, identityProtected)
	r.GET("/api/v1/flags", authMiddleware, identityProtected)
	r.GET("/api/v1/flags/counts", authMiddleware, identityProtected)
	r.GET("/api/v1/flags/:id", authMiddleware, identityProtected)
	r.PATCH("/api/v1/flags/:id", authMiddleware, limitStandard, identityProtected)
	// Admin's proactive-moderation shortcut — flag + hide in one call.
	// Distinct from POST /flags (which requires verified + rate-limits at
	// Strict); this one's admin-only and standard-tier since abuse-by-
	// admin is a different threat model.
	r.POST("/api/v1/flags/direct-hide", authMiddleware, limitStandard, identityProtected)
	// Audit log — admin-only read surface.
	r.GET("/api/v1/audit-logs", authMiddleware, identityProtected)

	// Platform-wide metrics + per-community stats — admin-only reads.
	r.GET("/api/v1/admin/metrics", authMiddleware, identityProtected)
	r.GET("/api/v1/admin/communities/:id/stats", authMiddleware, identityProtected)

	// User administration — admin-only. Ban/unban and role change all
	// write to the audit log inside the identity-service handlers.
	r.GET("/api/v1/users", authMiddleware, identityProtected)
	r.GET("/api/v1/users/:id", authMiddleware, identityProtected)
	r.PATCH("/api/v1/users/:id/role", authMiddleware, limitStandard, identityProtected)
	r.POST("/api/v1/users/:id/ban", authMiddleware, limitStandard, identityProtected)
	r.POST("/api/v1/users/:id/unban", authMiddleware, limitStandard, identityProtected)

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

	// --- Organization Service ---
	orgProxy := proxy.NewReverseProxy(cfg.OrganizationServiceURL, "/api")

	// Organizations (registry + membership)
	r.GET("/api/v1/organizations", orgProxy)
	r.GET("/api/v1/organizations/:id", orgProxy)
	r.POST("/api/v1/organizations", authMiddleware, limitStandard, orgProxy)
	r.PATCH("/api/v1/organizations/:id", authMiddleware, limitStandard, orgProxy)
	r.GET("/api/v1/organizations/:id/members", orgProxy)
	r.POST("/api/v1/organizations/:id/members", authMiddleware, limitStandard, orgProxy)
	r.PATCH("/api/v1/organizations/:id/members/:userId", authMiddleware, limitStandard, orgProxy)
	r.DELETE("/api/v1/organizations/:id/members/:userId", authMiddleware, limitStandard, orgProxy)

	// Announcements
	r.GET("/api/v1/announcements", orgProxy)
	r.GET("/api/v1/organizations/:id/announcements", authMiddleware, orgProxy)
	r.GET("/api/v1/announcements/:announcementId", orgProxy)
	r.POST("/api/v1/organizations/:id/announcements", authMiddleware, limitStandard, orgProxy)
	r.PATCH("/api/v1/announcements/:announcementId", authMiddleware, limitStandard, orgProxy)
	r.POST("/api/v1/announcements/:announcementId/publish", authMiddleware, limitStandard, orgProxy)
	r.POST("/api/v1/announcements/:announcementId/archive", authMiddleware, limitStandard, orgProxy)
	r.DELETE("/api/v1/announcements/:announcementId", authMiddleware, limitStandard, orgProxy)

	// Projects
	r.GET("/api/v1/projects", orgProxy)
	r.GET("/api/v1/organizations/:id/projects", orgProxy)
	r.GET("/api/v1/projects/:projectId", orgProxy)
	r.POST("/api/v1/organizations/:id/projects", authMiddleware, limitStandard, orgProxy)
	r.PATCH("/api/v1/projects/:projectId", authMiddleware, limitStandard, orgProxy)
	r.DELETE("/api/v1/projects/:projectId", authMiddleware, limitStandard, orgProxy)

	// Issue assignments (receive reports)
	r.GET("/api/v1/organizations/:id/assignments", authMiddleware, orgProxy)
	r.GET("/api/v1/issues/:id/assignments", orgProxy)
	r.POST("/api/v1/organizations/:id/assignments", authMiddleware, limitStandard, orgProxy)
	r.PATCH("/api/v1/assignments/:assignmentId", authMiddleware, limitStandard, orgProxy)
	r.DELETE("/api/v1/assignments/:assignmentId", authMiddleware, limitStandard, orgProxy)

	// Progress updates (public responses on issues, project progress)
	r.GET("/api/v1/issues/:id/progress-updates", orgProxy)
	r.GET("/api/v1/projects/:projectId/progress-updates", orgProxy)
	r.POST("/api/v1/organizations/:id/progress-updates", authMiddleware, limitStandard, orgProxy)
	r.DELETE("/api/v1/progress-updates/:updateId", authMiddleware, limitStandard, orgProxy)

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
