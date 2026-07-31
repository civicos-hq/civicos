package donations

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/civicos/organization-service/internal/domain"
	"github.com/google/uuid"
)

// Reconciliation — proving the ledger against the provider.
//
// The webhook is the only thing that settles a donation, which makes it a
// single point of failure: a delivery that never arrives, or arrives while
// the endpoint is down, leaves money that genuinely moved sitting PENDING in
// our ledger forever. Paystack retries and then gives up. Nothing in the
// system notices.
//
// This is the job that notices. It re-reads transactions directly from the
// provider and compares them against what we recorded, in two sweeps with
// deliberately different powers:
//
//   - PENDING rows are REPAIRED. A donation the provider says succeeded is
//     settled through the ordinary settle path, exactly as the missed
//     webhook would have. This is safe because it is idempotent and because
//     PENDING means we never told anyone anything.
//
//   - SETTLED rows are only ever REPORTED ON. If the provider disagrees with
//     a row we already banked, that is a human's problem to look at, not a
//     number for a background job to quietly rewrite. Auto-correcting
//     settled money would destroy the audit trail that makes the discrepancy
//     explicable in the first place.
//
// Every repair is itself reported as drift. A donation that only settled
// because reconciliation caught it means the webhook path failed, and that
// is worth knowing even though the money ended up in the right place.

// ApplyOutcome is what happened when a provider status was applied to a
// donation. Shared by the webhook and the reconciler, which both route
// through Service.applyStatus.
type ApplyOutcome string

const (
	OutcomeSettled        ApplyOutcome = "SETTLED"
	OutcomeAlreadySettled ApplyOutcome = "ALREADY_SETTLED"
	OutcomeFailed         ApplyOutcome = "FAILED"
	OutcomeAbandoned      ApplyOutcome = "ABANDONED"
	OutcomeStillPending   ApplyOutcome = "STILL_PENDING"
	OutcomeMismatch       ApplyOutcome = "MISMATCH"
	OutcomeError          ApplyOutcome = "ERROR"
)

// DriftKind classifies a disagreement between our ledger and the provider.
type DriftKind string

const (
	// DriftRecovered — the provider had a successful payment we never
	// settled. The money is now banked correctly, but the webhook path
	// failed and that needs explaining.
	DriftRecovered DriftKind = "RECOVERED_MISSED_WEBHOOK"

	// DriftAmount — we banked a different figure than the provider holds.
	// The most serious kind: our public totals are wrong.
	DriftAmount DriftKind = "AMOUNT_MISMATCH"

	DriftCurrency DriftKind = "CURRENCY_MISMATCH"

	// DriftSubaccount — the payment credited a different organization than
	// the campaign belongs to. Money reached the wrong people.
	DriftSubaccount DriftKind = "SUBACCOUNT_MISMATCH"

	// DriftNotAtProvider — we show SETTLED, the provider does not show a
	// successful transaction. Either a reversal we missed or a ledger bug.
	DriftNotAtProvider DriftKind = "SETTLED_HERE_BUT_NOT_AT_PROVIDER"

	// DriftMismatchStuck — a PENDING row the provider reports as successful
	// but whose details do not match what we opened. Cannot be repaired
	// automatically; settling it would bank a figure we never quoted.
	DriftMismatchStuck DriftKind = "PENDING_WITH_MISMATCHED_DETAILS"

	// DriftUnverifiable — the provider could not be reached for this row.
	// Not evidence of a problem, but it means this row was NOT checked, and
	// silently counting it as clean would be a lie.
	DriftUnverifiable DriftKind = "PROVIDER_UNREACHABLE"
)

// Drift is one disagreement, in a shape an admin can act on.
type Drift struct {
	DonationID  string    `json:"donationId"`
	CampaignID  string    `json:"campaignId"`
	Reference   string    `json:"reference"`
	Kind        DriftKind `json:"kind"`
	Detail      string    `json:"detail"`
	AmountMinor int64     `json:"amountMinor"`
}

// ReconcileOptions tunes a single run.
type ReconcileOptions struct {
	// PendingGrace is how old a PENDING row must be before we chase it. A
	// donor may legitimately still be on the Paystack checkout page, and
	// marking their in-flight payment abandoned out from under them would
	// be worse than waiting.
	//
	// A POINTER, so that an explicit zero is distinguishable from unset.
	// An admin investigating a specific complaint needs to say "check
	// everything now" and have it mean that; silently substituting the
	// default would answer a question they did not ask.
	PendingGrace *time.Duration

	// SettledWindow is how far back to re-check already-settled rows.
	// Bounded because this costs one provider call per row; older records
	// are covered by whichever run first saw them.
	SettledWindow time.Duration

	// Limit caps rows per sweep so a run has predictable cost.
	Limit int

	// Now is injectable for tests.
	Now time.Time
}

const (
	defaultPendingGrace  = 20 * time.Minute
	defaultSettledWindow = 48 * time.Hour
	defaultLimit         = 500
)

func (o *ReconcileOptions) applyDefaults() {
	if o.PendingGrace == nil {
		d := defaultPendingGrace
		o.PendingGrace = &d
	}
	if o.SettledWindow <= 0 {
		o.SettledWindow = defaultSettledWindow
	}
	if o.Limit <= 0 {
		o.Limit = defaultLimit
	}
	if o.Now.IsZero() {
		o.Now = time.Now().UTC()
	}
}

// ReconcileReport is the outcome of a run.
type ReconcileReport struct {
	// ID identifies this run. It is the audit row's target — reconciliation
	// is platform-wide and has no single donation to point at — and it ties
	// a drift line in the logs back to the run that produced it.
	ID             string    `json:"id"`
	RanAt          time.Time `json:"ranAt"`
	DurationMS     int64     `json:"durationMs"`
	PendingChecked int       `json:"pendingChecked"`
	SettledChecked int       `json:"settledChecked"`

	Recovered       int `json:"recovered"`
	MarkedFailed    int `json:"markedFailed"`
	MarkedAbandoned int `json:"markedAbandoned"`
	StillPending    int `json:"stillPending"`

	// RecoveredMinor is the money that would have been lost had this run
	// not happened. The single most useful number in the report.
	RecoveredMinor int64 `json:"recoveredMinor"`

	// ReceiptsSent counts donors who settled without a receipt and have now
	// been told. Non-zero means the inline send failed earlier — worth
	// noticing, even though the donor has since been emailed.
	ReceiptsSent   int `json:"receiptsSent"`
	ReceiptsFailed int `json:"receiptsFailed"`

	Drift []Drift `json:"drift"`
}

// Clean reports whether the run found nothing wrong.
func (r *ReconcileReport) Clean() bool { return len(r.Drift) == 0 }

// Reconcile re-reads transactions from the provider and repairs or reports.
//
// It returns a report even when individual rows fail to verify: a provider
// outage must not discard the findings from rows that did check out.
func (s *Service) Reconcile(ctx context.Context, opts ReconcileOptions) (*ReconcileReport, error) {
	if s.provider == nil {
		return nil, &AppError{Code: "DONATIONS_UNAVAILABLE", Message: "Donations are not enabled", Status: http.StatusServiceUnavailable}
	}
	opts.applyDefaults()
	started := time.Now()
	rep := &ReconcileReport{ID: uuid.NewString(), RanAt: opts.Now.UTC(), Drift: []Drift{}}

	if err := s.sweepPending(ctx, opts, rep); err != nil {
		return nil, err
	}
	if err := s.sweepSettled(ctx, opts, rep); err != nil {
		return nil, err
	}
	// Donors we banked but never told. Last, so a provider problem does not
	// stop it, and so it also covers anything the sweeps just recovered.
	s.sweepReceipts(opts, rep)

	rep.DurationMS = time.Since(started).Milliseconds()
	return rep, nil
}

// sweepPending chases donations we opened but never settled. This is the
// repair half.
func (s *Service) sweepPending(ctx context.Context, opts ReconcileOptions, rep *ReconcileReport) error {
	cutoff := opts.Now.Add(-*opts.PendingGrace)
	rows, err := s.repo.ListStale(domain.DonationPending, cutoff, opts.Limit)
	if err != nil {
		return err
	}

	for i := range rows {
		d := &rows[i]
		rep.PendingChecked++

		st, err := s.provider.VerifyTransaction(ctx, d.ProviderRef)
		if err != nil {
			rep.Drift = append(rep.Drift, drift(d, DriftUnverifiable, err.Error()))
			continue
		}

		outcome, note, err := s.applyStatus(d, st)
		if err != nil {
			rep.Drift = append(rep.Drift, drift(d, DriftUnverifiable, "could not apply status: "+err.Error()))
			continue
		}

		switch outcome {
		case OutcomeSettled:
			// Money that moved and that we would otherwise never have
			// recorded. Repaired, but still drift — the webhook failed.
			rep.Recovered++
			rep.RecoveredMinor += d.GrossMinor
			rep.Drift = append(rep.Drift, drift(d, DriftRecovered,
				"provider reported a successful payment that no webhook ever settled"))
		case OutcomeFailed:
			rep.MarkedFailed++
		case OutcomeAbandoned:
			rep.MarkedAbandoned++
		case OutcomeStillPending:
			rep.StillPending++
		case OutcomeMismatch:
			detail := "provider reports success but details do not match the intent"
			if note != nil {
				detail = *note
			}
			rep.Drift = append(rep.Drift, drift(d, DriftMismatchStuck, detail))
		}
	}
	return nil
}

// sweepSettled re-checks money we already banked. This half never mutates.
func (s *Service) sweepSettled(ctx context.Context, opts ReconcileOptions, rep *ReconcileReport) error {
	since := opts.Now.Add(-opts.SettledWindow)
	rows, err := s.repo.ListSettledSince(since, opts.Limit)
	if err != nil {
		return err
	}

	// One org lookup per organization, not per donation.
	subCache := map[string]string{}

	for i := range rows {
		d := &rows[i]
		rep.SettledChecked++

		st, err := s.provider.VerifyTransaction(ctx, d.ProviderRef)
		if err != nil {
			rep.Drift = append(rep.Drift, drift(d, DriftUnverifiable, err.Error()))
			continue
		}

		if !st.Succeeded {
			rep.Drift = append(rep.Drift, drift(d, DriftNotAtProvider,
				fmt.Sprintf("ledger says SETTLED; provider does not report success (failed=%t abandoned=%t)", st.Failed, st.Abandoned)))
			continue
		}
		if st.AmountMinor != d.GrossMinor {
			rep.Drift = append(rep.Drift, drift(d, DriftAmount,
				fmt.Sprintf("ledger holds %d, provider holds %d", d.GrossMinor, st.AmountMinor)))
		}
		if st.Currency != "" && !strings.EqualFold(st.Currency, d.Currency) {
			rep.Drift = append(rep.Drift, drift(d, DriftCurrency,
				fmt.Sprintf("ledger holds %s, provider holds %s", d.Currency, st.Currency)))
		}

		if st.SubaccountCode == "" {
			continue
		}
		want, ok := subCache[d.OrganizationID]
		if !ok {
			org, err := s.repo.Org(d.OrganizationID)
			if err != nil {
				continue // can't compare; not evidence of drift
			}
			if org.PSPSubaccountCode != nil {
				want = *org.PSPSubaccountCode
			}
			subCache[d.OrganizationID] = want
		}
		if want != "" && want != st.SubaccountCode {
			rep.Drift = append(rep.Drift, drift(d, DriftSubaccount,
				fmt.Sprintf("campaign belongs to an org settling to %s, payment credited %s", want, st.SubaccountCode)))
		}
	}
	return nil
}

func drift(d *domain.Donation, kind DriftKind, detail string) Drift {
	return Drift{
		DonationID:  d.ID,
		CampaignID:  d.CampaignID,
		Reference:   d.ProviderRef,
		Kind:        kind,
		Detail:      detail,
		AmountMinor: d.GrossMinor,
	}
}

// ─── Background runner ──────────────────────────────────────────────────

// StartReconciler runs reconciliation on a timer until ctx is cancelled.
//
// Deliberately fire-and-forget: a reconciliation failure must never take the
// service down, because the service still needs to accept donations while
// someone works out why the sweep is unhappy.
//
// Findings go to the operator log rather than a table. That is a real
// tradeoff — drift found at 3am lives only in logs until someone looks — but
// adding a table here would put a new financial record under AutoMigrate,
// which this repo has no migration tooling to evolve safely. The on-demand
// admin endpoint returns the same report synchronously, so an admin
// investigating a complaint never depends on log retention.
func StartReconciler(ctx context.Context, svc *Service, every time.Duration) {
	switch {
	case svc == nil || !svc.Enabled():
		log.Printf("reconciliation: DISABLED — payments are not configured")
		return
	case every <= 0:
		// Worth its own line: payments ARE live, so donations can settle,
		// but nothing is checking that they did. An operator seeing this
		// should know the safety net is off, not that payments are off.
		log.Printf("reconciliation: DISABLED by RECONCILE_INTERVAL_MINUTES=0 — payments are LIVE and unmonitored")
		return
	}
	log.Printf("reconciliation: enabled, every %s", every)

	go func() {
		// Wait one interval before the first run: at boot the service has
		// just come up, and any PENDING rows are more likely to be mid-flight
		// than genuinely stuck.
		t := time.NewTicker(every)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				runOnce(ctx, svc)
			}
		}
	}()
}

func runOnce(ctx context.Context, svc *Service) {
	defer func() {
		// A panic in a background goroutine takes the whole process with it.
		// Donations must keep working even if reconciliation is broken.
		if r := recover(); r != nil {
			log.Printf("reconciliation: PANIC recovered: %v", r)
		}
	}()

	rep, err := svc.Reconcile(ctx, ReconcileOptions{})
	if err != nil {
		log.Printf("reconciliation: run failed: %v", err)
		return
	}

	if rep.Clean() {
		log.Printf("reconciliation: run=%s clean (pending=%d settled=%d receipts=%d, %dms)",
			rep.ID, rep.PendingChecked, rep.SettledChecked, rep.ReceiptsSent, rep.DurationMS)
		return
	}

	// Loud, and one line per finding: these are the lines an operator greps
	// for, and a summary count alone would not say which donation to look at.
	log.Printf("reconciliation: run=%s %d DRIFT (pending=%d settled=%d recovered=%d worth %d minor units, %dms)",
		rep.ID, len(rep.Drift), rep.PendingChecked, rep.SettledChecked, rep.Recovered, rep.RecoveredMinor, rep.DurationMS)
	for _, d := range rep.Drift {
		log.Printf("reconciliation: DRIFT %s donation=%s campaign=%s ref=%s amount=%d — %s",
			d.Kind, d.DonationID, d.CampaignID, d.Reference, d.AmountMinor, d.Detail)
	}
}
