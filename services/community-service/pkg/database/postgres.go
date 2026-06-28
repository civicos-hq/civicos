package database

import (
	"log"

	"github.com/civicos/community-service/internal/domain"
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
	if err := db.AutoMigrate(
		&domain.Community{},
		&domain.Issue{},
		&domain.IssueComment{},
		&domain.Petition{},
		&domain.PetitionSignature{},
	); err != nil {
		log.Fatalf("❌ migration failed: %v", err)
	}
	log.Println("✅ Database connected")
	return db
}
