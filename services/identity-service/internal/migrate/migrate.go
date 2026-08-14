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
// references.
//
// # Why at boot rather than a deploy hook
//
// A pre-deploy hook runs a shell command inside the service image, and
// these images are distroless: no shell, no libc, nothing to run a command
// with. Running in-process at startup needs none of that, behaves
// identically on a laptop and in production, and sits exactly where schema
// work already happens. A Postgres advisory lock serialises concurrent
// deploys.
package migrate

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrations embed.FS

// lockKey is a fixed identifier for the advisory lock that serialises
// migration runs. Two instances starting together must not both migrate: the
// second waits, finds the work done, and carries on.
const lockKey = 4_812_007

// prerequisites names, per migration version, the tables that migration
// operates on but does not create.
//
// identity-service owns the migrations for the whole shared database, so
// most of these tables belong to community-service or organization-service
// and come into existence through *their* AutoMigrate. Nothing orders the
// three services, so on a brand-new database identity can reach a migration
// whose subject does not exist yet and fail with a bare
//
//	relation "petition_signatures" does not exist
//
// which says nothing about what to do next.
//
// The fix is deliberately to FAIL rather than to skip. An `IF EXISTS` guard
// on each statement would let the migration record itself as applied while
// doing nothing, and goose would never revisit it — leaving the schema
// permanently short of the things AutoMigrate cannot express:
//
//   - 00005's foreign keys carry ON DELETE policies. GORM's implicit keys
//     carry none, so skipping means the wrong delete behaviour forever.
//   - 00006, 00007 and 00008 create partial and composite indexes that no
//     model declares, so nothing else would ever create them.
//
// Failing costs one restart and is self-healing: goose does not record a
// version it did not apply, so the next boot — after the other services
// have created their tables — succeeds.
//
// A migration absent from this map has no external prerequisite.
var prerequisites = map[int64][]string{
	2: {"petition_signatures"},
	3: {"users"},
	4: {
		"users", "refresh_tokens",
		"communities", "issues", "issue_comments", "issue_upvotes",
		"petitions", "petition_signatures", "petition_comments",
		"representatives", "representative_followers", "representative_comments",
		"notifications",
		"organizations", "org_members", "announcements", "projects",
		"issue_assignments", "progress_updates",
	},
	5: {
		"users", "refresh_tokens",
		"communities", "issues", "issue_comments", "issue_upvotes",
		"petitions", "petition_signatures", "petition_comments",
		"representatives", "representative_followers", "representative_comments",
		"notifications",
		"organizations", "org_members", "announcements", "projects",
		"issue_assignments", "progress_updates",
	},
	6: {"reconciliation_findings"},
	7: {"representatives"},
	8: {"organizations"},
	9: {"org_invitations"},
}

// tableOwner maps a table to the service whose AutoMigrate creates it, so
// the error can tell an operator which service to start rather than leaving
// them to work it out.
var tableOwner = map[string]string{
	"users": "identity-service", "refresh_tokens": "identity-service",

	"communities": "community-service", "issues": "community-service",
	"issue_comments": "community-service", "issue_upvotes": "community-service",
	"petitions": "community-service", "petition_signatures": "community-service",
	"petition_comments": "community-service", "representatives": "community-service",
	"representative_followers": "community-service",
	"representative_comments":  "community-service",
	"notifications":            "community-service",

	"organizations": "organization-service", "org_members": "organization-service",
	"announcements": "organization-service", "projects": "organization-service",
	"issue_assignments": "organization-service", "progress_updates": "organization-service",
	"reconciliation_findings": "organization-service",
	"org_invitations":         "organization-service",
}

// checkPrerequisites reports which tables the pending migrations need but
// the database does not have. Only pending versions are considered — an
// already-applied migration's prerequisites are irrelevant, and checking
// them would break a database whose tables have legitimately moved on.
func checkPrerequisites(ctx context.Context, db *sql.DB, currentVersion int64) error {
	needed := map[string]struct{}{}
	for version, tables := range prerequisites {
		if version <= currentVersion {
			continue
		}
		for _, t := range tables {
			needed[t] = struct{}{}
		}
	}
	if len(needed) == 0 {
		return nil
	}

	names := make([]string, 0, len(needed))
	for t := range needed {
		names = append(names, t)
	}
	sort.Strings(names)

	missingByOwner := map[string][]string{}
	for _, t := range names {
		var exists bool
		if err := db.QueryRowContext(ctx,
			"SELECT to_regclass($1) IS NOT NULL", "public."+t,
		).Scan(&exists); err != nil {
			return fmt.Errorf("migrate: check for table %s: %w", t, err)
		}
		if !exists {
			owner := tableOwner[t]
			if owner == "" {
				owner = "unknown service"
			}
			missingByOwner[owner] = append(missingByOwner[owner], t)
		}
	}
	if len(missingByOwner) == 0 {
		return nil
	}

	owners := make([]string, 0, len(missingByOwner))
	for o := range missingByOwner {
		owners = append(owners, o)
	}
	sort.Strings(owners)

	var b strings.Builder
	b.WriteString("migrate: cannot run pending migrations — tables they operate on do not exist yet:\n")
	for _, o := range owners {
		fmt.Fprintf(&b, "    %s: %s\n", o, strings.Join(missingByOwner[o], ", "))
	}
	b.WriteString("  Those tables are created by the owning service's AutoMigrate at ITS startup.\n")
	b.WriteString("  Start the services listed above, then start identity-service again.\n")
	b.WriteString("  Nothing was applied and no version was recorded, so this retries cleanly.")
	return errors.New(b.String())
}

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
	// Verified under the lock and after the version read, so the check sees
	// the same state the migrations will.
	if err := checkPrerequisites(ctx, db, before); err != nil {
		return err
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
