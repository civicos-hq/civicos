package main

import (
    "log"
    "os"

    "github.com/gin-gonic/gin"
)

func main() {
    port := os.Getenv("PORT")
    if port == "" {
        port = "3002"
    }

    r := gin.Default()

    r.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "ok", "service": "community-service"})
    })

    log.Printf("starting community-service on :%s\n", port)
    if err := r.Run(":" + port); err != nil {
        log.Fatal(err)
    }
}
package main

import (
	"log"

	"github.com/civicos/community-service/internal/communities"
	"github.com/civicos/community-service/internal/domain"
	"github.com/civicos/community-service/internal/issues"
	"github.com/civicos/community-service/internal/middleware"
	"github.com/civicos/community-service/pkg/config"
	"github.com/civicos/community-service/pkg/database"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// config.Load() handles godotenv internally
	cfg := config.Load()

	db := database.Connect(cfg.DatabaseURL)

	// Run migrations explicitly here so it's clear what this service owns
	if err := db.AutoMigrate(
		&domain.Community{},
		&domain.Issue{},
		&domain.IssueComment{},
		&domain.Petition{},
		&domain.PetitionSignature{},
	); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	// Dependency injection
	communityRepo := communities.NewRepository(db)
	communitySvc := communities.NewService(communityRepo)
	communityHandler := communities.NewHandler(communitySvc)

	issueRepo := issues.NewRepository(db)
	issueSvc := issues.NewService(issueRepo)
	issueHandler := issues.NewHandler(issueSvc)

	authMiddleware := middleware.JWTAuth(cfg)

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "community-service"})
	})

	v1 := r.Group("/v1")
	communityHandler.RegisterRoutes(v1.Group("/communities"), authMiddleware)
	issueHandler.RegisterRoutes(v1.Group("/issues"), authMiddleware)

	addr := ":" + cfg.Port
	log.Printf("community-service listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
