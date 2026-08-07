package migrate

import (
	"strings"
	"testing"
)

// Every table a migration declares a prerequisite on must have a known
// owner, or the error tells an operator a table is missing without saying
// which service creates it — which is most of what makes the message
// worth having.
func TestEveryPrerequisiteHasAKnownOwner(t *testing.T) {
	for version, tables := range prerequisites {
		for _, table := range tables {
			if _, ok := tableOwner[table]; !ok {
				t.Errorf("migration %d requires %q but no service is recorded as its owner", version, table)
			}
		}
	}
}

// The map must describe the migrations that actually exist. A migration
// added without an entry here silently loses its guard, which is the exact
// failure this package was written to stop.
func TestPrerequisitesCoverEveryMigrationThatNeedsOne(t *testing.T) {
	files, err := migrations.ReadDir("migrations")
	if err != nil {
		t.Fatalf("read embedded migrations: %v", err)
	}

	// Versions with no external prerequisite. Listed explicitly so adding a
	// migration forces a decision rather than defaulting to unguarded.
	noPrerequisite := map[int64]bool{
		1: true, // baseline — creates nothing, touches nothing
	}

	seen := 0
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".sql") {
			continue
		}
		seen++
		var version int64
		if _, err := fmtSscan(f.Name(), &version); err != nil {
			t.Fatalf("migration %q does not start with a version number", f.Name())
		}
		_, guarded := prerequisites[version]
		if !guarded && !noPrerequisite[version] {
			t.Errorf("migration %d (%s) has neither a prerequisite entry nor an explicit exemption", version, f.Name())
		}
	}
	if seen == 0 {
		t.Fatal("no migrations found — the embed pattern is wrong")
	}
}

// Already-applied migrations must not be re-checked. Their prerequisites
// describe a past state, and a table that has since been legitimately
// dropped or renamed would otherwise block every future boot.
func TestAppliedMigrationsAreNotChecked(t *testing.T) {
	// checkPrerequisites collects nothing when every version is at or below
	// the current one, so it returns before touching the database — which
	// is what lets this run without one.
	if err := checkPrerequisites(nil, nil, 99); err != nil {
		t.Fatalf("a fully migrated database must need no prerequisite check, got %v", err)
	}
}

// fmtSscan pulls the leading version number off a goose filename
// ("00005_add_fk_constraints.sql" → 5).
func fmtSscan(name string, out *int64) (int, error) {
	var n int64
	i := 0
	for ; i < len(name) && name[i] >= '0' && name[i] <= '9'; i++ {
		n = n*10 + int64(name[i]-'0')
	}
	if i == 0 {
		return 0, errNoVersion
	}
	*out = n
	return 1, nil
}

var errNoVersion = &versionError{}

type versionError struct{}

func (*versionError) Error() string { return "no leading version number" }
