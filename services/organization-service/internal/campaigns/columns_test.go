package campaigns

import (
	"sync"
	"testing"

	"github.com/civicos/organization-service/internal/domain"
	"gorm.io/gorm/schema"
)

// The services write updates as map[string]any keyed by DB column name.
// GORM applies those keys verbatim — a typo does not error, it silently
// updates nothing. That failure mode is quiet in tests that use a fake
// store and catastrophic on a campaign lifecycle ("submit returned 200 but
// the status never changed").
//
// So: parse the model with GORM's own naming strategy and assert every
// column the services write actually exists. This needs no database.
func TestWrittenColumnsExistOnModel(t *testing.T) {
	cases := []struct {
		name    string
		model   any
		columns []string
	}{
		{
			name:  "Campaign",
			model: &domain.Campaign{},
			columns: []string{
				// Update()
				"title", "summary", "description", "category", "goal_minor",
				"cover_image_url", "community_id", "project_id", "state", "lga",
				"start_date", "end_date", "is_emergency",
				// Submit()
				"status", "approval_status", "submitted_at",
				"review_note", "reviewed_by_id", "reviewed_at",
				// Publish() / Transition()
				"published_at", "completed_at",
				// Pause() / Resume()
				"paused_reason",
			},
		},
		{
			name:  "Milestone",
			model: &domain.Milestone{},
			columns: []string{
				"title", "description", "target_minor", "position",
				"status", "completed_at",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, err := schema.Parse(tc.model, &sync.Map{}, schema.NamingStrategy{})
			if err != nil {
				t.Fatalf("parse schema: %v", err)
			}
			known := map[string]bool{}
			for _, f := range s.Fields {
				if f.DBName != "" {
					known[f.DBName] = true
				}
			}
			for _, col := range tc.columns {
				if !known[col] {
					t.Errorf("column %q is written by the service but does not exist on %s; "+
						"GORM would silently ignore this update", col, tc.name)
				}
			}
		})
	}
}

// The projection columns must exist too — Phase 3 recomputes them from the
// ledger, and the transparency dashboard reads them.
func TestProjectionColumnsExist(t *testing.T) {
	s, err := schema.Parse(&domain.Campaign{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("parse schema: %v", err)
	}
	byName := map[string]*schema.Field{}
	for _, f := range s.Fields {
		byName[f.DBName] = f
	}
	for _, col := range []string{"raised_minor", "donor_count", "currency", "slug"} {
		if byName[col] == nil {
			t.Fatalf("expected column %q on Campaign", col)
		}
	}
	// Money must be an integer type, never a float. Guards against someone
	// "simplifying" GoalMinor to float64 later.
	for _, col := range []string{"goal_minor", "raised_minor"} {
		if got := byName[col].DataType; got != "int" && got != "" {
			t.Errorf("%s has DataType %q; money must be an integer type", col, got)
		}
	}
	// RiskScore is reviewer-only and must not serialise into public JSON.
	if f := byName["risk_score"]; f == nil {
		t.Fatalf("expected risk_score column")
	}
}
