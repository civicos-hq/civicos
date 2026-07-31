package donations

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/civicos/organization-service/internal/domain"
	"github.com/civicos/organization-service/internal/notifications"
	"github.com/civicos/organization-service/pkg/mailer"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Store interface {
	Create(d *domain.Donation) error
	FindByRef(provider, ref string) (*domain.Donation, error)
	FindByIdempotencyKey(key string) (*domain.Donation, error)
	ListSettledForCampaign(campaignID string) ([]domain.Donation, error)
	// Reconciliation reads. ListStale finds payments whose webhook never
	// arrived; ListSettledSince re-checks money already banked.
	ListStale(status domain.DonationStatus, olderThan time.Time, limit int) ([]domain.Donation, error)
	ListSettledSince(since time.Time, limit int) ([]domain.Donation, error)
	// Receipt bookkeeping. MarkReceiptSent records delivery;
	// ListSettledWithoutReceipt finds donors we settled but never told.
	MarkReceiptSent(donationID string) error
	ListSettledWithoutReceipt(settledBefore time.Time, limit int) ([]domain.Donation, error)
	Campaign(id string) (*domain.Campaign, error)
	Org(id string) (*domain.Organization, error)
	Settle(donationID string, pspFeeMinor int64) (SettleResult, error)
	MarkFailed(donationID string, status domain.DonationStatus) error
	SetCampaignFunded(campaignID string) error
	RecordWebhook(e *domain.WebhookEvent) (bool, error)
	MarkWebhookHandled(id string, note *string) error
	ConnectSubaccount(orgID, provider, code, bankName, last4 string) error
}

type Service struct {
	repo           Store
	provider       PaymentProvider
	platformFeeBps int64
	callbackURL    string

	// receipts is optional. Without a mailer the service still takes
	// donations and settles them — a donor who gets no email is a worse
	// experience, but a donor whose payment is refused because SMTP is
	// misconfigured is a broken product.
	receipts ReceiptSender
	// appURL is the public web origin, used to link a receipt back to the
	// campaign it funded.
	appURL string

	// Notifications are optional. Money settling must never depend on a
	// notification being deliverable.
	notifier Notifier
	audience Audience
}

// Notifier is the slice of notifications this package needs.
type Notifier interface {
	EmitMany(userIDs []string, t notifications.NotificationType, title, body string, linkURL *string)
}

// Audience answers who has a stake in a campaign.
type Audience interface {
	OrgMembers(orgID string) []string
	Donors(campaignID string) []string
	Stakeholders(campaignID, orgID string) []string
}

// WithNotifications attaches fan-out for settlement events.
func (s *Service) WithNotifications(n Notifier, a Audience) *Service {
	s.notifier = n
	s.audience = a
	return s
}

func NewService(repo Store, provider PaymentProvider, platformFeeBps int64, callbackURL string) *Service {
	return &Service{repo: repo, provider: provider, platformFeeBps: platformFeeBps, callbackURL: callbackURL}
}

// WithReceipts attaches the mailer. Separate from NewService so that mail
// stays visibly optional at the wiring site.
func (s *Service) WithReceipts(sender ReceiptSender, appURL string) *Service {
	s.receipts = sender
	s.appURL = appURL
	return s
}

// PlatformFeeBps is exposed so the public campaign page can disclose the
// rate. A donor should be able to see what actually reaches the organization
// before giving.
func (s *Service) PlatformFeeBps() int64 { return s.platformFeeBps }

func (s *Service) Enabled() bool { return s.provider != nil }

// ─── Donation intent ────────────────────────────────────────────────────

type IntentInput struct {
	AmountMinor    int64   `json:"amountMinor" binding:"required"`
	Email          string  `json:"email" binding:"required,email"`
	DonorName      *string `json:"donorName"`
	IsAnonymous    bool    `json:"isAnonymous"`
	Message        *string `json:"message"`
	IdempotencyKey string  `json:"idempotencyKey" binding:"required,min=8,max=100"`
}

type IntentResult struct {
	AuthorizationURL string `json:"authorizationUrl"`
	Reference        string `json:"reference"`
	AmountMinor      int64  `json:"amountMinor"`
	PlatformFeeMinor int64  `json:"platformFeeMinor"`
	NetMinor         int64  `json:"netMinor"`
	Currency         string `json:"currency"`
}

// CreateIntent opens a payment and records a PENDING ledger row.
//
// The row is written BEFORE the donor is sent to Paystack, deliberately. If
// we called Paystack first and then failed to persist, a donor could pay
// against a reference we have no record of — money would move with nothing
// on our side to reconcile it against. A PENDING row with no matching
// payment is harmless by comparison; it simply never settles.
func (s *Service) CreateIntent(ctx context.Context, campaignID string, donorUserID *string, in IntentInput) (*IntentResult, error) {
	if s.provider == nil {
		return nil, &AppError{Code: "DONATIONS_UNAVAILABLE", Message: "Donations are not enabled on this deployment", Status: http.StatusServiceUnavailable}
	}

	// Idempotency first: a retried request must return the original intent,
	// not open a second transaction against the same key.
	if existing, err := s.repo.FindByIdempotencyKey(in.IdempotencyKey); err == nil && existing != nil {
		return nil, &AppError{
			Code:    "DONATION_ALREADY_STARTED",
			Message: "A donation with this key is already in progress",
			Status:  http.StatusConflict,
		}
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	c, err := s.repo.Campaign(campaignID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &AppError{Code: "CAMPAIGN_NOT_FOUND", Message: "Campaign not found", Status: http.StatusNotFound}
	} else if err != nil {
		return nil, err
	}

	// Only a live campaign may take money. PAUSED is explicitly excluded —
	// pausing is the one governance lever that still works once funds settle
	// directly to the org, so it must actually stop new donations.
	if c.Status != domain.CampaignPublished && c.Status != domain.CampaignFunded {
		return nil, &AppError{
			Code:    "CAMPAIGN_NOT_ACCEPTING",
			Message: "This campaign is not accepting donations",
			Status:  http.StatusConflict,
		}
	}

	org, err := s.repo.Org(c.OrganizationID)
	if err != nil {
		return nil, err
	}
	if ok, missing := org.FundingEligible(); !ok {
		return nil, &AppError{
			Code:    "ORG_NOT_FUNDING_ELIGIBLE",
			Message: "This organization cannot receive donations: " + strings.Join(missing, ", "),
			Status:  http.StatusConflict,
		}
	}

	split, err := ComputeSplit(in.AmountMinor, s.platformFeeBps)
	if err != nil {
		return nil, &AppError{Code: "INVALID_AMOUNT", Message: err.Error(), Status: http.StatusBadRequest}
	}
	if !split.Valid() {
		return nil, &AppError{Code: "INVALID_AMOUNT", Message: "Could not compute a valid split", Status: http.StatusBadRequest}
	}

	// Our own reference, not Paystack's. Prefixed so it is recognisable in
	// their dashboard during a dispute.
	ref := "civicos_" + strings.ReplaceAll(uuid.NewString(), "-", "")

	d := &domain.Donation{
		ID:               uuid.New().String(),
		CampaignID:       c.ID,
		OrganizationID:   c.OrganizationID,
		Currency:         c.Currency,
		GrossMinor:       split.GrossMinor,
		PlatformFeeMinor: split.PlatformFeeMinor,
		NetMinor:         split.NetMinor,
		PlatformFeeBps:   s.platformFeeBps,
		Status:           domain.DonationPending,
		Provider:         s.provider.Name(),
		ProviderRef:      ref,
		IdempotencyKey:   in.IdempotencyKey,
		DonorUserID:      donorUserID,
		DonorName:        in.DonorName,
		IsAnonymous:      in.IsAnonymous,
		DonorEmail:       &in.Email,
		Message:          in.Message,
	}
	if err := s.repo.Create(d); err != nil {
		if errors.Is(err, ErrDuplicate) {
			return nil, &AppError{Code: "DONATION_ALREADY_STARTED", Message: "A donation with this key is already in progress", Status: http.StatusConflict}
		}
		return nil, err
	}

	init, err := s.provider.InitializeTransaction(ctx, InitializeInput{
		Reference:      ref,
		AmountMinor:    split.GrossMinor,
		Currency:       c.Currency,
		Email:          in.Email,
		SubaccountCode: *org.PSPSubaccountCode,
		PlatformFeeBps: s.platformFeeBps,
		CallbackURL:    s.callbackURL,
		Metadata: map[string]string{
			"campaignId":     c.ID,
			"organizationId": c.OrganizationID,
		},
	})
	if err != nil {
		// The PENDING row stays. It is a record that we tried, and it will
		// never settle — cleaner than deleting evidence of a failed attempt.
		_ = s.repo.MarkFailed(d.ID, domain.DonationFailed)
		return nil, &AppError{Code: "PROVIDER_ERROR", Message: "Could not start the payment. Please try again.", Status: http.StatusBadGateway}
	}

	return &IntentResult{
		AuthorizationURL: init.AuthorizationURL,
		Reference:        ref,
		AmountMinor:      split.GrossMinor,
		PlatformFeeMinor: split.PlatformFeeMinor,
		NetMinor:         split.NetMinor,
		Currency:         c.Currency,
	}, nil
}

// ─── Webhook ────────────────────────────────────────────────────────────

// HandleWebhook verifies, records and applies a provider callback.
//
// The order is deliberate: verify → record → apply. Recording before
// applying means a delivery that later fails to apply is still on file, and
// an unverified delivery is recorded too — repeated failures are how we find
// out someone is probing the endpoint.
func (s *Service) HandleWebhook(ctx context.Context, rawBody []byte, signature string) error {
	if s.provider == nil {
		return &AppError{Code: "DONATIONS_UNAVAILABLE", Message: "Donations are not enabled", Status: http.StatusServiceUnavailable}
	}

	ev, verifyErr := s.provider.VerifyWebhook(rawBody, signature)
	if verifyErr != nil {
		// Store the rejection, then refuse. Never act on it.
		_, _ = s.repo.RecordWebhook(&domain.WebhookEvent{
			ID:        uuid.New().String(),
			Provider:  s.provider.Name(),
			EventType: "unverified",
			Verified:  false,
			Payload:   string(truncate(rawBody)),
		})
		return &AppError{Code: "INVALID_SIGNATURE", Message: "Signature verification failed", Status: http.StatusUnauthorized}
	}

	// First dedupe layer, on the provider's own event id.
	rowID := uuid.New().String()
	providerEventID := ev.ID
	fresh, err := s.repo.RecordWebhook(&domain.WebhookEvent{
		ID:              rowID,
		Provider:        s.provider.Name(),
		ProviderEventID: &providerEventID,
		EventType:       ev.Type,
		ProviderRef:     ev.Reference,
		Verified:        true,
		Payload:         string(truncate(ev.Raw)),
	})
	if err != nil {
		return err
	}
	if !fresh {
		return nil // already processed; replays are a no-op
	}

	d, err := s.repo.FindByRef(s.provider.Name(), ev.Reference)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// A payment we have no record of. Not an error to the provider —
		// returning non-2xx would make Paystack retry forever — but it is
		// left unhandled so reconciliation surfaces it.
		note := "no matching donation for reference"
		_ = s.repo.MarkWebhookHandled(rowID, &note)
		return nil
	} else if err != nil {
		return err
	}

	_, note, err := s.applyStatus(d, ev.Status)
	if err != nil {
		return err
	}
	_ = s.repo.MarkWebhookHandled(rowID, note)
	return nil
}

// applyStatus moves a donation to whatever terminal state the provider
// reports, and is the ONLY place that does so.
//
// Both the webhook and the reconciliation sweep come through here. That is
// the entire point: if reconciliation carried its own notion of what
// settling means, the job meant to detect drift would become a source of it.
//
// The note is returned rather than written, so each caller can record it
// where it belongs — on the webhook row, or in the reconciliation report.
func (s *Service) applyStatus(d *domain.Donation, st TransactionStatus) (ApplyOutcome, *string, error) {
	switch {
	case st.Failed:
		if err := s.repo.MarkFailed(d.ID, domain.DonationFailed); err != nil {
			return OutcomeError, nil, err
		}
		return OutcomeFailed, nil, nil

	case st.Abandoned:
		// The donor opened checkout and walked away. Terminal, and worth
		// recording as its own thing — otherwise the row sits PENDING
		// forever and looks like a payment we lost track of.
		if err := s.repo.MarkFailed(d.ID, domain.DonationAbandoned); err != nil {
			return OutcomeError, nil, err
		}
		return OutcomeAbandoned, nil, nil

	case !st.Succeeded:
		note := "event carried no terminal status"
		return OutcomeStillPending, &note, nil
	}

	// Trust the signature for authenticity, but still check the contents
	// against what we asked for. A signed message is proof it came from
	// Paystack, not proof it describes the transaction we opened.
	if err := s.reconcileAgainstIntent(d, st); err != nil {
		note := err.Error()
		// Deliberately NOT settled. A mismatch is for a human to look at.
		return OutcomeMismatch, &note, nil
	}

	res, err := s.repo.Settle(d.ID, st.PSPFeeMinor)
	if err != nil {
		return OutcomeError, nil, err
	}
	if res.AlreadySettled {
		return OutcomeAlreadySettled, nil, nil
	}
	if res.Campaign != nil && res.Campaign.RaisedMinor >= res.Campaign.GoalMinor {
		// Goal reached is ledger truth, asserted by the platform alone —
		// see campaigns.ActorSystem.
		_ = s.repo.SetCampaignFunded(res.Campaign.ID)
	}

	// Receipt on FRESH settlement only. A replayed webhook lands on
	// AlreadySettled above and returns before reaching here, so a donor
	// cannot be emailed twice for one donation. Deliberately outside the
	// settlement transaction, and deliberately unable to fail this call —
	// see receipt.go.
	s.deliverReceipt(d)
	s.announceSettlement(d, res.Campaign)

	return OutcomeSettled, nil, nil
}

// reconcileAgainstIntent refuses to settle a donation whose reported details
// do not match what was opened. Any of these means something is wrong, and
// quietly banking it would corrupt the ledger we rely on for reconciliation.
func (s *Service) reconcileAgainstIntent(d *domain.Donation, st TransactionStatus) error {
	if st.AmountMinor != d.GrossMinor {
		return fmt.Errorf("amount mismatch: opened %d, provider reported %d", d.GrossMinor, st.AmountMinor)
	}
	if st.Currency != "" && !strings.EqualFold(st.Currency, d.Currency) {
		return fmt.Errorf("currency mismatch: opened %s, provider reported %s", d.Currency, st.Currency)
	}
	return nil
}

// announceSettlement tells the organization money arrived, and tells
// everyone when the goal is met.
//
// Reached from the same fresh-settlement path as the receipt, so a donation
// recovered by reconciliation days later still notifies — the money moved,
// and when we found out does not change who deserves to hear about it.
//
// Never returns an error: the donation is already banked.
func (s *Service) announceSettlement(d *domain.Donation, c *domain.Campaign) {
	if s.notifier == nil || s.audience == nil || c == nil {
		return
	}
	amount := mailer.FormatMoney(d.GrossMinor, d.Currency)
	link := "/campaigns/" + c.Slug

	// The organization needs to know money arrived. Donors do not need a
	// notification about their own donation — they already have the receipt,
	// and telling every previous donor about every new one would make the
	// tray unusable on a busy campaign.
	s.notifier.EmitMany(s.audience.OrgMembers(c.OrganizationID),
		notifications.TypeDonationReceived,
		"Donation received: "+amount,
		amount+" was donated to "+c.Title+".",
		&link)

	// Goal reached is ledger truth and is announced once, at the crossing.
	// Guarding on AlreadySettled upstream is what keeps a replayed webhook
	// from re-announcing it.
	if c.GoalMinor > 0 && c.RaisedMinor >= c.GoalMinor {
		s.notifier.EmitMany(s.audience.Stakeholders(c.ID, c.OrganizationID),
			notifications.TypeFundingGoalReached,
			"Goal reached: "+c.Title,
			c.Title+" has reached its funding goal of "+mailer.FormatMoney(c.GoalMinor, c.Currency)+".",
			&link)
	}
}

func truncate(b []byte) []byte {
	const max = 16 << 10
	if len(b) > max {
		return b[:max]
	}
	return b
}

// ─── Public reads ───────────────────────────────────────────────────────

// PublicDonation is the anonymised donor list. No email, no user id, and no
// name at all when the donor asked to be anonymous.
type PublicDonation struct {
	DonorName   string `json:"donorName"`
	AmountMinor int64  `json:"amountMinor"`
	Message     string `json:"message,omitempty"`
	SettledAt   string `json:"settledAt"`
}

func (s *Service) PublicDonations(campaignID string) ([]PublicDonation, error) {
	rows, err := s.repo.ListSettledForCampaign(campaignID)
	if err != nil {
		return nil, err
	}
	out := make([]PublicDonation, 0, len(rows))
	for _, d := range rows {
		name := "Anonymous"
		if !d.IsAnonymous && d.DonorName != nil && strings.TrimSpace(*d.DonorName) != "" {
			name = *d.DonorName
		}
		msg := ""
		if d.Message != nil {
			msg = *d.Message
		}
		settled := ""
		if d.SettledAt != nil {
			settled = d.SettledAt.UTC().Format("2006-01-02T15:04:05Z")
		}
		out = append(out, PublicDonation{
			DonorName:   name,
			AmountMinor: d.GrossMinor,
			Message:     msg,
			SettledAt:   settled,
		})
	}
	return out, nil
}

// ─── Sub-account connection ─────────────────────────────────────────────

type ConnectInput struct {
	BankCode      string `json:"bankCode" binding:"required"`
	AccountNumber string `json:"accountNumber" binding:"required,min=6,max=20"`
	BusinessName  string `json:"businessName" binding:"required,min=2"`
	ContactEmail  string `json:"contactEmail" binding:"omitempty,email"`
}

// ConnectSubaccount registers the organization's payout destination with
// Paystack and stores only the returned code.
//
// The account number passes through this function and is never persisted —
// see Organization.PSPSubaccountCode.
func (s *Service) ConnectSubaccount(ctx context.Context, orgID string, in ConnectInput) (*domain.Organization, error) {
	if s.provider == nil {
		return nil, &AppError{Code: "DONATIONS_UNAVAILABLE", Message: "Donations are not enabled on this deployment", Status: http.StatusServiceUnavailable}
	}
	org, err := s.repo.Org(orgID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &AppError{Code: "ORG_NOT_FOUND", Message: "Organization not found", Status: http.StatusNotFound}
	} else if err != nil {
		return nil, err
	}

	sub, err := s.provider.CreateSubaccount(ctx, CreateSubaccountInput{
		BusinessName:        in.BusinessName,
		BankCode:            in.BankCode,
		AccountNumber:       in.AccountNumber,
		PlatformFeeBps:      s.platformFeeBps,
		PrimaryContactEmail: in.ContactEmail,
	})
	if err != nil {
		return nil, &AppError{
			Code:    "PSP_SUBACCOUNT_FAILED",
			Message: "Could not connect that account. Check the bank and account number.",
			Status:  http.StatusBadGateway,
		}
	}
	if err := s.repo.ConnectSubaccount(org.ID, s.provider.Name(), sub.Code, sub.BankName, sub.AccountLast4); err != nil {
		return nil, err
	}
	return s.repo.Org(org.ID)
}

type AppError struct {
	Code    string
	Message string
	Status  int
}

func (e *AppError) Error() string { return e.Message }
