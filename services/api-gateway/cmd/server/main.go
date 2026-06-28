package main

import (
	"log"

	"github.com/civicos/api-gateway/internal/middleware"
	"github.com/civicos/api-gateway/internal/proxy"
	"github.com/civicos/api-gateway/pkg/config"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

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
	r.GET("/api/v1/auth/me", authMiddleware, identityProtected)

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

	addr := ":" + cfg.Port
	log.Printf("api-gateway listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
