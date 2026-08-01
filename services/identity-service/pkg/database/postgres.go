package database

import (
	"context"
	"log"

	"github.com/civicos/identity-service/internal/domain"
	"github.com/civicos/identity-service/internal/migrate"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(dsn string) *gorm.DB {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("❌ failed to connect to database: %v", err)
	}

	// Versioned migrations FIRST, before AutoMigrate.
	//
	// Order matters: these carry constraints and type changes that AutoMigrate
	// would otherwise trip over, and a failed migration must stop the service
	// rather than let it serve against a half-changed schema.
	//
	// identity-service is the single owner of migrations for the shared
	// database — see internal/migrate for why.
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("❌ failed to reach the underlying database handle: %v", err)
	}
	if err := migrate.Run(context.Background(), sqlDB); err != nil {
		log.Fatalf("❌ %v", err)
	}

	// Auto-migrate — keeps schema in sync during development. Identity-service
	// is the source of truth for the moderation-infrastructure tables
	// (audit_logs, content_flags); other services may query/insert but
	// don't AutoMigrate them.
	if err := db.AutoMigrate(
		&domain.User{},
		&domain.UserCommunityMembership{},
		&domain.RefreshToken{},
		&domain.AuditLog{},
		&domain.ContentFlag{},
		&domain.ApplicationReviewEvent{},
		&domain.RepresentativeApplication{},
		&domain.OrganizationApplication{},
	); err != nil {
		log.Fatalf("❌ failed to run migrations: %v", err)
	}

	// The primary_community_id backfill that used to live here is now
	// migration 00003. It was a one-off data fix wearing the costume of
	// connection logic, and it re-ran on every boot forever with no record
	// that it had ever succeeded.

	log.Println("✅ Database connected")
	return db
}
