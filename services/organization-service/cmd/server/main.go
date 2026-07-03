package main

import (
	"log"

	"github.com/civicos/organization-service/internal/announcements"
	"github.com/civicos/organization-service/internal/assignments"
	"github.com/civicos/organization-service/internal/domain"
	"github.com/civicos/organization-service/internal/middleware"
	"github.com/civicos/organization-service/internal/organizations"
	"github.com/civicos/organization-service/internal/progress"
	"github.com/civicos/organization-service/internal/projects"
	"github.com/civicos/organization-service/pkg/config"
	"github.com/civicos/organization-service/pkg/database"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	db := database.Connect(cfg.DatabaseURL)

	if err := db.AutoMigrate(
		&domain.Organization{},
		&domain.OrgMember{},
		&domain.Announcement{},
		&domain.Project{},
		&domain.IssueAssignment{},
		&domain.ProgressUpdate{},
	); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	// Organizations (registry + membership).
	orgRepo := organizations.NewRepository(db)
	orgSvc := organizations.NewService(orgRepo)
	orgHandler := organizations.NewHandler(orgSvc)

	// Announcements — depend on orgSvc for member/admin checks.
	annRepo := announcements.NewRepository(db)
	annSvc := announcements.NewService(annRepo, orgSvc)
	annHandler := announcements.NewHandler(annSvc, orgSvc)

	// Projects.
	projRepo := projects.NewRepository(db)
	projSvc := projects.NewService(projRepo, orgSvc)
	projHandler := projects.NewHandler(projSvc, orgSvc)

	// Issue assignments — the "receive reports" capability.
	asgRepo := assignments.NewRepository(db)
	asgSvc := assignments.NewService(asgRepo, orgSvc)
	asgHandler := assignments.NewHandler(asgSvc, orgSvc)

	// Progress updates — the "respond publicly" + "update progress" capability.
	progRepo := progress.NewRepository(db)
	progSvc := progress.NewService(progRepo, orgSvc)
	progHandler := progress.NewHandler(progSvc, orgSvc)

	authMiddleware := middleware.JWTAuth(cfg)
	// Roles allowed to create a brand-new organization. Anyone inside the
	// org can be promoted to ADMIN once it exists; this only gates who can
	// register a new one.
	requireOrgCreator := middleware.RequireRole("GOVERNMENT_ADMIN", "PLATFORM_ADMIN", "NGO")

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "organization-service"})
	})

	v1 := r.Group("/v1")
	orgHandler.RegisterRoutes(v1.Group("/organizations"), authMiddleware, requireOrgCreator)

	// Announcements, projects, assignments, progress all mount on v1 because
	// their URL shapes span multiple resource roots (org, issue, project).
	annHandler.RegisterRoutes(v1, authMiddleware)
	projHandler.RegisterRoutes(v1, authMiddleware)
	asgHandler.RegisterRoutes(v1, authMiddleware)
	progHandler.RegisterRoutes(v1, authMiddleware)

	addr := ":" + cfg.Port
	log.Printf("organization-service listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
