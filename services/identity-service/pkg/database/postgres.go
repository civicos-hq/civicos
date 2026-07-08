package database

import (
	"log"

	"github.com/civicos/identity-service/internal/domain"
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

	// Backfill primary_community_id for users who existed before the field
	// was introduced. Their active community becomes their primary — the
	// safest guess and the one that keeps current behaviour unchanged.
	if err := db.Exec(
		`UPDATE users
		 SET primary_community_id = community_id
		 WHERE primary_community_id IS NULL AND community_id IS NOT NULL`,
	).Error; err != nil {
		log.Fatalf("❌ failed to backfill primary_community_id: %v", err)
	}

	log.Println("✅ Database connected")
	return db
}
