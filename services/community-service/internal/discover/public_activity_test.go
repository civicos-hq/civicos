package discover

import (
	"database/sql"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	_ "github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// PublicActivity is the only unauthenticated aggregate on CivicOS: anyone on
// the internet can call it without an account. The failure that matters is
// therefore not "the ticker looks wrong" but "a draft leaked", so these tests
// run against a real Postgres and seed every kind in every status.
//
// A fake repository would prove nothing here — the entire behaviour under test
// lives in six SQL WHERE clauses.

// Only the columns PublicActivity reads. Deliberately not the full domain
// schema: if the query starts selecting something else, this test should fail
// loudly rather than quietly widen.
var testTables = []struct{ name, cols string }{
	{"communities", "id text primary key, state text, lga text"},
	{"organizations", "id text primary key, state text, lga text"},
	{"issues", "id text primary key, title text, status text, community_id text, created_at timestamptz"},
	{"petitions", "id text primary key, title text, status text, community_id text, created_at timestamptz"},
	{"consultations", "id text primary key, title text, status text, community_id text, published_at timestamptz, created_at timestamptz"},
	{"announcements", "id text primary key, title text, status text, organization_id text, published_at timestamptz, created_at timestamptz"},
	{"campaigns", "id text primary key, title text, status text, state text, lga text, published_at timestamptz, created_at timestamptz"},
	{"representative_announcements", "id text primary key, title text, status text, community_id text, published_at timestamptz, created_at timestamptz"},
}

var (
	testDB     *gorm.DB
	testDBSkip string // non-empty when Postgres could not be started
)

func freePort() (uint32, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return uint32(l.Addr().(*net.TCPAddr).Port), nil
}

// TestMain runs ONE Postgres for the whole package.
//
// # Two things here are not incidental
//
// **The runtime path must be unique to this package.** embedded-postgres
// defaults to a single shared directory (~/.embedded-postgres-go/extracted)
// and calls os.RemoveAll on it at the start of every Start(). `go test ./...`
// runs package test binaries in parallel, so once a second package in this
// service used embedded-postgres, one package's Start() would delete the
// binaries out from under the other's running initdb — which is exactly how
// this broke petitions in CI ("could not access file dict_snowball"). A
// private runtime path means the two never see each other's files.
//
// **One instance, not one per test.** Each Start() re-extracts the whole
// distribution; four tests meant four extractions and ~80s. Tests get a clean
// database by truncation instead.
func TestMain(m *testing.M) {
	os.Exit(func() int {
		// Short base name on purpose: the postgres unix socket lives under the
		// data directory and the path has a ~107 character limit.
		runtimeDir, err := os.MkdirTemp("", "cvpg")
		if err != nil {
			testDBSkip = fmt.Sprintf("temp dir: %v", err)
			return m.Run()
		}
		defer os.RemoveAll(runtimeDir)

		port, err := freePort()
		if err != nil {
			testDBSkip = fmt.Sprintf("free port: %v", err)
			return m.Run()
		}

		pg := embeddedpostgres.NewDatabase(
			embeddedpostgres.DefaultConfig().Port(port).RuntimePath(runtimeDir),
		)
		if err := pg.Start(); err != nil {
			// Not a hard failure: a machine without the ability to run the
			// embedded distribution should not fail the suite. CI does run it,
			// so the coverage is real where it counts.
			testDBSkip = fmt.Sprintf("embedded postgres unavailable: %v", err)
			return m.Run()
		}
		defer func() { _ = pg.Stop() }()

		dsn := fmt.Sprintf("host=127.0.0.1 port=%d user=postgres password=postgres dbname=postgres sslmode=disable", port)
		raw, err := sql.Open("pgx", dsn)
		if err != nil {
			testDBSkip = fmt.Sprintf("open: %v", err)
			return m.Run()
		}
		for _, tbl := range testTables {
			if _, err := raw.Exec(fmt.Sprintf("CREATE TABLE %s (%s)", tbl.name, tbl.cols)); err != nil {
				raw.Close()
				testDBSkip = fmt.Sprintf("schema %s: %v", tbl.name, err)
				return m.Run()
			}
		}
		raw.Close()

		testDB, err = gorm.Open(postgres.New(postgres.Config{DSN: dsn}), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			testDBSkip = fmt.Sprintf("gorm open: %v", err)
		}
		return m.Run()
	}())
}

// newTestDB hands back the shared database, emptied. Tests must not depend on
// each other's rows — CI runs with -shuffle=on.
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	if testDBSkip != "" {
		t.Skip(testDBSkip)
	}
	for _, tbl := range testTables {
		if err := testDB.Exec("TRUNCATE TABLE " + tbl.name).Error; err != nil {
			t.Fatalf("truncate %s: %v", tbl.name, err)
		}
	}
	return testDB
}

// seed inserts one record of every kind in every status we care about, each at
// a distinct time so ordering is unambiguous.
func seed(t *testing.T, db *gorm.DB) {
	t.Helper()
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	at := func(mins int) time.Time { return base.Add(time.Duration(mins) * time.Minute) }

	exec := func(q string, args ...any) {
		if err := db.Exec(q, args...).Error; err != nil {
			t.Fatalf("seed %q: %v", q, err)
		}
	}

	exec(`INSERT INTO communities VALUES ('c1','Kaduna','Zaria')`)
	exec(`INSERT INTO organizations VALUES ('o1','Lagos','Eti-Osa')`)

	exec(`INSERT INTO issues VALUES ('i1','issue open','OPEN','c1',?)`, at(1))
	exec(`INSERT INTO issues VALUES ('i2','issue resolved','RESOLVED','c1',?)`, at(2))
	exec(`INSERT INTO petitions VALUES ('p1','petition active','ACTIVE','c1',?)`, at(3))

	// Statuses that must never surface. Each is the state in which a record
	// exists but is not yet — or no longer — public.
	exec(`INSERT INTO consultations VALUES ('s1','consultation draft','DRAFT','c1',NULL,?)`, at(4))
	exec(`INSERT INTO consultations VALUES ('s2','consultation published','PUBLISHED','c1',?,?)`, at(5), at(0))
	exec(`INSERT INTO announcements VALUES ('a1','announcement draft','DRAFT','o1',NULL,?)`, at(6))
	exec(`INSERT INTO announcements VALUES ('a2','announcement published','PUBLISHED','o1',?,?)`, at(7), at(0))
	exec(`INSERT INTO campaigns VALUES ('m1','campaign draft','DRAFT','Kano','Fagge',NULL,?)`, at(8))
	exec(`INSERT INTO campaigns VALUES ('m2','campaign pending','PENDING_REVIEW','Kano','Fagge',NULL,?)`, at(9))
	exec(`INSERT INTO campaigns VALUES ('m3','campaign rejected','REJECTED','Kano','Fagge',NULL,?)`, at(10))
	exec(`INSERT INTO campaigns VALUES ('m4','campaign published','PUBLISHED','Kano','Fagge',?,?)`, at(11), at(0))
	exec(`INSERT INTO representative_announcements VALUES ('r1','rep draft','DRAFT','c1',NULL,?)`, at(12))
	exec(`INSERT INTO representative_announcements VALUES ('r2','rep archived','ARCHIVED','c1',?,?)`, at(13), at(0))
	exec(`INSERT INTO representative_announcements VALUES ('r3','rep published','PUBLISHED','c1',?,?)`, at(14), at(0))
}

func titles(items []PublicActivityItem) []string {
	out := make([]string, 0, len(items))
	for _, i := range items {
		out = append(out, i.Title)
	}
	return out
}

func TestPublicActivity_NeverLeaksUnpublished(t *testing.T) {
	db := newTestDB(t)
	seed(t, db)
	svc := &Service{db: db}

	items, err := svc.PublicActivity(publicActivityMax)
	if err != nil {
		t.Fatalf("PublicActivity: %v", err)
	}

	// Named explicitly rather than derived, so adding a leaky status to the
	// query cannot also silently update the expectation.
	forbidden := []string{
		"consultation draft",
		"announcement draft",
		"campaign draft",
		"campaign pending",
		"campaign rejected",
		"rep draft",
		"rep archived",
	}
	got := titles(items)
	for _, f := range forbidden {
		for _, g := range got {
			if g == f {
				t.Errorf("unpublished record %q reachable without authentication", f)
			}
		}
	}

	expected := []string{
		"issue open", "issue resolved", "petition active",
		"consultation published", "announcement published",
		"campaign published", "rep published",
	}
	for _, e := range expected {
		found := false
		for _, g := range got {
			if g == e {
				found = true
			}
		}
		if !found {
			t.Errorf("public record %q missing from activity; got %v", e, got)
		}
	}
	if len(items) != len(expected) {
		t.Errorf("expected exactly %d public records, got %d: %v", len(expected), len(items), got)
	}
}

func TestPublicActivity_NewestFirstAcrossKinds(t *testing.T) {
	db := newTestDB(t)
	seed(t, db)
	svc := &Service{db: db}

	items, err := svc.PublicActivity(publicActivityMax)
	if err != nil {
		t.Fatalf("PublicActivity: %v", err)
	}

	// The whole point of merging six queries is a single recency order. Assert
	// the full sequence, not just "sorted": each kind is queried separately, so
	// a merge bug shows up as correct-within-kind and wrong across kinds.
	want := []string{
		"rep published",          // at(14)
		"campaign published",     // at(11)
		"announcement published", // at(7)
		"consultation published", // at(5)
		"petition active",        // at(3)
		"issue resolved",         // at(2)
		"issue open",             // at(1)
	}
	got := titles(items)
	if len(got) != len(want) {
		t.Fatalf("got %d items, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("position %d: got %q, want %q\nfull order: %v", i, got[i], want[i], got)
		}
	}

	// A published record must be ordered by when it was PUBLISHED, not when it
	// was created. Every published fixture above was created at at(0), so an
	// order that fell back to created_at would produce a different sequence
	// than the one just asserted.
	for _, it := range items {
		if it.At.IsZero() {
			t.Errorf("%q has a zero timestamp", it.Title)
		}
	}
}

func TestPublicActivity_LimitIsClamped(t *testing.T) {
	db := newTestDB(t)
	seed(t, db)
	svc := &Service{db: db}

	items, err := svc.PublicActivity(2)
	if err != nil {
		t.Fatalf("PublicActivity: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("limit 2 returned %d items", len(items))
	}
	// Truncation must keep the NEWEST, not whichever kind was queried first.
	if len(items) == 2 && items[0].Title != "rep published" {
		t.Errorf("truncated list dropped the newest record; head is %q", items[0].Title)
	}

	// A caller asking for more than the cap gets the cap, not an error and not
	// an unbounded scan.
	over, err := svc.PublicActivity(9999)
	if err != nil {
		t.Fatalf("PublicActivity(9999): %v", err)
	}
	if len(over) > publicActivityMax {
		t.Errorf("limit not clamped: got %d, max %d", len(over), publicActivityMax)
	}

	zero, err := svc.PublicActivity(0)
	if err != nil {
		t.Fatalf("PublicActivity(0): %v", err)
	}
	if len(zero) == 0 {
		t.Error("limit 0 should fall back to the default, not return nothing")
	}
}

func TestPublicActivity_EmptyIsEmptySliceNotNil(t *testing.T) {
	db := newTestDB(t)
	svc := &Service{db: db}

	items, err := svc.PublicActivity(0)
	if err != nil {
		t.Fatalf("PublicActivity: %v", err)
	}
	// A nil slice marshals to JSON `null`, which the homepage would call
	// `.map()` on. This has broken a production page on this codebase before.
	if items == nil {
		t.Error("empty result is a nil slice; it must marshal as [] not null")
	}
	if len(items) != 0 {
		t.Errorf("expected no items on an empty database, got %d", len(items))
	}
}
