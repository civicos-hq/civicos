package donations

import (
	"context"
	"errors"
)

// PaymentProvider is the port every payment rail sits behind.
//
// It exists so the ledger, the intent flow and the webhook handler never
// import a vendor SDK. Paystack is the only implementation today; the spec
// also names a LinkiSwap crypto rail (Phase 6, still without an API spec),
// and a second fiat provider would arrive the same way.
//
// Everything here is in integer minor units. No implementation may accept or
// return a float.
type PaymentProvider interface {
	// Name identifies the provider in ledger rows ("paystack").
	Name() string

	// CreateSubaccount registers an organization's settlement account and
	// returns the provider's code for it.
	//
	// This is the ONLY point at which CivicOS handles a bank account number.
	// The code comes back, the account number is not persisted — see the
	// note on Organization.PSPSubaccountCode.
	CreateSubaccount(ctx context.Context, in CreateSubaccountInput) (Subaccount, error)

	// InitializeTransaction opens a payment and returns somewhere to send
	// the donor. The provider splits to the org's sub-account; CivicOS's
	// share is expressed as the platform's transaction charge.
	InitializeTransaction(ctx context.Context, in InitializeInput) (Initialized, error)

	// VerifyWebhook authenticates a raw callback body. It must fail closed:
	// any doubt about authenticity returns an error rather than an event,
	// because this endpoint is unauthenticated by necessity and is the one
	// place an attacker can try to fabricate money.
	VerifyWebhook(rawBody []byte, signature string) (WebhookEvent, error)

	// VerifyTransaction re-reads a transaction from the provider. The
	// webhook says what happened; this confirms it independently, which is
	// what reconciliation and any manual repair are built on.
	VerifyTransaction(ctx context.Context, reference string) (TransactionStatus, error)
}

type CreateSubaccountInput struct {
	BusinessName  string
	BankCode      string
	AccountNumber string
	// PlatformFeeBps is the platform's cut, forwarded so the provider can
	// apply the split itself rather than CivicOS handling the money.
	PlatformFeeBps int64
	// PrimaryContactEmail receives provider-side settlement notices.
	PrimaryContactEmail string
}

type Subaccount struct {
	Code         string
	BankName     string
	AccountLast4 string
}

type InitializeInput struct {
	Reference      string
	AmountMinor    int64
	Currency       string
	Email          string
	SubaccountCode string
	PlatformFeeBps int64
	CallbackURL    string
	Metadata       map[string]string
}

type Initialized struct {
	// AuthorizationURL is where the donor completes payment.
	AuthorizationURL string
	Reference        string
}

// WebhookEvent is the provider-agnostic shape of a verified callback.
type WebhookEvent struct {
	// ID is the provider's own event identifier where one exists; it is the
	// dedupe key for replayed deliveries.
	ID        string
	Type      string
	Reference string
	Status    TransactionStatus
	Raw       []byte
}

// TransactionStatus is what the provider says happened. Amounts are echoed
// back so the handler can check them against what was requested rather than
// trusting the intent — a mismatch means something is wrong and must not be
// silently accepted.
type TransactionStatus struct {
	Reference string
	Succeeded bool
	Failed    bool
	// Abandoned means the donor opened checkout and walked away. Terminal,
	// but distinct from Failed: nothing went wrong, the person simply did
	// not pay. Conflating the two makes the funnel unreadable — "how many
	// donations failed" would silently include everyone who changed their
	// mind.
	Abandoned   bool
	AmountMinor int64
	Currency    string
	// PSPFeeMinor is the provider's own charge, only knowable after the fact.
	PSPFeeMinor int64
	// SubaccountCode confirms which org was actually credited.
	SubaccountCode string
	PaidAt         string
}

var (
	ErrProviderUnavailable = errors.New("payment provider is not configured")
	ErrBadSignature        = errors.New("webhook signature verification failed")
	ErrUnexpectedPayload   = errors.New("webhook payload could not be understood")
)
