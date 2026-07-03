package main

import (
	"fmt"
	"log"

	"github.com/civicos/identity-service/internal/auth"
	"github.com/civicos/identity-service/internal/middleware"
	"github.com/civicos/identity-service/pkg/config"
	"github.com/civicos/identity-service/pkg/database"
	"github.com/civicos/identity-service/pkg/mailer"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	db := database.Connect(cfg.DatabaseURL)

	var mail mailer.Mailer
	if cfg.SMTPHost != "" {
		mail = mailer.NewSMTPMailer(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPFrom)
		log.Printf("📨 Mailer: SMTP via %s:%d", cfg.SMTPHost, cfg.SMTPPort)
	} else {
		mail = mailer.NewConsoleMailer(cfg.SMTPFrom)
		log.Printf("📨 Mailer: console (set SMTP_HOST to send real mail)")
	}

	authRepo := auth.NewRepository(db)
	refreshRepo := auth.NewRefreshRepository(db)
	authSvc := auth.NewService(authRepo, refreshRepo, cfg, mail)
	authHandler := auth.NewHandler(authSvc)

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "identity-service"})
	})

	authGroup := r.Group("/v1/auth")
	authHandler.RegisterRoutes(authGroup, middleware.JWTAuth(cfg))

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("🚀 Identity Service running on %s", addr)

	if err := r.Run(addr); err != nil {
		log.Fatalf("❌ server failed: %v", err)
	}
}
