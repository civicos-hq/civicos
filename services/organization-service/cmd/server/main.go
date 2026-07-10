package main

import (
	"log"

	"github.com/civicos/organization-service/internal/announcements"
	"github.com/civicos/organization-service/internal/assignments"
	"github.com/civicos/organization-service/internal/audit"
	"github.com/civicos/organization-service/internal/communities"
	"github.com/civicos/organization-service/internal/consultations"
	"github.com/civicos/organization-service/internal/domain"
	"github.com/civicos/organization-service/internal/middleware"
	"github.com/civicos/organization-service/internal/notifications"
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
		&domain.Consultation{},
		&domain.ConsultationQuestion{},
		&domain.ConsultationResponse{},
		&domain.ConsultationAnswer{},
		&domain.ConsultationOutcome{},
	); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	// Shared audit writer.
	auditor := audit.New(db)

	// Organizations (registry + membership).
	orgRepo := organizations.NewRepository(db)
	orgSvc := organizations.NewService(orgRepo)
	orgHandler := organizations.NewHandler(orgSvc, auditor)

	// Shared notification writer — INSERTs directly into the community-
	// service-owned notifications table (same shared-DB pattern as audit).
	// Constructed early so downstream handlers (announcements, consultations)
	// can wire it in.
	notifier := notifications.NewDBNotifier(db)

	// Announcements — depend on orgSvc for member/admin checks and the
	// notifier for publish fan-out.
	annRepo := announcements.NewRepository(db)
	annSvc := announcements.NewService(annRepo, orgSvc)
	annHandler := announcements.NewHandler(annSvc, orgSvc, auditor, notifier)

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

	// Community-membership reader — same shared-DB pattern as audit +
	// notifications; identity-service owns the schema, we read from it.
	communityReader := communities.NewReader(db)

	// Consultations — structured feedback asks with a full lifecycle
	// (DRAFT → PUBLISHED → CLOSED) plus the "close the loop" outcome.
	// Fans notifications out to org members plus (when the consultation
	// is community-scoped) to the community's members too.
	consultRepo := consultations.NewRepository(db)
	consultSvc := consultations.NewService(consultRepo)
	consultHandler := consultations.NewHandler(consultSvc, orgSvc, auditor, notifier, communityReader)

	authMiddleware := middleware.JWTAuth(cfg, db)
	requireVerified := middleware.RequireVerified()
	// Roles allowed to create a brand-new organization. Anyone inside the
	// org can be promoted to ADMIN once it exists; this only gates who can
	// register a new one.
	requireOrgCreator := middleware.RequireRole("GOVERNMENT_ADMIN", "PLATFORM_ADMIN", "NGO")

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:5174"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "organization-service"})
	})

	v1 := r.Group("/v1")
	orgHandler.RegisterRoutes(v1.Group("/organizations"), authMiddleware, requireOrgCreator)
	orgHandler.RegisterMeRoutes(v1.Group("/me"), authMiddleware)

	// Announcements, projects, assignments, progress all mount on v1 because
	// their URL shapes span multiple resource roots (org, issue, project).
	annHandler.RegisterRoutes(v1, authMiddleware)
	projHandler.RegisterRoutes(v1, authMiddleware)
	asgHandler.RegisterRoutes(v1, authMiddleware)
	progHandler.RegisterRoutes(v1, authMiddleware)
	consultHandler.RegisterRoutes(v1, authMiddleware, requireVerified)

	addr := ":" + cfg.Port
	log.Printf("organization-service listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
