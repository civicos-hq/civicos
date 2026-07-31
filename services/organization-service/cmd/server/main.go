package main

import (
	"context"
	"log"
	"time"

	"github.com/civicos/organization-service/internal/announcements"
	"github.com/civicos/organization-service/internal/assignments"
	"github.com/civicos/organization-service/internal/audience"
	"github.com/civicos/organization-service/internal/audit"
	"github.com/civicos/organization-service/internal/campaigns"
	"github.com/civicos/organization-service/internal/communities"
	"github.com/civicos/organization-service/internal/consultations"
	"github.com/civicos/organization-service/internal/domain"
	"github.com/civicos/organization-service/internal/donations"
	"github.com/civicos/organization-service/internal/middleware"
	"github.com/civicos/organization-service/internal/milestones"
	"github.com/civicos/organization-service/internal/notifications"
	"github.com/civicos/organization-service/internal/organizations"
	"github.com/civicos/organization-service/internal/progress"
	"github.com/civicos/organization-service/internal/projects"
	"github.com/civicos/organization-service/internal/spend"
	"github.com/civicos/organization-service/pkg/config"
	"github.com/civicos/organization-service/pkg/database"
	"github.com/civicos/organization-service/pkg/mailer"
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
		&domain.Campaign{},
		&domain.Milestone{},
		&domain.Donation{},
		&domain.WebhookEvent{},
		&domain.SpendRecord{},
	); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	// Realtime notification bus. organization-service writes notification
	// rows itself, but the SSE hub that pushes them to browsers lives in
	// community-service — without this bridge they sit until the next fetch.
	// Optional by design: a missing broker costs realtime, not delivery.
	eventBus := notifications.ConnectNATS(cfg.NATSURL)
	if eventBus != nil {
		defer eventBus.Close()
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
	notifier := notifications.NewDBNotifier(db).WithBus(eventBus)

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

	// Community Funding — Phase 1: campaigns + their spend plan (milestones).
	// No donations, withdrawals or payment provider yet; see
	// docs/product/community-funding-plan.md for the phase order and why
	// transparency lands before money.
	// Who hears about a campaign event. One definition shared by campaigns,
	// milestones and donations — see internal/audience.
	aud := audience.New(db)

	campRepo := campaigns.NewRepository(db)
	campSvc := campaigns.NewService(campRepo).WithPlatformFee(cfg.PlatformFeeBps)
	campHandler := campaigns.NewHandler(campSvc, orgSvc, auditor).WithNotifications(notifier, aud)

	// Spend reporting (Phase 4). Because donations settle straight to the
	// organization, disclosure is the accountability lever that remains —
	// see internal/spend.
	spendRepo := spend.NewRepository(db)
	spendSvc := spend.NewService(spendRepo)
	spendHandler := spend.NewHandler(spendSvc, orgSvc, auditor)

	// The public campaign page shows reported spend alongside the ledger.
	// Wired after both exist; campaigns depends on the interface, not on
	// spend's storage.
	campSvc.WithSpend(spend.NewReader(spendSvc))

	msRepo := milestones.NewRepository(db)
	msSvc := milestones.NewService(msRepo)
	msHandler := milestones.NewHandler(msSvc, orgSvc).WithNotifications(notifier, aud, campSvc)

	// Donations (Phase 3). Paystack is OPTIONAL: without a key the provider
	// is nil, donation endpoints return 503, and everything else — campaigns,
	// admin review, the public pages — keeps working. A missing payment key
	// must degrade giving, not take the service down.
	var payProvider donations.PaymentProvider
	if cfg.PaystackEnabled() {
		payProvider = donations.NewPaystack(cfg.PaystackSecretKey)
		log.Printf("payments: paystack enabled, platform fee %d bps", cfg.PlatformFeeBps)
	} else {
		log.Printf("payments: DISABLED (no PAYSTACK_SECRET_KEY) — donation endpoints will return 503")
	}
	// Receipts. Mail is optional infrastructure: with no SMTP host the
	// console mailer prints the receipt to the log, so a dev environment
	// still exercises the path and a misconfigured relay never costs a
	// donation.
	var receiptMailer donations.ReceiptSender
	if cfg.SMTPHost != "" {
		receiptMailer = mailer.NewSMTPMailer(cfg.SMTPHost, int(cfg.SMTPPort), cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPFrom)
		log.Printf("receipts: smtp %s:%d", cfg.SMTPHost, cfg.SMTPPort)
	} else {
		receiptMailer = mailer.NewConsoleMailer(cfg.SMTPFrom)
		log.Printf("receipts: no SMTP_HOST — receipts will be printed to this log")
	}

	donRepo := donations.NewRepository(db)
	donSvc := donations.NewService(donRepo, payProvider, cfg.PlatformFeeBps, cfg.DonationCallbackURL).
		WithReceipts(receiptMailer, cfg.AppURL).
		WithNotifications(notifier, aud)
	donHandler := donations.NewHandler(donSvc, orgSvc, auditor)

	// Reconciliation. The webhook is the only thing that settles a donation,
	// which makes it a single point of failure — a delivery that never
	// arrives leaves money that genuinely moved sitting PENDING forever.
	// This sweep is how we find that out instead of a donor telling us.
	reconcileCtx, stopReconciler := context.WithCancel(context.Background())
	defer stopReconciler()
	donations.StartReconciler(reconcileCtx, donSvc,
		time.Duration(cfg.ReconcileIntervalMinutes)*time.Minute)

	authMiddleware := middleware.JWTAuth(cfg, db)
	requireVerified := middleware.RequireVerified()

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
	orgHandler.RegisterRoutes(v1.Group("/organizations"), authMiddleware)
	orgHandler.RegisterMeRoutes(v1.Group("/me"), authMiddleware)

	// Announcements, projects, assignments, progress all mount on v1 because
	// their URL shapes span multiple resource roots (org, issue, project).
	annHandler.RegisterRoutes(v1, authMiddleware)
	projHandler.RegisterRoutes(v1, authMiddleware)
	asgHandler.RegisterRoutes(v1, authMiddleware)
	progHandler.RegisterRoutes(v1, authMiddleware)
	consultHandler.RegisterRoutes(v1, authMiddleware, requireVerified)
	campHandler.RegisterRoutes(v1, authMiddleware)
	msHandler.RegisterRoutes(v1, authMiddleware)
	donHandler.RegisterRoutes(v1, authMiddleware)
	spendHandler.RegisterRoutes(v1, authMiddleware)

	addr := ":" + cfg.Port
	log.Printf("organization-service listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
