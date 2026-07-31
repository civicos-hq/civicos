package donations

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/civicos/organization-service/internal/domain"
	"github.com/civicos/organization-service/pkg/mailer"
)

// Receipts — telling a donor what happened to their money.
//
// The governing rule: **sending a receipt must never affect whether a
// donation settles.** Money moving is the fact; the email is a description
// of it. An SMTP outage, a bounced address, or a slow relay must not roll
// back a settlement, fail a webhook (Paystack would retry it), or block the
// reconciliation sweep. Every failure here is logged and swallowed.
//
// The cost of that choice is a window: the process can settle a donation and
// then die before the mail goes out. Donation.ReceiptSentAt is how that
// window is closed — reconciliation re-sends anything settled without a
// receipt, so the gap is measured in minutes rather than forever.

// ReceiptSender is the port. Mail is optional infrastructure: without it the
// service still takes donations, and this is nil.
type ReceiptSender interface {
	Send(to, subject, htmlBody, textBody string) error
}

// receiptContext is everything the template needs that the donation row does
// not carry itself.
type receiptContext struct {
	CampaignTitle    string
	CampaignSlug     string
	OrganizationName string
}

// sendReceipt emails the donor. Returns whether a receipt was actually sent,
// so the caller can record it — never an error, because there is no error
// here a caller should act on.
func (s *Service) sendReceipt(d *domain.Donation) bool {
	if s.receipts == nil {
		return false
	}
	if d.DonorEmail == nil || strings.TrimSpace(*d.DonorEmail) == "" {
		// Guest donations always carry an email (it is required to open a
		// transaction), so this means something upstream is wrong.
		log.Printf("receipts: donation=%s settled with no donor email — no receipt sent", d.ID)
		return false
	}

	ctx := s.receiptContext(d)
	settled := time.Now().UTC()
	if d.SettledAt != nil {
		settled = d.SettledAt.UTC()
	}

	subject, html, text := mailer.DonationReceiptEmail(mailer.Receipt{
		// The donor's own name, used even when the donation is anonymous:
		// anonymity is about the PUBLIC donor list, not about hiding a
		// person's gift from themselves in their own receipt.
		DonorName:        derefOr(d.DonorName, ""),
		CampaignTitle:    ctx.CampaignTitle,
		CampaignURL:      s.campaignURL(ctx.CampaignSlug),
		OrganizationName: ctx.OrganizationName,
		Reference:        d.ProviderRef,
		Currency:         d.Currency,
		GrossMinor:       d.GrossMinor,
		PlatformFeeMinor: d.PlatformFeeMinor,
		NetMinor:         d.NetMinor,
		PlatformFeeBps:   d.PlatformFeeBps,
		SettledAt:        settled,
	})

	if err := s.receipts.Send(*d.DonorEmail, subject, html, text); err != nil {
		// Logged with the donation id, not the address: an operator needs to
		// find the row, and mail logs are not the place for donor emails.
		log.Printf("receipts: send failed for donation=%s: %v", d.ID, err)
		return false
	}
	return true
}

// deliverReceipt sends and records, in that order. Recording a receipt we
// did not send would silence the retry that reconciliation is meant to
// perform.
func (s *Service) deliverReceipt(d *domain.Donation) {
	if !s.sendReceipt(d) {
		return
	}
	if err := s.repo.MarkReceiptSent(d.ID); err != nil {
		// The donor has their receipt; we just failed to write that down.
		// Reconciliation will send a duplicate later, which is a far better
		// failure than a donor with no record of their donation.
		log.Printf("receipts: sent for donation=%s but could not record it: %v", d.ID, err)
	}
}

// receiptContext resolves the campaign and organization names. Best-effort:
// a receipt with a blank campaign title is worth sending; no receipt is not.
func (s *Service) receiptContext(d *domain.Donation) receiptContext {
	var ctx receiptContext
	if c, err := s.repo.Campaign(d.CampaignID); err == nil && c != nil {
		ctx.CampaignTitle = c.Title
		ctx.CampaignSlug = c.Slug
	}
	if o, err := s.repo.Org(d.OrganizationID); err == nil && o != nil {
		ctx.OrganizationName = o.Name
	}
	if ctx.OrganizationName == "" {
		ctx.OrganizationName = "the organization"
	}
	return ctx
}

func (s *Service) campaignURL(slug string) string {
	base := strings.TrimSuffix(s.appURL, "/")
	if base == "" || slug == "" {
		return base
	}
	return fmt.Sprintf("%s/campaigns/%s", base, slug)
}

func derefOr(p *string, fallback string) string {
	if p == nil {
		return fallback
	}
	return *p
}

// ─── Backfill ───────────────────────────────────────────────────────────

// sweepReceipts sends receipts for donations that settled but never got one.
//
// This is the counterpart to settling outside a mail transaction. It catches
// both the crash-after-settle window and any period where SMTP was down, and
// it is bounded so one bad run cannot turn into a mail storm.
func (s *Service) sweepReceipts(opts ReconcileOptions, rep *ReconcileReport) {
	if s.receipts == nil {
		return
	}
	// Only chase donations old enough that the inline send has certainly
	// been attempted; anything newer is probably in flight right now.
	rows, err := s.repo.ListSettledWithoutReceipt(opts.Now.Add(-receiptRetryDelay), receiptSweepLimit)
	if err != nil {
		log.Printf("receipts: could not list unsent receipts: %v", err)
		return
	}
	for i := range rows {
		d := &rows[i]
		if s.sendReceipt(d) {
			rep.ReceiptsSent++
			if err := s.repo.MarkReceiptSent(d.ID); err != nil {
				log.Printf("receipts: sent for donation=%s but could not record it: %v", d.ID, err)
			}
		} else {
			rep.ReceiptsFailed++
		}
	}
}

const (
	// receiptRetryDelay is how long after settlement a missing receipt is
	// considered missing rather than in flight.
	receiptRetryDelay = 5 * time.Minute
	// receiptSweepLimit caps a backfill so a long outage drains gradually
	// instead of firing thousands of emails in one tick.
	receiptSweepLimit = 200
)
