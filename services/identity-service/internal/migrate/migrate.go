// Package migrate applies the shared database migrations.
//
// # Why this exists
//
// Every service calls GORM's AutoMigrate at boot. AutoMigrate creates tables
// and ADDS columns, and that is all — it never drops a column, narrows a
// type, adds a constraint, backfills data, or records that anything happened.
// Everything outside that narrow band was done by hand, and it showed:
//
//   - Three .sql files sat in the repo that NOTHING executed, including
//     foreign keys and a uuid normalisation. Whether they had reached
//     production was unknowable from the repo alone.
//   - identity-service ran a backfill UPDATE on every single boot, inside its
//     connection code, because there was nowhere else to put it.
//
// This owns the changes AutoMigrate cannot express, and records them.
//
// # Why identity-service owns it
//
// All services share one database, so migrations need exactly one owner —
// three services racing to migrate at startup would contend over a single
// version table. identity-service owns `users`, which everything else
// references, and it runs on a plan that never sleeps.
//
// # Why at boot rather than a deploy hook
//
// Render's preDeployCommand runs a shell command inside the service image,
// and these images are distroless: no shell, no libc, nothing to run a
// command with. Running in-process at startup needs none of that, behaves
// identically on a laptop and in production, and sits exactly where schema
// work already happens. A Postgres advisory lock serialises concurrent
// deploys.
package migrate

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log"
	"time"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrations embed.FS

// lockKey is a fixed identifier for the advisory lock that serialises
// migration runs. Two instances starting together must not both migrate: the
// second waits, finds the work done, and carries on.
const lockKey = 4_812_007

// Run applies every pending migration. It blocks until they are done or the
// context expires.
//
// Returns an error rather than exiting, so the caller decides whether a
// failed migration should stop the service — it should, and does.
func Run(ctx context.Context, db *sql.DB) error {
	goose.SetBaseFS(migrations)
	goose.SetTableName("schema_migrations")
	// goose logs each applied migration; silence the banner it prints when
	// there is nothing to do, which is the common case on restart.
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// A dedicated connection: the advisory lock lives on the session, so it
	// must not be handed back to the pool mid-run.
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("migrate: acquire connection: %w", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", lockKey); err != nil {
		return fmt.Errorf("migrate: take lock: %w", err)
	}
	defer func() {
		// Best effort on a fresh context: the run's context may already be
		// cancelled, and the lock is released with the session regardless.
		_, _ = conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", lockKey)
	}()

	before, err := goose.GetDBVersion(db)
	if err != nil {
		return fmt.Errorf("migrate: read version: %w", err)
	}
	if err := goose.UpContext(ctx, db, "migrations"); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	after, err := goose.GetDBVersion(db)
	if err != nil {
		return fmt.Errorf("migrate: read version: %w", err)
	}

	if after > before {
		log.Printf("✅ migrations applied: %d → %d", before, after)
	} else {
		log.Printf("✅ migrations up to date (version %d)", after)
	}
	return nil
}
