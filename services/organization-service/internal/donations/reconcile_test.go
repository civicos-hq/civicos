package donations

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/civicos/organization-service/internal/domain"
)

// aged backdates a donation so the pending sweep will consider it.
func aged(d *domain.Donation, age time.Duration) *domain.Donation {
	d.CreatedAt = time.Now().UTC().Add(-age)
	return d
}

func run(t *testing.T, svc *Service) *ReconcileReport {
	t.Helper()
	rep, err := svc.Reconcile(context.Background(), ReconcileOptions{})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	return rep
}

func driftKinds(rep *ReconcileReport) []DriftKind {
	out := make([]DriftKind, 0, len(rep.Drift))
	for _, d := range rep.Drift {
		out = append(out, d.Kind)
	}
	return out
}

func hasDrift(rep *ReconcileReport, k DriftKind) bool {
	for _, d := range rep.Drift {
		if d.Kind == k {
			return true
		}
	}
	return false
}

// ─── The repair half ────────────────────────────────────────────────────

// The reason this job exists: Paystack took the money, the webhook never
// arrived (or arrived at a 404), and without reconciliation that donation is
// invisible forever.
func TestReconcile_RecoversAPaymentNoWebhookEverSettled(t *testing.T) {
	st, svc, d := newSettledFixture(t, 5_000_000)
	aged(d, time.Hour)
	st.p.says(d.ProviderRef, TransactionStatus{
		Succeeded: true, AmountMinor: 5_000_000, Currency: "NGN", PSPFeeMinor: 75_000,
	})

	rep := run(t, svc)

	if got := st.donations[d.ID].Status; got != domain.DonationSettled {
		t.Fatalf("status = %s, want SETTLED — the whole point is recovering this row", got)
	}
	if rep.Recovered != 1 {
		t.Fatalf("Recovered = %d, want 1", rep.Recovered)
	}
	if rep.RecoveredMinor != 5_000_000 {
		t.Fatalf("RecoveredMinor = %d, want 5000000", rep.RecoveredMinor)
	}
	// The money is now banked correctly, but the webhook path still failed.
	// Reporting it as clean would hide a broken delivery pipeline.
	if !hasDrift(rep, DriftRecovered) {
		t.Fatalf("a recovered payment must still be reported as drift, got %v", driftKinds(rep))
	}
	if got := st.campaigns[d.CampaignID].RaisedMinor; got != 5_000_000 {
		t.Fatalf("campaign total = %d, want 5000000 — recovery must rebuild the projection", got)
	}
}

// Recovery goes through the ordinary settle path, so running twice must not
// double-count. This is the property that makes a background job safe to run
// on a timer.
func TestReconcile_IsIdempotent(t *testing.T) {
	st, svc, d := newSettledFixture(t, 3_000_000)
	aged(d, time.Hour)
	st.p.says(d.ProviderRef, TransactionStatus{Succeeded: true, AmountMinor: 3_000_000, Currency: "NGN"})

	first := run(t, svc)
	second := run(t, svc)

	if first.Recovered != 1 {
		t.Fatalf("first run should recover once, got %d", first.Recovered)
	}
	if second.Recovered != 0 {
		t.Fatalf("second run recovered %d — it is double-counting", second.Recovered)
	}
	if got := st.campaigns[d.CampaignID].RaisedMinor; got != 3_000_000 {
		t.Fatalf("campaign total = %d after two runs, want 3000000", got)
	}
}

// A donor may still be sitting on the Paystack checkout page. Chasing that
// row and marking it abandoned would be worse than waiting.
func TestReconcile_LeavesInFlightDonationsAlone(t *testing.T) {
	st, svc, d := newSettledFixture(t, 1_000_000)
	aged(d, 2*time.Minute) // well inside the grace period
	st.p.says(d.ProviderRef, TransactionStatus{Abandoned: true})

	rep := run(t, svc)

	if rep.PendingChecked != 0 {
		t.Fatalf("checked %d in-flight donations, want 0", rep.PendingChecked)
	}
	if got := st.donations[d.ID].Status; got != domain.DonationPending {
		t.Fatalf("status = %s, want PENDING — a donor mid-checkout was written off", got)
	}
}

func TestReconcile_MarksAbandonedAndFailed(t *testing.T) {
	st, svc, d := newSettledFixture(t, 1_000_000)
	aged(d, time.Hour)
	st.p.says(d.ProviderRef, TransactionStatus{Abandoned: true})

	rep := run(t, svc)

	if rep.MarkedAbandoned != 1 {
		t.Fatalf("MarkedAbandoned = %d, want 1", rep.MarkedAbandoned)
	}
	if got := st.donations[d.ID].Status; got != domain.DonationAbandoned {
		t.Fatalf("status = %s, want ABANDONED", got)
	}
	if st.campaigns[d.CampaignID].RaisedMinor != 0 {
		t.Fatalf("an abandoned donation moved the campaign total")
	}
}

// A PENDING row the provider reports as successful for a DIFFERENT amount
// must not be auto-settled: banking a figure we never quoted the donor is
// worse than leaving the row for a human.
func TestReconcile_RefusesToRepairAMismatchedAmount(t *testing.T) {
	st, svc, d := newSettledFixture(t, 2_000_000)
	aged(d, time.Hour)
	st.p.says(d.ProviderRef, TransactionStatus{Succeeded: true, AmountMinor: 9_900_000, Currency: "NGN"})

	rep := run(t, svc)

	if got := st.donations[d.ID].Status; got != domain.DonationPending {
		t.Fatalf("status = %s, want PENDING — a mismatched amount was banked", got)
	}
	if !hasDrift(rep, DriftMismatchStuck) {
		t.Fatalf("want PENDING_WITH_MISMATCHED_DETAILS, got %v", driftKinds(rep))
	}
	if rep.Recovered != 0 {
		t.Fatalf("a mismatch must not count as a recovery")
	}
	if st.campaigns[d.CampaignID].RaisedMinor != 0 {
		t.Fatalf("campaign total moved on a mismatched row")
	}
}

// ─── The report-only half ───────────────────────────────────────────────

// Settled money is never rewritten by a background job. The drift is
// reported; the ledger and the audit trail stay intact for a human.
func TestReconcile_NeverMutatesSettledRows(t *testing.T) {
	st, svc, d := newSettledFixture(t, 4_000_000)
	aged(d, time.Hour)
	st.p.says(d.ProviderRef, TransactionStatus{Succeeded: true, AmountMinor: 4_000_000, Currency: "NGN"})
	run(t, svc) // settle it legitimately

	before := *st.donations[d.ID]
	totalBefore := st.campaigns[d.CampaignID].RaisedMinor

	// Now the provider starts disagreeing about the amount. Nothing left
	// for the pending sweep to do, so any write at all comes from the
	// settled sweep — which must only ever report.
	st.p.says(d.ProviderRef, TransactionStatus{Succeeded: true, AmountMinor: 4_500_000, Currency: "NGN"})
	writesBefore := st.writeCalls
	rep := run(t, svc)

	if st.writeCalls != writesBefore {
		t.Fatalf("the settled sweep made %d store writes, want 0 — it must report drift, never rewrite banked money",
			st.writeCalls-writesBefore)
	}

	if !hasDrift(rep, DriftAmount) {
		t.Fatalf("want AMOUNT_MISMATCH, got %v", driftKinds(rep))
	}
	after := st.donations[d.ID]
	if after.GrossMinor != before.GrossMinor || after.Status != before.Status {
		t.Fatalf("settled row was mutated: %d/%s -> %d/%s",
			before.GrossMinor, before.Status, after.GrossMinor, after.Status)
	}
	if got := st.campaigns[d.CampaignID].RaisedMinor; got != totalBefore {
		t.Fatalf("campaign total rewritten by reconciliation: %d -> %d", totalBefore, got)
	}
}

// We banked it; the provider has no successful transaction. Either a
// reversal we missed or a ledger bug — both need a human.
func TestReconcile_FlagsSettledHereButNotAtProvider(t *testing.T) {
	st, svc, d := newSettledFixture(t, 2_500_000)
	aged(d, time.Hour)
	st.p.says(d.ProviderRef, TransactionStatus{Succeeded: true, AmountMinor: 2_500_000, Currency: "NGN"})
	run(t, svc)

	st.p.says(d.ProviderRef, TransactionStatus{Failed: true, AmountMinor: 2_500_000})
	rep := run(t, svc)

	if !hasDrift(rep, DriftNotAtProvider) {
		t.Fatalf("want SETTLED_HERE_BUT_NOT_AT_PROVIDER, got %v", driftKinds(rep))
	}
	if got := st.donations[d.ID].Status; got != domain.DonationSettled {
		t.Fatalf("status = %s — reconciliation reversed settled money on its own", got)
	}
}

// Money reached the wrong organization. The most serious thing this job can
// find, and invisible from our side without asking the provider.
func TestReconcile_FlagsPaymentCreditedToTheWrongOrg(t *testing.T) {
	st, svc, d := newSettledFixture(t, 1_500_000)
	aged(d, time.Hour)
	st.p.says(d.ProviderRef, TransactionStatus{
		Succeeded: true, AmountMinor: 1_500_000, Currency: "NGN", SubaccountCode: "ACCT_someone_else",
	})

	rep := run(t, svc)

	if !hasDrift(rep, DriftSubaccount) {
		t.Fatalf("want SUBACCOUNT_MISMATCH, got %v", driftKinds(rep))
	}
}

// A provider outage means the row was NOT checked. Counting it as clean
// would make an unreachable provider look like a healthy ledger.
func TestReconcile_UnreachableProviderIsNotSilentlyClean(t *testing.T) {
	st, svc, d := newSettledFixture(t, 1_000_000)
	aged(d, time.Hour)
	st.p.failsFor(d.ProviderRef, errors.New("paystack timeout"))

	rep := run(t, svc)

	if rep.Clean() {
		t.Fatalf("an unverifiable row must not report clean")
	}
	if !hasDrift(rep, DriftUnverifiable) {
		t.Fatalf("want PROVIDER_UNREACHABLE, got %v", driftKinds(rep))
	}
	if got := st.donations[d.ID].Status; got != domain.DonationPending {
		t.Fatalf("status = %s, want PENDING — a timeout must not decide anything", got)
	}
}

// One row failing to verify must not discard the findings from the rest.
func TestReconcile_OneBadRowDoesNotAbortTheRun(t *testing.T) {
	st, svc, bad := newSettledFixture(t, 1_000_000)
	aged(bad, time.Hour)
	st.p.failsFor(bad.ProviderRef, errors.New("boom"))

	good := &domain.Donation{
		ID: "d-good", CampaignID: bad.CampaignID, OrganizationID: bad.OrganizationID,
		Currency: "NGN", GrossMinor: 2_000_000, Status: domain.DonationPending,
		Provider: "paystack", ProviderRef: "civicos_good", IdempotencyKey: "k-good",
		CreatedAt: time.Now().UTC().Add(-time.Hour),
	}
	st.donations[good.ID] = good
	st.p.says(good.ProviderRef, TransactionStatus{Succeeded: true, AmountMinor: 2_000_000, Currency: "NGN"})

	rep := run(t, svc)

	if rep.Recovered != 1 {
		t.Fatalf("Recovered = %d, want 1 — a failing row aborted the sweep", rep.Recovered)
	}
	if st.donations[good.ID].Status != domain.DonationSettled {
		t.Fatalf("the healthy row was not recovered")
	}
}

func TestReconcile_CleanLedgerReportsClean(t *testing.T) {
	st, svc, d := newSettledFixture(t, 1_000_000)
	aged(d, time.Hour)
	st.p.says(d.ProviderRef, TransactionStatus{Succeeded: true, AmountMinor: 1_000_000, Currency: "NGN"})
	run(t, svc) // recover it; that run reports DriftRecovered

	// Now everything agrees.
	rep := run(t, svc)
	if !rep.Clean() {
		t.Fatalf("a ledger that agrees with the provider should be clean, got %v", driftKinds(rep))
	}
	if rep.SettledChecked != 1 {
		t.Fatalf("SettledChecked = %d, want 1 — settled rows must actually be re-checked", rep.SettledChecked)
	}
}

func TestReconcile_RequiresAProvider(t *testing.T) {
	svc := NewService(newFakeStore(), nil, 250, "")
	_, err := svc.Reconcile(context.Background(), ReconcileOptions{})
	if !isCode(err, "DONATIONS_UNAVAILABLE") {
		t.Fatalf("want DONATIONS_UNAVAILABLE, got %v", err)
	}
}

// An admin investigating a complaint asks for zero grace: check everything
// now. Substituting the default because zero looks like "unset" would answer
// a question they did not ask, and the endpoint would look like it worked.
func TestReconcile_ExplicitZeroGraceChecksEverything(t *testing.T) {
	st, svc, d := newSettledFixture(t, 1_000_000)
	aged(d, time.Second) // far inside the default grace
	st.p.says(d.ProviderRef, TransactionStatus{Succeeded: true, AmountMinor: 1_000_000, Currency: "NGN"})

	zero := time.Duration(0)
	rep, err := svc.Reconcile(context.Background(), ReconcileOptions{PendingGrace: &zero})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if rep.PendingChecked != 1 {
		t.Fatalf("PendingChecked = %d, want 1 — an explicit zero grace was ignored", rep.PendingChecked)
	}
	if st.donations[d.ID].Status != domain.DonationSettled {
		t.Fatalf("row not recovered despite zero grace")
	}
}

// Omitting the option still means the default, not zero.
func TestReconcile_UnsetGraceStillDefaults(t *testing.T) {
	st, svc, d := newSettledFixture(t, 1_000_000)
	aged(d, time.Second)
	st.p.says(d.ProviderRef, TransactionStatus{Succeeded: true, AmountMinor: 1_000_000, Currency: "NGN"})

	rep := run(t, svc)
	if rep.PendingChecked != 0 {
		t.Fatalf("PendingChecked = %d, want 0 — unset grace must still protect in-flight donors", rep.PendingChecked)
	}
	_ = st
}

// Each run is identifiable, so a drift line in the logs and the audit row
// can be tied back to the run that produced them.
func TestReconcile_RunHasAnID(t *testing.T) {
	_, svc, _ := newSettledFixture(t, 1_000_000)
	a, b := run(t, svc), run(t, svc)
	if a.ID == "" || b.ID == "" {
		t.Fatal("a run must carry an id — it is the audit row's target")
	}
	if a.ID == b.ID {
		t.Fatal("two runs share an id")
	}
}

// ─── Drift persistence ──────────────────────────────────────────────────

// The reason this exists: drift found by the hourly sweep used to live only
// in the operator log, so a disagreement discovered at 3am waited for someone
// to grep for it.
func TestDrift_SurvivesTheRunThatFoundIt(t *testing.T) {
	st, svc, d := newSettledFixture(t, 2_500_000)
	aged(d, time.Hour)
	st.p.says(d.ProviderRef, TransactionStatus{Succeeded: true, AmountMinor: 2_500_000, Currency: "NGN"})
	run(t, svc) // settle it
	st.p.says(d.ProviderRef, TransactionStatus{Failed: true, AmountMinor: 2_500_000})

	rep := run(t, svc)

	if !hasDrift(rep, DriftNotAtProvider) {
		t.Fatalf("expected drift, got %v", driftKinds(rep))
	}
	open, _ := st.CountOpenFindings()
	if open != 1 {
		t.Fatalf("open findings = %d, want 1 — the finding did not outlive the run", open)
	}
	got, _ := st.ListFindings(false, 10)
	if got[0].Kind != string(DriftNotAtProvider) || got[0].DonationID != d.ID {
		t.Fatalf("wrong finding recorded: %+v", got[0])
	}
	if got[0].RunID == "" {
		t.Error("finding is not traceable back to the run that produced it")
	}
}

// The sweep runs hourly and re-detects the same unresolved drift every pass.
// A row per detection would bury the finding that is new under repeats of one
// that is not.
func TestDrift_RepeatedDetectionUpdatesRatherThanDuplicates(t *testing.T) {
	st, svc, d := newSettledFixture(t, 2_500_000)
	aged(d, time.Hour)
	st.p.says(d.ProviderRef, TransactionStatus{Succeeded: true, AmountMinor: 2_500_000, Currency: "NGN"})
	run(t, svc)
	st.p.says(d.ProviderRef, TransactionStatus{Failed: true, AmountMinor: 2_500_000})

	run(t, svc)
	run(t, svc)
	run(t, svc)

	open, _ := st.CountOpenFindings()
	if open != 1 {
		t.Fatalf("open findings = %d after three sweeps, want 1", open)
	}
	got, _ := st.ListFindings(false, 10)
	if got[0].TimesSeen < 3 {
		t.Fatalf("timesSeen = %d, want at least 3 — repeat detections are not being counted", got[0].TimesSeen)
	}
}

// Money already banked correctly must not sit in the list forever. A
// permanently open finding for something already fixed trains admins to
// ignore the list, which is how a real finding gets missed.
func TestDrift_RecoveredWebhookIsReportedButNotFiled(t *testing.T) {
	st, svc, d := newSettledFixture(t, 5_000_000)
	aged(d, time.Hour)
	st.p.says(d.ProviderRef, TransactionStatus{Succeeded: true, AmountMinor: 5_000_000, Currency: "NGN"})

	rep := run(t, svc)

	if !hasDrift(rep, DriftRecovered) {
		t.Fatal("recovery should still be reported in the run")
	}
	if open, _ := st.CountOpenFindings(); open != 0 {
		t.Fatalf("open findings = %d — a recovered payment should not leave a standing finding", open)
	}
}

// Resolution is manual, attributed, and explained.
func TestDrift_ResolutionIsAttributed(t *testing.T) {
	st, svc, d := newSettledFixture(t, 2_500_000)
	aged(d, time.Hour)
	st.p.says(d.ProviderRef, TransactionStatus{Succeeded: true, AmountMinor: 2_500_000, Currency: "NGN"})
	run(t, svc)
	st.p.says(d.ProviderRef, TransactionStatus{Failed: true, AmountMinor: 2_500_000})
	run(t, svc)

	got, _ := st.ListFindings(false, 10)
	if err := svc.ResolveFinding(got[0].ID, "admin-1", "Chidi Admin", "Reversed at the bank; confirmed with Paystack support."); err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if open, _ := st.CountOpenFindings(); open != 0 {
		t.Fatalf("open findings = %d after resolving, want 0", open)
	}
	all, _ := st.ListFindings(true, 10)
	if all[0].ResolvedByName == nil || *all[0].ResolvedByName != "Chidi Admin" {
		t.Fatal("resolution is not attributed to a named person")
	}
	if all[0].ResolutionNote == nil || *all[0].ResolutionNote == "" {
		t.Fatal("resolution has no explanation — this is the audit trail for money that did not reconcile")
	}
}

// Drift that comes back after being resolved is a regression, not history.
func TestDrift_ReappearingAfterResolutionReopens(t *testing.T) {
	st, svc, d := newSettledFixture(t, 2_500_000)
	aged(d, time.Hour)
	st.p.says(d.ProviderRef, TransactionStatus{Succeeded: true, AmountMinor: 2_500_000, Currency: "NGN"})
	run(t, svc)
	st.p.says(d.ProviderRef, TransactionStatus{Failed: true, AmountMinor: 2_500_000})
	run(t, svc)

	got, _ := st.ListFindings(false, 10)
	_ = svc.ResolveFinding(got[0].ID, "admin-1", "Chidi", "Looked into it.")
	if open, _ := st.CountOpenFindings(); open != 0 {
		t.Fatal("precondition: should be resolved")
	}

	run(t, svc) // still failing at the provider

	if open, _ := st.CountOpenFindings(); open != 1 {
		t.Fatalf("open findings = %d — drift that returned stayed closed and is now invisible", open)
	}
}

// A storage failure must not make a correct reconciliation run look failed.
func TestDrift_ReportIsStillReturnedIfPersistenceFails(t *testing.T) {
	st, svc, d := newSettledFixture(t, 2_500_000)
	aged(d, time.Hour)
	st.p.says(d.ProviderRef, TransactionStatus{Succeeded: true, AmountMinor: 2_500_000, Currency: "NGN"})
	run(t, svc)
	st.p.says(d.ProviderRef, TransactionStatus{Failed: true, AmountMinor: 2_500_000})
	st.findingErr = true

	rep, err := svc.Reconcile(context.Background(), ReconcileOptions{})
	if err != nil {
		t.Fatalf("a storage failure must not fail the run: %v", err)
	}
	if !hasDrift(rep, DriftNotAtProvider) {
		t.Fatal("the report should still describe the drift it found")
	}
}
