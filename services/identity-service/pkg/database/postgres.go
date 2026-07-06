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
		&domain.RefreshToken{},
		&domain.AuditLog{},
		&domain.ContentFlag{},
		&domain.RepresentativeApplication{},
		&domain.OrganizationApplication{},
	); err != nil {
		log.Fatalf("❌ failed to run migrations: %v", err)
	}

	log.Println("✅ Database connected")
	return db
}
