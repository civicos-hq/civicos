package main

import (
	"log"

	"github.com/civicos/api-gateway/internal/middleware"
	"github.com/civicos/api-gateway/internal/proxy"
	"github.com/civicos/api-gateway/pkg/config"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()

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

	// --- Identity Service ---
	// Public routes (register, login, refresh)
	identityPublic := proxy.NewReverseProxy(cfg.IdentityServiceURL, "/api")
	// Protected routes (/me)
	identityProtected := proxy.NewReverseProxy(cfg.IdentityServiceURL, "/api")

	r.POST("/api/v1/auth/register", identityPublic)
	r.POST("/api/v1/auth/login", identityPublic)
	r.POST("/api/v1/auth/refresh", identityPublic)
	r.POST("/api/v1/auth/verify-email", identityPublic)
	r.POST("/api/v1/auth/resend-verification", authMiddleware, identityProtected)
	r.GET("/api/v1/auth/me", authMiddleware, identityProtected)
	r.PATCH("/api/v1/auth/me", authMiddleware, identityProtected)
	r.POST("/api/v1/auth/me/community", authMiddleware, identityProtected)

	// --- Community Service ---
	communityProxy := proxy.NewReverseProxy(cfg.CommunityServiceURL, "/api")

	r.GET("/api/v1/communities", communityProxy)
	r.GET("/api/v1/communities/:id", communityProxy)
	r.POST("/api/v1/communities", authMiddleware, communityProxy)

	r.GET("/api/v1/issues", communityProxy)
	r.GET("/api/v1/issues/:id", communityProxy)
	r.POST("/api/v1/issues", authMiddleware, communityProxy)
	r.POST("/api/v1/issues/:id/upvote", authMiddleware, communityProxy)
	r.PATCH("/api/v1/issues/:id/status", authMiddleware, communityProxy)
	r.GET("/api/v1/issues/:id/comments", communityProxy)
	r.POST("/api/v1/issues/:id/comments", authMiddleware, communityProxy)

	// Petitions
	r.GET("/api/v1/petitions", communityProxy)
	r.GET("/api/v1/petitions/:id", communityProxy)
	r.POST("/api/v1/petitions", authMiddleware, communityProxy)
	r.POST("/api/v1/petitions/:id/sign", authMiddleware, communityProxy)
	r.GET("/api/v1/petitions/:id/comments", communityProxy)
	r.POST("/api/v1/petitions/:id/comments", authMiddleware, communityProxy)

	// Representatives
	r.GET("/api/v1/representatives", communityProxy)
	r.GET("/api/v1/representatives/:id", communityProxy)
	r.POST("/api/v1/representatives", authMiddleware, communityProxy)
	r.PATCH("/api/v1/representatives/:id", authMiddleware, communityProxy)
	r.POST("/api/v1/representatives/:id/follow", authMiddleware, communityProxy)
	r.DELETE("/api/v1/representatives/:id/follow", authMiddleware, communityProxy)
	r.GET("/api/v1/representatives/:id/comments", communityProxy)
	r.POST("/api/v1/representatives/:id/comments", authMiddleware, communityProxy)
	r.GET("/api/v1/me/follows/representatives", authMiddleware, communityProxy)

	// Uploads (POST is auth-protected; GET is public so images render in <img>)
	r.POST("/api/v1/uploads", authMiddleware, communityProxy)
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
