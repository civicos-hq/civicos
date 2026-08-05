package donations

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/civicos/organization-service/internal/domain"
	"github.com/civicos/organization-service/pkg/mailer"
)

// fakeMailer records what would have been sent.
type fakeMailer struct {
	sent []sentMail
	err  error
}

type sentMail struct{ to, subject, html, text string }

func (m *fakeMailer) Send(to, subject, html, text string) error {
	if m.err != nil {
		return m.err
	}
	m.sent = append(m.sent, sentMail{to, subject, html, text})
	return nil
}

// withMail attaches a fake mailer and names the campaign/org, so the
// receipt has something real to render.
func withMail(t *testing.T, st *fakeStore, svc *Service, d *domain.Donation) *fakeMailer {
	t.Helper()
	m := &fakeMailer{}
	svc.WithReceipts(m, "https://civicos.ng")
	st.campaigns[d.CampaignID].Title = "Flood relief for Sabon Gari"
	st.campaigns[d.CampaignID].Slug = "flood-relief-sabon-gari"
	st.orgs[d.OrganizationID].Name = "Zaria Relief Trust"
	email := "donor@example.com"
	d.DonorEmail = &email
	name := "Ada"
	d.DonorName = &name
	return m
}

func settleViaWebhook(t *testing.T, st *fakeStore, svc *Service, d *domain.Donation, amount int64) {
	t.Helper()
	body := st.p.body(d.ProviderRef, amount)
	if err := svc.HandleWebhook(context.Background(), body, st.p.sigFor(body)); err != nil {
		t.Fatalf("handle webhook: %v", err)
	}
}

// ─── Sending ────────────────────────────────────────────────────────────

func TestReceipt_SentOnSettlement(t *testing.T) {
	st, svc, d := newSettledFixture(t, 250_000)
	m := withMail(t, st, svc, d)

	settleViaWebhook(t, st, svc, d, 250_000)

	if len(m.sent) != 1 {
		t.Fatalf("sent %d receipts, want 1", len(m.sent))
	}
	if m.sent[0].to != "donor@example.com" {
		t.Fatalf("sent to %q", m.sent[0].to)
	}
	if st.donations[d.ID].ReceiptSentAt == nil {
		t.Fatal("ReceiptSentAt not recorded — the backfill sweep would send a duplicate")
	}
}

// A replayed webhook must not email the donor a second time. Paystack
// retries deliveries as a matter of course.
func TestReceipt_ReplayedWebhookDoesNotSendTwice(t *testing.T) {
	st, svc, d := newSettledFixture(t, 250_000)
	m := withMail(t, st, svc, d)

	settleViaWebhook(t, st, svc, d, 250_000)
	// Same reference, a fresh event id, so it gets past the event dedupe and
	// reaches the already-settled check — the case that actually matters.
	body := st.p.bodyWithEvent(d.ProviderRef, 250_000, 998877)
	if err := svc.HandleWebhook(context.Background(), body, st.p.sigFor(body)); err != nil {
		t.Fatalf("replay: %v", err)
	}

	if len(m.sent) != 1 {
		t.Fatalf("sent %d receipts after a replay, want 1", len(m.sent))
	}
}

// The governing rule: money moving is the fact, the email is a description
// of it. An SMTP outage must never cost a donation.
func TestReceipt_MailFailureDoesNotBlockSettlement(t *testing.T) {
	st, svc, d := newSettledFixture(t, 400_000)
	m := withMail(t, st, svc, d)
	m.err = errors.New("smtp: connection refused")

	settleViaWebhook(t, st, svc, d, 400_000)

	if got := st.donations[d.ID].Status; got != domain.DonationSettled {
		t.Fatalf("status = %s — a mail failure rolled back a settlement", got)
	}
	if got := st.campaigns[d.CampaignID].RaisedMinor; got != 400_000 {
		t.Fatalf("campaign total = %d — a mail failure cost the campaign its money", got)
	}
	if st.donations[d.ID].ReceiptSentAt != nil {
		t.Fatal("ReceiptSentAt was recorded despite the send failing — the backfill would never retry")
	}
}

// No mailer configured at all: still settles, still no panic.
func TestReceipt_NoMailerConfiguredStillSettles(t *testing.T) {
	st, svc, d := newSettledFixture(t, 100_000)
	settleViaWebhook(t, st, svc, d, 100_000)

	if got := st.donations[d.ID].Status; got != domain.DonationSettled {
		t.Fatalf("status = %s, want SETTLED with no mailer wired", got)
	}
}

// Anonymity is about the PUBLIC donor list, not about hiding a person's own
// gift from them in their own receipt.
func TestReceipt_AnonymousDonorStillGetsTheirReceipt(t *testing.T) {
	st, svc, d := newSettledFixture(t, 300_000)
	m := withMail(t, st, svc, d)
	d.IsAnonymous = true

	settleViaWebhook(t, st, svc, d, 300_000)

	if len(m.sent) != 1 {
		t.Fatalf("an anonymous donor got %d receipts, want 1", len(m.sent))
	}
	if !strings.Contains(m.sent[0].text, "Ada") {
		t.Fatal("the donor's own name should appear in their own receipt")
	}
}

// ─── Backfill ───────────────────────────────────────────────────────────

// The window between settling and mailing: the process can die in it. This
// is what closes it.
func TestReceipt_ReconciliationBackfillsAMissedReceipt(t *testing.T) {
	st, svc, d := newSettledFixture(t, 500_000)
	m := withMail(t, st, svc, d)
	m.err = errors.New("smtp down")

	settleViaWebhook(t, st, svc, d, 500_000)
	if st.donations[d.ID].ReceiptSentAt != nil {
		t.Fatal("precondition: receipt should not be marked sent")
	}

	// SMTP recovers, and the settlement is now old enough to chase.
	m.err = nil
	old := time.Now().UTC().Add(-time.Hour)
	st.donations[d.ID].SettledAt = &old

	rep := run(t, svc)

	if rep.ReceiptsSent != 1 {
		t.Fatalf("ReceiptsSent = %d, want 1", rep.ReceiptsSent)
	}
	if len(m.sent) != 1 {
		t.Fatalf("backfill sent %d receipts, want 1", len(m.sent))
	}
	if st.donations[d.ID].ReceiptSentAt == nil {
		t.Fatal("backfill did not record the send — the next sweep would duplicate it")
	}
}

// A donation that settled moments ago is probably mid-send. Chasing it would
// race the inline send and email the donor twice.
func TestReceipt_BackfillLeavesFreshSettlementsAlone(t *testing.T) {
	st, svc, d := newSettledFixture(t, 200_000)
	m := withMail(t, st, svc, d)
	m.err = errors.New("smtp down")
	settleViaWebhook(t, st, svc, d, 200_000)
	m.err = nil

	rep := run(t, svc) // SettledAt is "now"

	if rep.ReceiptsSent != 0 {
		t.Fatalf("ReceiptsSent = %d — the backfill raced the inline send", rep.ReceiptsSent)
	}
}

// Once recorded, a receipt is never sent again by the sweep.
func TestReceipt_BackfillIsIdempotent(t *testing.T) {
	st, svc, d := newSettledFixture(t, 500_000)
	m := withMail(t, st, svc, d)
	m.err = errors.New("smtp down")
	settleViaWebhook(t, st, svc, d, 500_000)
	m.err = nil
	old := time.Now().UTC().Add(-time.Hour)
	st.donations[d.ID].SettledAt = &old

	run(t, svc)
	run(t, svc)

	if len(m.sent) != 1 {
		t.Fatalf("donor received %d receipts across two sweeps, want 1", len(m.sent))
	}
}

// ─── Content ────────────────────────────────────────────────────────────

func TestReceipt_DisclosesTheSplitAndIsNotATaxReceipt(t *testing.T) {
	st, svc, d := newSettledFixture(t, 250_000)
	d.PlatformFeeMinor = 6_250
	d.NetMinor = 243_750
	d.PlatformFeeBps = 250
	m := withMail(t, st, svc, d)

	settleViaWebhook(t, st, svc, d, 250_000)

	body := m.sent[0].text
	for _, want := range []string{
		"₦2,500.00", // gross
		"₦62.50",    // CivicOS platform fee
		"₦15.00",    // Paystack's fee, borne by the organization
		"₦2,422.50", // what actually reached the org: 2500 - 62.50 - 15
		"Paystack processing fee",
		"2.5%", // the rate
		"Zaria Relief Trust",
		"Flood relief for Sabon Gari",
		d.ProviderRef,
		"https://civicos.ng/campaigns/flood-relief-sabon-gari",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("receipt is missing %q\n---\n%s", want, body)
		}
	}

	// The three deductions must reconcile against the gross. A receipt whose
	// own figures do not add up is worse than no receipt — this is the one
	// document where a donor is asked to trust our arithmetic.
	{
		gross, plat, psp := int64(250_000), int64(6_250), int64(1_500)
		reached := gross - plat - psp
		if !strings.Contains(body, mailer.FormatMoney(reached, "NGN")) {
			t.Errorf("receipt does not show gross-platform-psp = %s\n---\n%s",
				mailer.FormatMoney(reached, "NGN"), body)
		}
	}

	// CivicOS is not the merchant of record. Implying this can be filed for
	// tax relief would be a claim made on another entity's behalf.
	if !strings.Contains(body, "not a tax") {
		t.Error("the receipt must say it is not a tax receipt")
	}

	if !strings.Contains(m.sent[0].subject, "Zaria Relief Trust") {
		t.Errorf("subject should name the recipient, got %q", m.sent[0].subject)
	}
}

// A campaign title is user-supplied and this template is assembled with
// Sprintf, not html/template.
func TestReceipt_EscapesUserSuppliedText(t *testing.T) {
	st, svc, d := newSettledFixture(t, 100_000)
	m := withMail(t, st, svc, d)
	st.campaigns[d.CampaignID].Title = `Flood <script>alert(1)</script> relief`

	settleViaWebhook(t, st, svc, d, 100_000)

	if strings.Contains(m.sent[0].html, "<script>") {
		t.Fatalf("unescaped user text reached the HTML body")
	}
	if !strings.Contains(m.sent[0].html, "&lt;script&gt;") {
		t.Fatalf("expected the title to be escaped, not dropped")
	}
}

// The receipt must show ₦62.50, not ₦63. A donor who adds up a rounded
// receipt gets a different answer than the ledger holds, and a receipt whose
// arithmetic does not add up is worse than no receipt.
func TestFormatMoney_KeepsKoboSoTheReceiptAddsUp(t *testing.T) {
	cases := []struct {
		minor int64
		want  string
	}{
		{6_250, "₦62.50"},
		{250_000, "₦2,500.00"},
		{243_750, "₦2,437.50"},
		{0, "₦0.00"},
		{5, "₦0.05"},
		{100_000_000, "₦1,000,000.00"},
		{-6_250, "-₦62.50"},
	}
	for _, c := range cases {
		if got := mailer.FormatMoney(c.minor, "NGN"); got != c.want {
			t.Errorf("FormatMoney(%d) = %q, want %q", c.minor, got, c.want)
		}
	}

	// Gross must equal fee plus net, as rendered — not just as stored.
	gross, fee, net := int64(250_000), int64(6_250), int64(243_750)
	if fee+net != gross {
		t.Fatalf("fixture is wrong: %d + %d != %d", fee, net, gross)
	}
	if mailer.FormatMoney(gross, "NGN") != "₦2,500.00" ||
		mailer.FormatMoney(fee, "NGN") != "₦62.50" ||
		mailer.FormatMoney(net, "NGN") != "₦2,437.50" {
		t.Fatal("the rendered figures do not reconcile")
	}
}

// The money column must line up whatever the organization is called. This is
// the one document where a donor is asked to trust our arithmetic, and a
// staggering column undermines that before they read a figure.
func TestReceipt_MoneyColumnAlignsForAnyOrgName(t *testing.T) {
	for _, org := range []string{"Hope", "Zaria Relief Trust", "Ikeja Community Health Initiative"} {
		_, _, text := mailer.DonationReceiptEmail(mailer.Receipt{
			DonorName: "Ada", OrganizationName: org, Currency: "NGN",
			GrossMinor: 250_000, PlatformFeeMinor: 6_250, NetMinor: 243_750, PlatformFeeBps: 250,
			SettledAt: time.Now().UTC(),
		})

		var ends []int
		for _, line := range strings.Split(text, "\n") {
			if !strings.HasPrefix(line, "  ") {
				continue
			}
			for _, amt := range []string{"₦2,500.00", "-₦62.50", "₦2,437.50"} {
				if strings.HasSuffix(line, amt) {
					ends = append(ends, len([]rune(line)))
				}
			}
		}
		if len(ends) != 3 {
			t.Fatalf("%s: found %d split rows, want 3", org, len(ends))
		}
		for _, e := range ends {
			if e != ends[0] {
				t.Errorf("%s: money column ragged — row widths %v", org, ends)
				break
			}
		}
	}
}
