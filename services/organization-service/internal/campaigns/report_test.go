package campaigns

import (
	"strings"
	"testing"

	"github.com/civicos/organization-service/internal/domain"
)

const goodReport = "We distributed food and water to 180 households across Sabon Gari, " +
	"and completed two of the three planned boreholes before the season ended."

// A final report has to say something. REPORTED used to be a bare status
// flip, which let an organization declare itself accountable having published
// nothing at all.
func TestFileReport_RefusesAnEmptyAccount(t *testing.T) {
	st, svc, c := reportFixture(t)
	for _, body := range []string{"", "   ", "Done."} {
		if _, err := svc.FileReport(c.ID, ReportInput{Body: body}); err == nil {
			t.Fatalf("accepted a %q final report", body)
		}
		if st.items[c.ID].Status == domain.CampaignReported {
			t.Fatal("status moved despite the report being rejected")
		}
	}
}

func TestFileReport_PublishesTheAccount(t *testing.T) {
	st, svc, c := reportFixture(t)

	updated, err := svc.FileReport(c.ID, ReportInput{
		Body:        goodReport,
		Attachments: []string{"https://cdn.example.org/final.pdf"},
	})
	if err != nil {
		t.Fatalf("file: %v", err)
	}
	if updated.Status != domain.CampaignReported {
		t.Fatalf("status = %s, want REPORTED", updated.Status)
	}
	if updated.FinalReportBody == nil || !strings.Contains(*updated.FinalReportBody, "180 households") {
		t.Fatalf("report body not stored: %v", updated.FinalReportBody)
	}
	if len(updated.FinalReportURLs) != 1 {
		t.Fatalf("attachments lost: %v", updated.FinalReportURLs)
	}
	if updated.ReportedAt == nil {
		t.Fatal("reportedAt not set")
	}
	_ = st
}

// Filing is allowed with money unexplained — blocking would strand campaigns
// in COMPLETED and teach organizations that silence is safer. But the
// shortfall must be recorded alongside the report.
func TestFileReport_RecordsWhatWasStillUnexplained(t *testing.T) {
	_, svc, c := reportFixture(t)
	svc.WithSpend(fakeSpend{unreported: 4_000_000})

	updated, err := svc.FileReport(c.ID, ReportInput{Body: goodReport})
	if err != nil {
		t.Fatalf("file: %v", err)
	}
	if updated.UnaccountedAtReportMinor == nil {
		t.Fatal("shortfall not recorded — an incomplete report would look complete")
	}
	if *updated.UnaccountedAtReportMinor != 4_000_000 {
		t.Fatalf("recorded %d, want 4000000", *updated.UnaccountedAtReportMinor)
	}
}

// The figure is frozen. A live number would let spend published afterwards
// make an incomplete report look complete in hindsight.
func TestFileReport_ShortfallIsASnapshot(t *testing.T) {
	_, svc, c := reportFixture(t)
	spend := fakeSpend{unreported: 4_000_000}
	svc.WithSpend(spend)

	updated, err := svc.FileReport(c.ID, ReportInput{Body: goodReport})
	if err != nil {
		t.Fatalf("file: %v", err)
	}
	// More spend is published later; the recorded figure must not move.
	svc.WithSpend(fakeSpend{unreported: 0})
	after, err := svc.Get(c.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if after.UnaccountedAtReportMinor == nil || *after.UnaccountedAtReportMinor != 4_000_000 {
		t.Fatalf("snapshot changed to %v — the report has been retroactively made to look complete",
			after.UnaccountedAtReportMinor)
	}
	_ = updated
}

// A campaign can only be reported once it is COMPLETED.
func TestFileReport_OnlyFromCompleted(t *testing.T) {
	st, svc, c := reportFixture(t)
	for _, from := range []domain.CampaignStatus{
		domain.CampaignDraft, domain.CampaignPublished, domain.CampaignFunded, domain.CampaignReported,
	} {
		st.items[c.ID].Status = from
		if _, err := svc.FileReport(c.ID, ReportInput{Body: goodReport}); err == nil {
			t.Errorf("filed a report from %s", from)
		}
	}
}

// Spend reporting being unavailable must not block an organization from
// closing out its campaign.
func TestFileReport_WorksWithoutSpendReporting(t *testing.T) {
	_, svc, c := reportFixture(t)
	updated, err := svc.FileReport(c.ID, ReportInput{Body: goodReport})
	if err != nil {
		t.Fatalf("file: %v", err)
	}
	if updated.Status != domain.CampaignReported {
		t.Fatalf("status = %s", updated.Status)
	}
}

func reportFixture(t *testing.T) (*fakeStore, *Service, *domain.Campaign) {
	t.Helper()
	st := newFakeStore()
	c := &domain.Campaign{
		ID: "camp-1", OrganizationID: "org-1", Title: "Flood relief",
		Slug: "flood-relief", Currency: "NGN", GoalMinor: 200_000_000,
		RaisedMinor: 10_000_000, Status: domain.CampaignCompleted,
	}
	st.items[c.ID] = c
	return st, NewService(st), c
}

type fakeSpend struct{ unreported int64 }

func (f fakeSpend) SummaryFor(_ string, _ int64, _ string) (*SpendSummary, error) {
	return &SpendSummary{UnreportedMinor: f.unreported, PerMilestone: map[string]int64{}}, nil
}
