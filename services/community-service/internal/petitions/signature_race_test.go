package petitions

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/civicos/community-service/internal/domain"
	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestAddSignature_LosesTheInsertRace pins the fix for a bug that CI caught
// and a developer machine would not.
//
// The concurrent integration test in this package hits the interleaving
// only sometimes — it failed in CI under -race and passed locally three
// times running, including at 25 concurrent signers. A test that finds a
// bug one run in ten is barely a test, so this one does not race at all:
// it holds a conflicting row in an uncommitted transaction, releases it at
// exactly the wrong moment, and forces the loser's path every time.
//
// What used to happen: the losing INSERT violated idx_petition_user, the
// code caught that error and carried on, and Postgres — which aborts a
// transaction the moment any statement in it fails — rejected the next
// statement with "current transaction is aborted". The sign returned 500.
func TestAddSignature_LosesTheInsertRace(t *testing.T) {
	db, cleanup := newTestDB(t)
	defer cleanup()

	petitionID := uuid.New().String()
	userID := uuid.New().String()

	if err := db.Create(&domain.Petition{
		ID: petitionID, Title: "Fix the bridge", Description: "It is out",
		Goal: 100, CommunityID: uuid.New().String(), CreatedByID: userID,
		Status: domain.PetitionActive, CreatedAt: time.Now(),
	}).Error; err != nil {
		t.Fatalf("seed petition: %v", err)
	}

	repo := NewRepository(db)

	// The winner: a transaction that has inserted the signature but not yet
	// committed. Any other insert of the same (petition, user) now blocks
	// on the unique index until this resolves.
	winner := db.Begin()
	if err := winner.Create(&domain.PetitionSignature{
		ID: uuid.New().String(), PetitionID: petitionID, UserID: userID, CreatedAt: time.Now(),
	}).Error; err != nil {
		winner.Rollback()
		t.Fatalf("winner insert: %v", err)
	}

	// The loser, started while the winner still holds the row.
	type result struct {
		added    bool
		newCount int
		err      error
	}
	done := make(chan result, 1)
	go func() {
		added, newCount, err := repo.AddSignature(petitionID, userID)
		done <- result{added, newCount, err}
	}()

	// Give the loser time to reach its INSERT and block there. Without
	// this it may complete before the winner exists and never take the
	// conflicting path at all — which is exactly how this bug hid.
	time.Sleep(300 * time.Millisecond)

	// Release. The loser's insert now conflicts for certain.
	if err := winner.Commit().Error; err != nil {
		t.Fatalf("winner commit: %v", err)
	}

	select {
	case got := <-done:
		if got.err != nil {
			t.Fatalf("losing the insert race must not error — this is the bug: %v", got.err)
		}
		if got.added {
			t.Fatal("the loser must not report added=true, or milestone notifications fire twice")
		}
		// The winner's row is the only one, and it was inserted directly
		// without going through AddSignature, so signature_count is still
		// 0 — the point here is that the read SUCCEEDED rather than dying
		// on an aborted transaction.
		if got.newCount < 0 {
			t.Fatalf("unexpected count: %d", got.newCount)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("AddSignature never returned — it is still blocked on the unique index")
	}

	// Exactly one signature exists.
	var signatures int64
	if err := db.Model(&domain.PetitionSignature{}).
		Where("petition_id = ? AND user_id = ?", petitionID, userID).
		Count(&signatures).Error; err != nil {
		t.Fatalf("count signatures: %v", err)
	}
	if signatures != 1 {
		t.Fatalf("expected exactly 1 signature, got %d", signatures)
	}
}

// A plain repeat sign, with nothing racing. Cheap, and it pins the
// idempotency the caller relies on to avoid duplicate notifications.
func TestAddSignature_RepeatSignIsIdempotent(t *testing.T) {
	db, cleanup := newTestDB(t)
	defer cleanup()

	petitionID := uuid.New().String()
	userID := uuid.New().String()
	if err := db.Create(&domain.Petition{
		ID: petitionID, Title: "Fix the bridge", Description: "It is out",
		Goal: 100, CommunityID: uuid.New().String(), CreatedByID: userID,
		Status: domain.PetitionActive, CreatedAt: time.Now(),
	}).Error; err != nil {
		t.Fatalf("seed petition: %v", err)
	}
	repo := NewRepository(db)

	added, count, err := repo.AddSignature(petitionID, userID)
	if err != nil || !added || count != 1 {
		t.Fatalf("first sign: added=%v count=%d err=%v", added, count, err)
	}

	added, count, err = repo.AddSignature(petitionID, userID)
	if err != nil {
		t.Fatalf("second sign errored: %v", err)
	}
	if added {
		t.Fatal("second sign must report added=false")
	}
	if count != 1 {
		t.Fatalf("count must not double-increment, got %d", count)
	}
}

// newTestDB starts an embedded Postgres and migrates just the tables these
// tests touch. Separate from the service-level integration test in this
// package: that one builds and boots the whole binary, which is far more
// than is needed to exercise one repository method.
func newTestDB(t *testing.T) (*gorm.DB, func()) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("pick port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()

	// Private runtime path: embedded-postgres otherwise shares one
	// extraction directory and wipes it at the start of every Start(), so
	// two packages running in parallel delete each other's binaries. Short
	// base name because the unix socket lives under it and the path has a
	// ~107 character limit.
	runtimeDir, err := os.MkdirTemp("", "cvsig")
	if err != nil {
		t.Fatalf("temp runtime dir: %v", err)
	}

	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().Port(uint32(port)).RuntimePath(runtimeDir),
	)
	if err := pg.Start(); err != nil {
		os.RemoveAll(runtimeDir)
		t.Fatalf("start embedded postgres: %v", err)
	}

	dsn := fmt.Sprintf("postgres://postgres:postgres@localhost:%d/postgres?sslmode=disable", port)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for {
		raw, oerr := sql.Open("pgx", dsn)
		if oerr == nil {
			perr := raw.Ping()
			raw.Close()
			if perr == nil {
				break
			}
		}
		select {
		case <-ctx.Done():
			_ = pg.Stop()
			os.RemoveAll(runtimeDir)
			t.Fatal("timed out waiting for postgres")
		case <-time.After(250 * time.Millisecond):
		}
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		_ = pg.Stop()
		os.RemoveAll(runtimeDir)
		t.Fatalf("open gorm: %v", err)
	}
	// AutoMigrate creates idx_petition_user from the model's uniqueIndex
	// tag, which is the constraint the whole test turns on.
	if err := db.AutoMigrate(&domain.Petition{}, &domain.PetitionSignature{}); err != nil {
		_ = pg.Stop()
		os.RemoveAll(runtimeDir)
		t.Fatalf("automigrate: %v", err)
	}

	return db, func() {
		_ = pg.Stop()
		os.RemoveAll(runtimeDir)
	}
}
