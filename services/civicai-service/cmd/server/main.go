package main

import (
	"context"
	"log"

	"github.com/civicos/civicai-service/internal/classify"
	"github.com/civicos/civicai-service/internal/gemini"
	"github.com/civicos/civicai-service/internal/middleware"
	"github.com/civicos/civicai-service/pkg/config"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
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

	addr := ":" + cfg.Port
	log.Printf("civicai-service listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
