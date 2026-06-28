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

	// Auto-migrate — keeps schema in sync during development
	if err := db.AutoMigrate(&domain.User{}); err != nil {
		log.Fatalf("❌ failed to run migrations: %v", err)
	}

	log.Println("✅ Database connected")
	return db
}
