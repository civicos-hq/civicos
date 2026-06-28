package main

import (
	"fmt"
	"log"

	"github.com/civicos/identity-service/internal/auth"
	"github.com/civicos/identity-service/internal/middleware"
	"github.com/civicos/identity-service/pkg/config"
	"github.com/civicos/identity-service/pkg/database"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// Load and validate config at startup — fail fast if anything is missing
	cfg := config.Load()

	// Connect to database and run auto-migrations
	db := database.Connect(cfg.DatabaseURL)

	// Wire up dependencies (manual DI — no magic, no global state)
	authRepo := auth.NewRepository(db)
	authSvc := auth.NewService(authRepo, cfg)
	authHandler := auth.NewHandler(authSvc)

	// HTTP router
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "identity-service"})
	})

	// Auth routes — /v1/auth/... so the gateway can strip /api and forward correctly
	authGroup := r.Group("/v1/auth")
	authHandler.RegisterRoutes(authGroup, middleware.JWTAuth(cfg))

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("🚀 Identity Service running on %s", addr)

	if err := r.Run(addr); err != nil {
		log.Fatalf("❌ server failed: %v", err)
	}
}
