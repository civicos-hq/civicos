package donations

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Paystack implements PaymentProvider.
//
// Chosen for Phase 3 because it is Nigeria-native and its Subaccounts API
// matches the merchant-of-record decision directly: CivicOS creates a
// sub-account per organization, Paystack splits each transaction and settles
// the organization's share to their own bank account. CivicOS never takes
// custody.
type Paystack struct {
	secretKey string
	baseURL   string
	http      *http.Client
}

const paystackAPI = "https://api.paystack.co"

func NewPaystack(secretKey string) *Paystack {
	return &Paystack{
		secretKey: secretKey,
		baseURL:   paystackAPI,
		// A bounded timeout matters here: a hung provider call must not pin
		// a request goroutine holding a database transaction open.
		http: &http.Client{Timeout: 20 * time.Second},
	}
}

func (p *Paystack) Name() string { return "paystack" }

// ─── Webhook verification ───────────────────────────────────────────────

// VerifyWebhook authenticates a callback using Paystack's scheme: HMAC
// SHA-512 of the raw request body, keyed with the secret key, hex-encoded in
// the x-paystack-signature header.
//
// Three things this must get right, because it is the only unauthenticated
// write path in the service:
//
//  1. It hashes the RAW body. Re-marshalling parsed JSON would change
//     whitespace and key order and never match.
//  2. It compares with hmac.Equal, not ==. A byte-wise early-exit compare
//     leaks timing information an attacker can use to forge a signature.
//  3. It fails closed. No key configured means no verification is possible,
//     which is a refusal, not a pass.
func (p *Paystack) VerifyWebhook(rawBody []byte, signature string) (WebhookEvent, error) {
	if p.secretKey == "" {
		return WebhookEvent{}, ErrProviderUnavailable
	}
	if signature == "" {
		return WebhookEvent{}, ErrBadSignature
	}

	mac := hmac.New(sha512.New, []byte(p.secretKey))
	mac.Write(rawBody)
	expected := hex.EncodeToString(mac.Sum(nil))

	// Both sides lowercased before the constant-time compare — Paystack
	// sends lowercase hex, but a case difference must not read as tampering.
	if !hmac.Equal([]byte(strings.ToLower(signature)), []byte(expected)) {
		return WebhookEvent{}, ErrBadSignature
	}

	var body struct {
		Event string `json:"event"`
		Data  struct {
			ID         int64  `json:"id"`
			Reference  string `json:"reference"`
			Status     string `json:"status"`
			Amount     int64  `json:"amount"`
			Currency   string `json:"currency"`
			Fees       int64  `json:"fees"`
			PaidAt     string `json:"paid_at"`
			Subaccount struct {
				SubaccountCode string `json:"subaccount_code"`
			} `json:"subaccount"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rawBody, &body); err != nil {
		return WebhookEvent{}, ErrUnexpectedPayload
	}
	if body.Data.Reference == "" {
		return WebhookEvent{}, ErrUnexpectedPayload
	}

	return WebhookEvent{
		// Paystack's numeric transaction id, scoped by event type, is the
		// dedupe key for repeated deliveries of the same occurrence.
		ID:        fmt.Sprintf("%s:%d", body.Event, body.Data.ID),
		Type:      body.Event,
		Reference: body.Data.Reference,
		Raw:       rawBody,
		Status: TransactionStatus{
			Reference:      body.Data.Reference,
			Succeeded:      body.Data.Status == "success",
			Failed:         body.Data.Status == "failed" || body.Data.Status == "reversed",
			Abandoned:      body.Data.Status == "abandoned",
			AmountMinor:    body.Data.Amount,
			Currency:       body.Data.Currency,
			PSPFeeMinor:    body.Data.Fees,
			SubaccountCode: body.Data.Subaccount.SubaccountCode,
			PaidAt:         body.Data.PaidAt,
		},
	}, nil
}

// ─── API calls ──────────────────────────────────────────────────────────

func (p *Paystack) CreateSubaccount(ctx context.Context, in CreateSubaccountInput) (Subaccount, error) {
	// Paystack expresses the sub-account's share as the percentage the
	// SUBACCOUNT keeps... but percentage_charge is the platform's cut. Send
	// basis points converted to a percentage float only at this boundary —
	// the wire format requires it; nothing internal does.
	payload := map[string]any{
		"business_name":     in.BusinessName,
		"settlement_bank":   in.BankCode,
		"account_number":    in.AccountNumber,
		"percentage_charge": float64(in.PlatformFeeBps) / 100.0,
	}
	if in.PrimaryContactEmail != "" {
		payload["primary_contact_email"] = in.PrimaryContactEmail
	}

	var res struct {
		Status  bool   `json:"status"`
		Message string `json:"message"`
		Data    struct {
			SubaccountCode string `json:"subaccount_code"`
			AccountNumber  string `json:"account_number"`
			SettlementBank string `json:"settlement_bank"`
		} `json:"data"`
	}
	if err := p.call(ctx, http.MethodPost, "/subaccount", payload, &res); err != nil {
		return Subaccount{}, err
	}
	if !res.Status || res.Data.SubaccountCode == "" {
		return Subaccount{}, fmt.Errorf("paystack refused the sub-account: %s", res.Message)
	}

	last4 := res.Data.AccountNumber
	if len(last4) > 4 {
		last4 = last4[len(last4)-4:]
	}
	return Subaccount{
		Code:         res.Data.SubaccountCode,
		BankName:     res.Data.SettlementBank,
		AccountLast4: last4,
	}, nil
}

func (p *Paystack) InitializeTransaction(ctx context.Context, in InitializeInput) (Initialized, error) {
	payload := map[string]any{
		"reference": in.Reference,
		"amount":    in.AmountMinor,
		"currency":  in.Currency,
		"email":     in.Email,
		// subaccount + bearer: the sub-account receives the money and
		// Paystack takes its charge from CivicOS's share, so the
		// organization's net is not eroded by the transaction fee twice.
		"subaccount": in.SubaccountCode,
		"bearer":     "account",
	}
	if in.CallbackURL != "" {
		payload["callback_url"] = in.CallbackURL
	}
	if len(in.Metadata) > 0 {
		payload["metadata"] = in.Metadata
	}

	var res struct {
		Status  bool   `json:"status"`
		Message string `json:"message"`
		Data    struct {
			AuthorizationURL string `json:"authorization_url"`
			Reference        string `json:"reference"`
		} `json:"data"`
	}
	if err := p.call(ctx, http.MethodPost, "/transaction/initialize", payload, &res); err != nil {
		return Initialized{}, err
	}
	if !res.Status || res.Data.AuthorizationURL == "" {
		return Initialized{}, fmt.Errorf("paystack refused the transaction: %s", res.Message)
	}
	return Initialized{
		AuthorizationURL: res.Data.AuthorizationURL,
		Reference:        res.Data.Reference,
	}, nil
}

func (p *Paystack) VerifyTransaction(ctx context.Context, reference string) (TransactionStatus, error) {
	var res struct {
		Status bool `json:"status"`
		Data   struct {
			Reference  string `json:"reference"`
			Status     string `json:"status"`
			Amount     int64  `json:"amount"`
			Currency   string `json:"currency"`
			Fees       int64  `json:"fees"`
			PaidAt     string `json:"paid_at"`
			Subaccount struct {
				SubaccountCode string `json:"subaccount_code"`
			} `json:"subaccount"`
		} `json:"data"`
	}
	if err := p.call(ctx, http.MethodGet, "/transaction/verify/"+reference, nil, &res); err != nil {
		return TransactionStatus{}, err
	}
	return TransactionStatus{
		Reference:      res.Data.Reference,
		Succeeded:      res.Data.Status == "success",
		Failed:         res.Data.Status == "failed" || res.Data.Status == "reversed",
		Abandoned:      res.Data.Status == "abandoned",
		AmountMinor:    res.Data.Amount,
		Currency:       res.Data.Currency,
		PSPFeeMinor:    res.Data.Fees,
		SubaccountCode: res.Data.Subaccount.SubaccountCode,
		PaidAt:         res.Data.PaidAt,
	}, nil
}

func (p *Paystack) call(ctx context.Context, method, path string, payload any, out any) error {
	if p.secretKey == "" {
		return ErrProviderUnavailable
	}
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, p.baseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.secretKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := p.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return err
	}
	if res.StatusCode >= 400 {
		// Never echo the response verbatim into an error that might reach a
		// user — Paystack errors can quote submitted account details.
		return fmt.Errorf("paystack returned %d for %s", res.StatusCode, path)
	}
	return json.Unmarshal(raw, out)
}

// ListBanks fetches the current Nigerian bank list from Paystack.
func (p *Paystack) ListBanks(ctx context.Context) ([]Bank, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		p.baseURL+"/bank?country=nigeria&perPage=100", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.secretKey)

	res, err := p.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var body struct {
		Status bool `json:"status"`
		Data   []struct {
			Name string `json:"name"`
			Code string `json:"code"`
		} `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return nil, err
	}
	if !body.Status {
		return nil, fmt.Errorf("paystack: bank list unavailable")
	}
	out := make([]Bank, 0, len(body.Data))
	for _, b := range body.Data {
		if b.Code == "" {
			continue
		}
		out = append(out, Bank{Name: b.Name, Code: b.Code})
	}
	return out, nil
}
