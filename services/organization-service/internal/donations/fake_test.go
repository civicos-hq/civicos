package donations

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"sort"
	"time"

	"github.com/civicos/organization-service/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// fakeProvider signs with the real HMAC scheme so the signature path under
// test is the production one — only the network calls are stubbed.
type fakeProvider struct {
	secret      string
	initErr     error
	verify      map[string]TransactionStatus
	verifyErrs  map[string]error
	verifyCalls int
}

func (f *fakeProvider) Name() string { return "paystack" }

func (f *fakeProvider) sigFor(body []byte) string {
	mac := hmac.New(sha512.New, []byte(f.secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func (f *fakeProvider) body(ref string, amount int64) []byte {
	return f.bodyCurrency(ref, amount, "NGN")
}

func (f *fakeProvider) bodyCurrency(ref string, amount int64, currency string) []byte {
	return []byte(fmt.Sprintf(
		`{"event":"charge.success","data":{"id":991,"reference":%q,"status":"success","amount":%d,"currency":%q,"fees":1500,"subaccount":{"subaccount_code":"ACCT_test"}}}`,
		ref, amount, currency))
}

// bodyWithEvent varies the provider event id so a delivery gets PAST the
// event-level dedupe and reaches the already-settled check. That is the
// replay path that actually matters for receipts.
func (f *fakeProvider) bodyWithEvent(ref string, amount int64, eventID int) []byte {
	return []byte(fmt.Sprintf(
		`{"event":"charge.success","data":{"id":%d,"reference":%q,"status":"success","amount":%d,"currency":"NGN","fees":1500,"subaccount":{"subaccount_code":"ACCT_test"}}}`,
		eventID, ref, amount))
}

func (f *fakeProvider) bodyStatus(ref string, amount int64, status string) []byte {
	return []byte(fmt.Sprintf(
		`{"event":"charge.failed","data":{"id":992,"reference":%q,"status":%q,"amount":%d,"currency":"NGN","fees":0,"subaccount":{"subaccount_code":"ACCT_test"}}}`,
		ref, status, amount))
}

func (f *fakeProvider) sig(ref string, amount int64) string { return f.sigFor(f.body(ref, amount)) }

// VerifyWebhook delegates to the real Paystack implementation — the point of
// these tests is that production verification logic is what runs.
func (f *fakeProvider) VerifyWebhook(raw []byte, signature string) (WebhookEvent, error) {
	return NewPaystack(f.secret).VerifyWebhook(raw, signature)
}

func (f *fakeProvider) CreateSubaccount(context.Context, CreateSubaccountInput) (Subaccount, error) {
	return Subaccount{Code: "ACCT_test", BankName: "Test Bank", AccountLast4: "4321"}, nil
}

func (f *fakeProvider) InitializeTransaction(_ context.Context, in InitializeInput) (Initialized, error) {
	if f.initErr != nil {
		return Initialized{}, f.initErr
	}
	return Initialized{AuthorizationURL: "https://checkout.test/" + in.Reference, Reference: in.Reference}, nil
}

// VerifyTransaction answers from a per-reference script so reconciliation
// tests can pose the exact disagreements the job exists to catch.
func (f *fakeProvider) VerifyTransaction(_ context.Context, ref string) (TransactionStatus, error) {
	if err, ok := f.verifyErrs[ref]; ok {
		return TransactionStatus{}, err
	}
	if st, ok := f.verify[ref]; ok {
		f.verifyCalls++
		return st, nil
	}
	f.verifyCalls++
	return TransactionStatus{Reference: ref}, nil
}

// says scripts what the provider will report for a reference.
func (f *fakeProvider) says(ref string, st TransactionStatus) {
	if f.verify == nil {
		f.verify = map[string]TransactionStatus{}
	}
	st.Reference = ref
	f.verify[ref] = st
}

func (f *fakeProvider) failsFor(ref string, err error) {
	if f.verifyErrs == nil {
		f.verifyErrs = map[string]error{}
	}
	f.verifyErrs[ref] = err
}

type fakeStore struct {
	p            *fakeProvider
	donations    map[string]*domain.Donation
	campaigns    map[string]*domain.Campaign
	orgs         map[string]*domain.Organization
	webhooks     []*domain.WebhookEvent
	seenEvents   map[string]bool
	settled      []domain.Donation
	settleCalls  int
	writeCalls   int // every mutating store call, for the "reports, never rewrites" guard
	fundedCalled bool
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		p:          &fakeProvider{secret: "sk_test_fixture_secret"},
		donations:  map[string]*domain.Donation{},
		campaigns:  map[string]*domain.Campaign{},
		orgs:       map[string]*domain.Organization{},
		seenEvents: map[string]bool{},
	}
}

func (f *fakeStore) Create(d *domain.Donation) error {
	for _, existing := range f.donations {
		if existing.IdempotencyKey == d.IdempotencyKey || existing.ProviderRef == d.ProviderRef {
			return ErrDuplicate
		}
	}
	copied := *d
	f.donations[d.ID] = &copied
	return nil
}

func (f *fakeStore) FindByRef(_, ref string) (*domain.Donation, error) {
	for _, d := range f.donations {
		if d.ProviderRef == ref {
			return d, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeStore) FindByIdempotencyKey(key string) (*domain.Donation, error) {
	for _, d := range f.donations {
		if d.IdempotencyKey == key {
			return d, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeStore) ListSettledForCampaign(string) ([]domain.Donation, error) {
	return f.settled, nil
}

// ListStale mirrors the repository: strictly older than the cutoff, oldest
// first, capped. A fake that ignored the cutoff would let a test pass while
// the real job chased donations whose donor is still on the checkout page.
func (f *fakeStore) ListStale(status domain.DonationStatus, olderThan time.Time, limit int) ([]domain.Donation, error) {
	var out []domain.Donation
	for _, d := range f.donations {
		if d.Status == status && d.CreatedAt.Before(olderThan) {
			out = append(out, *d)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (f *fakeStore) ListSettledSince(since time.Time, limit int) ([]domain.Donation, error) {
	var out []domain.Donation
	for _, d := range f.donations {
		if d.Status == domain.DonationSettled && d.SettledAt != nil && !d.SettledAt.Before(since) {
			out = append(out, *d)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SettledAt.Before(*out[j].SettledAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (f *fakeStore) MarkReceiptSent(donationID string) error {
	f.writeCalls++
	d, ok := f.donations[donationID]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	now := time.Now().UTC()
	d.ReceiptSentAt = &now
	return nil
}

// ListSettledWithoutReceipt mirrors the repository's filter exactly. A fake
// that ignored receipt_sent_at would let a test pass while the real sweep
// emailed every donor again on every tick.
func (f *fakeStore) ListSettledWithoutReceipt(settledBefore time.Time, limit int) ([]domain.Donation, error) {
	var out []domain.Donation
	for _, d := range f.donations {
		if d.Status == domain.DonationSettled && d.ReceiptSentAt == nil &&
			d.SettledAt != nil && d.SettledAt.Before(settledBefore) {
			out = append(out, *d)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SettledAt.Before(*out[j].SettledAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (f *fakeStore) Campaign(id string) (*domain.Campaign, error) {
	if c, ok := f.campaigns[id]; ok {
		return c, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeStore) Org(id string) (*domain.Organization, error) {
	if o, ok := f.orgs[id]; ok {
		return o, nil
	}
	return nil, gorm.ErrRecordNotFound
}

// Settle mirrors the real repository's contract: idempotent, and the
// projection is SUMMED from settled rows rather than incremented.
func (f *fakeStore) Settle(id string, pspFee int64) (SettleResult, error) {
	f.settleCalls++
	f.writeCalls++
	d, ok := f.donations[id]
	if !ok {
		return SettleResult{}, gorm.ErrRecordNotFound
	}
	if d.Status == domain.DonationSettled {
		return SettleResult{AlreadySettled: true, Campaign: f.campaigns[d.CampaignID]}, nil
	}
	d.Status = domain.DonationSettled
	d.PSPFeeMinor = pspFee
	now := time.Now().UTC()
	d.SettledAt = &now

	var total int64
	var count int
	for _, row := range f.donations {
		if row.CampaignID == d.CampaignID && row.Status == domain.DonationSettled {
			total += row.GrossMinor
			count++
		}
	}
	c := f.campaigns[d.CampaignID]
	c.RaisedMinor = total
	c.DonorCount = count
	return SettleResult{Campaign: c}, nil
}

func (f *fakeStore) MarkFailed(id string, status domain.DonationStatus) error {
	f.writeCalls++
	if d, ok := f.donations[id]; ok {
		d.Status = status
	}
	return nil
}

func (f *fakeStore) SetCampaignFunded(id string) error {
	f.writeCalls++
	f.fundedCalled = true
	if c, ok := f.campaigns[id]; ok && c.Status == domain.CampaignPublished {
		c.Status = domain.CampaignFunded
	}
	return nil
}

func (f *fakeStore) RecordWebhook(e *domain.WebhookEvent) (bool, error) {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	// Dedupe on the PROVIDER's event id, mirroring the unique index in
	// Postgres. The fake previously keyed on e.ID and so never noticed that
	// a non-UUID was being written into a uuid column.
	if e.ProviderEventID != nil {
		if f.seenEvents[*e.ProviderEventID] {
			return false, nil
		}
		f.seenEvents[*e.ProviderEventID] = true
	}
	copied := *e
	f.webhooks = append(f.webhooks, &copied)
	return true, nil
}

func (f *fakeStore) MarkWebhookHandled(id string, note *string) error {
	for _, w := range f.webhooks {
		if w.ID == id {
			w.Handled = true
			w.Note = note
		}
	}
	return nil
}

func (f *fakeStore) ConnectSubaccount(orgID, provider, code, bank, last4 string) error {
	if o, ok := f.orgs[orgID]; ok {
		o.PSPProvider, o.PSPSubaccountCode = &provider, &code
		o.PSPBankName, o.PSPAccountLast4 = &bank, &last4
	}
	return nil
}

func (f *fakeProvider) ListBanks(context.Context) ([]Bank, error) {
	return []Bank{{Name: "Zenith Bank", Code: "057"}, {Name: "Access Bank", Code: "044"}}, nil
}
