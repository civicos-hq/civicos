package database

import (
	"context"
	"database/sql"
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

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("❌ failed to reach the underlying database handle: %v", err)
	}

	// Ordering between the versioned migrations and AutoMigrate depends on
	// whether this database already has a schema.
	//
	// On an EXISTING database the migrations must go first: they carry
	// constraints and type changes that AutoMigrate would otherwise trip
	// over — 00004 narrows FK columns to uuid, and AutoMigrate meeting a
	// legacy text column before that runs is exactly the collision the
	// original ordering was written to avoid.
	//
	// On a BRAND-NEW database that ordering cannot work. Migrations 3, 4
	// and 5 operate on `users`, which nothing has created yet — and it is
	// identity-service's own table, so waiting for another service would
	// never help. Running them first meant identity could not boot against
	// an empty database at all, and there is nothing for AutoMigrate to
	// trip over on a database with no columns in it.
	//
	// So: create the tables first when there is nothing there, and migrate
	// first when there is.
	fresh := !tableExists(context.Background(), sqlDB, "users")

	autoMigrate := func() {
		// identity-service is the source of truth for the
		// moderation-infrastructure tables (audit_logs, content_flags);
		// other services may query/insert but don't AutoMigrate them.
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
	}

	if fresh {
		log.Println("🆕 empty database — creating tables before applying migrations")
		autoMigrate()
	}

	// identity-service is the single owner of migrations for the shared
	// database — see internal/migrate for why. A failed migration stops the
	// service rather than letting it serve against a half-changed schema.
	if err := migrate.Run(context.Background(), sqlDB); err != nil {
		log.Fatalf("❌ %v", err)
	}

	if !fresh {
		autoMigrate()
	}

	// The primary_community_id backfill that used to live here is now
	// migration 00003. It was a one-off data fix wearing the costume of
	// connection logic, and it re-ran on every boot forever with no record
	// that it had ever succeeded.

	log.Println("✅ Database connected")
	return db
}

// tableExists reports whether a table is present in the public schema.
// Used only to tell a brand-new database from an established one; a query
// error is treated as "not fresh" so an unexpected failure falls back to
// the long-standing migrate-then-AutoMigrate order rather than silently
// changing behaviour.
func tableExists(ctx context.Context, db *sql.DB, name string) bool {
	var exists bool
	if err := db.QueryRowContext(ctx,
		"SELECT to_regclass($1) IS NOT NULL", "public."+name,
	).Scan(&exists); err != nil {
		return false
	}
	return exists
}
