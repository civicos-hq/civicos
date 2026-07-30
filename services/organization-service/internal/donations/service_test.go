package donations

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/civicos/organization-service/internal/domain"
	"github.com/google/uuid"
)

// These exercise the webhook path against a fake store, because that is
// where a mistake silently corrupts the ledger: a double-count, or banking a
// payment that does not match what was opened.

func TestWebhook_ReplayIsANoOp(t *testing.T) {
	st, svc, d := newSettledFixture(t, 2_000_000)

	if err := svc.HandleWebhook(context.Background(), st.p.body(d.ProviderRef, 2_000_000), st.p.sig(d.ProviderRef, 2_000_000)); err != nil {
		t.Fatalf("first delivery: %v", err)
	}
	if got := st.campaigns[d.CampaignID].RaisedMinor; got != 2_000_000 {
		t.Fatalf("after one delivery raised = %d, want 2000000", got)
	}

	// Paystack retries. Deliver the identical event four more times.
	for i := 0; i < 4; i++ {
		if err := svc.HandleWebhook(context.Background(), st.p.body(d.ProviderRef, 2_000_000), st.p.sig(d.ProviderRef, 2_000_000)); err != nil {
			t.Fatalf("replay %d: %v", i, err)
		}
	}
	if got := st.campaigns[d.CampaignID].RaisedMinor; got != 2_000_000 {
		t.Fatalf("replays inflated the total to %d — a donor's money would be double counted", got)
	}
	if got := st.donations[d.ID].Status; got != domain.DonationSettled {
		t.Fatalf("status = %s", got)
	}
	if st.settleCalls > 1 {
		t.Fatalf("Settle ran %d times; the event dedupe should have short-circuited replays", st.settleCalls)
	}
}

// A signed message proves it came from Paystack. It does not prove it
// describes the transaction we opened.
func TestWebhook_RefusesToSettleOnMismatch(t *testing.T) {
	t.Run("inflated amount", func(t *testing.T) {
		st, svc, d := newSettledFixture(t, 2_000_000)
		// Correctly signed, but for twice the money.
		if err := svc.HandleWebhook(context.Background(), st.p.body(d.ProviderRef, 4_000_000), st.p.sig(d.ProviderRef, 4_000_000)); err != nil {
			t.Fatalf("handle: %v", err)
		}
		if st.donations[d.ID].Status == domain.DonationSettled {
			t.Fatalf("a mismatched amount was SETTLED — the ledger would now disagree with Paystack")
		}
		if st.campaigns[d.CampaignID].RaisedMinor != 0 {
			t.Fatalf("mismatched donation still moved the total")
		}
	})

	t.Run("wrong currency", func(t *testing.T) {
		st, svc, d := newSettledFixture(t, 2_000_000)
		body := st.p.bodyCurrency(d.ProviderRef, 2_000_000, "GHS")
		if err := svc.HandleWebhook(context.Background(), body, st.p.sigFor(body)); err != nil {
			t.Fatalf("handle: %v", err)
		}
		if st.donations[d.ID].Status == domain.DonationSettled {
			t.Fatalf("a currency mismatch was settled")
		}
	})
}

func TestWebhook_ForgedSignatureNeverSettles(t *testing.T) {
	st, svc, d := newSettledFixture(t, 2_000_000)
	err := svc.HandleWebhook(context.Background(), st.p.body(d.ProviderRef, 2_000_000), "deadbeef")
	if err == nil {
		t.Fatalf("a forged signature was accepted")
	}
	if st.donations[d.ID].Status == domain.DonationSettled {
		t.Fatalf("forged delivery settled a donation")
	}
	// The attempt must still be on file — repeated failures are the signal
	// that someone is probing the endpoint.
	if len(st.webhooks) == 0 {
		t.Fatalf("rejected delivery was not recorded")
	}
	if st.webhooks[0].Verified {
		t.Fatalf("rejected delivery recorded as verified")
	}
}

func TestWebhook_FailedChargeDoesNotCount(t *testing.T) {
	st, svc, d := newSettledFixture(t, 2_000_000)
	body := st.p.bodyStatus(d.ProviderRef, 2_000_000, "failed")
	if err := svc.HandleWebhook(context.Background(), body, st.p.sigFor(body)); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if st.donations[d.ID].Status != domain.DonationFailed {
		t.Fatalf("status = %s, want FAILED", st.donations[d.ID].Status)
	}
	if st.campaigns[d.CampaignID].RaisedMinor != 0 {
		t.Fatalf("a failed charge moved the total")
	}
}

// Reaching the goal is asserted by the platform from ledger truth.
func TestWebhook_SettingGoalFlipsToFunded(t *testing.T) {
	st, svc, d := newSettledFixture(t, 2_000_000)
	st.campaigns[d.CampaignID].GoalMinor = 2_000_000 // this donation completes it

	if err := svc.HandleWebhook(context.Background(), st.p.body(d.ProviderRef, 2_000_000), st.p.sig(d.ProviderRef, 2_000_000)); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if !st.fundedCalled {
		t.Fatalf("goal was met but the campaign was not marked FUNDED")
	}
}

func TestWebhook_UnknownReferenceIsRecordedNotErrored(t *testing.T) {
	st, svc, _ := newSettledFixture(t, 2_000_000)
	body := st.p.body("civicos_never_seen", 500)
	// Must not error: a non-2xx would make Paystack retry this forever.
	if err := svc.HandleWebhook(context.Background(), body, st.p.sigFor(body)); err != nil {
		t.Fatalf("unknown reference should not error, got %v", err)
	}
	var noted bool
	for _, w := range st.webhooks {
		if w.Note != nil {
			noted = true
		}
	}
	if !noted {
		t.Fatalf("unknown reference should be left noted for reconciliation")
	}
}

// ─── Intent ─────────────────────────────────────────────────────────────

func TestCreateIntent_RefusesWhenCampaignCannotTakeMoney(t *testing.T) {
	for _, status := range []domain.CampaignStatus{
		domain.CampaignDraft, domain.CampaignPendingReview, domain.CampaignApproved,
		domain.CampaignPaused, domain.CampaignRejected, domain.CampaignArchived,
	} {
		st, svc, d := newSettledFixture(t, 1_000)
		st.campaigns[d.CampaignID].Status = status
		_, err := svc.CreateIntent(context.Background(), d.CampaignID, nil, IntentInput{
			AmountMinor: 1000, Email: "a@b.test", IdempotencyKey: uuid.NewString(),
		})
		if !isCode(err, "CAMPAIGN_NOT_ACCEPTING") {
			t.Errorf("%s should not accept donations, got %v", status, err)
		}
	}
}

// Pausing is the only governance lever left once funds settle straight to
// the org, so it has to actually stop money.
func TestCreateIntent_PausedCampaignTakesNothing(t *testing.T) {
	st, svc, d := newSettledFixture(t, 1_000)
	st.campaigns[d.CampaignID].Status = domain.CampaignPaused
	if _, err := svc.CreateIntent(context.Background(), d.CampaignID, nil, IntentInput{
		AmountMinor: 1000, Email: "a@b.test", IdempotencyKey: uuid.NewString(),
	}); !isCode(err, "CAMPAIGN_NOT_ACCEPTING") {
		t.Fatalf("paused campaign accepted a donation: %v", err)
	}
}

func TestCreateIntent_RequiresAConnectedPayoutAccount(t *testing.T) {
	st, svc, d := newSettledFixture(t, 1_000)
	st.orgs[d.OrganizationID].PSPSubaccountCode = nil
	if _, err := svc.CreateIntent(context.Background(), d.CampaignID, nil, IntentInput{
		AmountMinor: 1000, Email: "a@b.test", IdempotencyKey: uuid.NewString(),
	}); !isCode(err, "ORG_NOT_FUNDING_ELIGIBLE") {
		t.Fatalf("expected ORG_NOT_FUNDING_ELIGIBLE, got %v", err)
	}
}

func TestCreateIntent_ReusedKeyIsRejected(t *testing.T) {
	_, svc, d := newSettledFixture(t, 1_000)
	key := uuid.NewString()
	if _, err := svc.CreateIntent(context.Background(), d.CampaignID, nil, IntentInput{
		AmountMinor: 1000, Email: "a@b.test", IdempotencyKey: key,
	}); err != nil {
		t.Fatalf("first intent: %v", err)
	}
	if _, err := svc.CreateIntent(context.Background(), d.CampaignID, nil, IntentInput{
		AmountMinor: 1000, Email: "a@b.test", IdempotencyKey: key,
	}); !isCode(err, "DONATION_ALREADY_STARTED") {
		t.Fatalf("a double-tapped donate button opened a second transaction: %v", err)
	}
}

func TestCreateIntent_RecordsTheFeeRateOnTheRow(t *testing.T) {
	st, svc, d := newSettledFixture(t, 1_000)
	svc.platformFeeBps = 250
	res, err := svc.CreateIntent(context.Background(), d.CampaignID, nil, IntentInput{
		AmountMinor: 2_000_000, Email: "a@b.test", IdempotencyKey: uuid.NewString(),
	})
	if err != nil {
		t.Fatalf("intent: %v", err)
	}
	if res.PlatformFeeMinor != 50_000 || res.NetMinor != 1_950_000 {
		t.Fatalf("split wrong: %+v", res)
	}
	// The rate must be pinned to the row so a later rate change cannot
	// rewrite what this donor was told.
	var found bool
	for _, row := range st.donations {
		if row.ProviderRef == res.Reference {
			found = true
			if row.PlatformFeeBps != 250 {
				t.Fatalf("fee rate not pinned to the row: %d", row.PlatformFeeBps)
			}
		}
	}
	if !found {
		t.Fatalf("no ledger row written for the intent")
	}
}

func TestPublicDonations_HidesIdentity(t *testing.T) {
	st, svc, d := newSettledFixture(t, 1_000)
	email := "donor@example.test"
	name := "Ada"
	st.settled = []domain.Donation{
		{GrossMinor: 500, IsAnonymous: true, DonorName: &name, DonorEmail: &email, SettledAt: ptrTime()},
		{GrossMinor: 700, IsAnonymous: false, DonorName: &name, DonorEmail: &email, SettledAt: ptrTime()},
	}
	out, err := svc.PublicDonations(d.CampaignID)
	if err != nil {
		t.Fatalf("public donations: %v", err)
	}
	if out[0].DonorName != "Anonymous" {
		t.Fatalf("anonymous donor exposed as %q", out[0].DonorName)
	}
	if out[1].DonorName != "Ada" {
		t.Fatalf("named donor lost their name: %q", out[1].DonorName)
	}
}

// ─── Fixture ────────────────────────────────────────────────────────────

func ptrTime() *time.Time { t := time.Now().UTC(); return &t }

func isCode(err error, code string) bool {
	var e *AppError
	return errors.As(err, &e) && e.Code == code
}

func newSettledFixture(t *testing.T, amount int64) (*fakeStore, *Service, *domain.Donation) {
	t.Helper()
	st := newFakeStore()
	campaignID, orgID := uuid.NewString(), uuid.NewString()
	code := "ACCT_test"
	set := "x"
	st.orgs[orgID] = &domain.Organization{
		ID: orgID, Verified: true, RegistrationNumber: &set, Country: &set,
		OfficialEmail: &set, RepresentativeName: &set, BankAccountVerified: true,
		PSPSubaccountCode: &code,
	}
	st.campaigns[campaignID] = &domain.Campaign{
		ID: campaignID, OrganizationID: orgID, Status: domain.CampaignPublished,
		Currency: "NGN", GoalMinor: 100_000_000,
	}
	d := &domain.Donation{
		ID: uuid.NewString(), CampaignID: campaignID, OrganizationID: orgID,
		Currency: "NGN", GrossMinor: amount, Status: domain.DonationPending,
		Provider: "paystack", ProviderRef: "civicos_" + uuid.NewString()[:8],
		IdempotencyKey: uuid.NewString(),
	}
	st.donations[d.ID] = d
	svc := NewService(st, st.p, 0, "")
	return st, svc, d
}
