package donations

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"strings"
	"testing"
)

// The webhook is the only unauthenticated write path in the service, and the
// only place an attacker can try to fabricate a donation. These tests are
// about that boundary, not about Paystack's happy path.

const testSecret = "sk_test_deadbeefdeadbeefdeadbeefdeadbeef"

func sign(body, secret string) string {
	mac := hmac.New(sha512.New, []byte(secret))
	mac.Write([]byte(body))
	return hex.EncodeToString(mac.Sum(nil))
}

const successBody = `{"event":"charge.success","data":{"id":302961,"reference":"civicos_abc123","status":"success","amount":2000000,"currency":"NGN","fees":30000,"paid_at":"2026-07-30T10:00:00.000Z","subaccount":{"subaccount_code":"ACCT_xyz"}}}`

func TestVerifyWebhook_AcceptsGenuineSignature(t *testing.T) {
	p := NewPaystack(testSecret)
	ev, err := p.VerifyWebhook([]byte(successBody), sign(successBody, testSecret))
	if err != nil {
		t.Fatalf("genuine signature rejected: %v", err)
	}
	if ev.Reference != "civicos_abc123" {
		t.Fatalf("reference = %q", ev.Reference)
	}
	if ev.Type != "charge.success" || !ev.Status.Succeeded {
		t.Fatalf("expected a successful charge, got type=%q succeeded=%v", ev.Type, ev.Status.Succeeded)
	}
	if ev.Status.AmountMinor != 2_000_000 || ev.Status.PSPFeeMinor != 30_000 {
		t.Fatalf("amounts not carried through: %+v", ev.Status)
	}
	if ev.Status.SubaccountCode != "ACCT_xyz" {
		t.Fatalf("subaccount not carried through: %q", ev.Status.SubaccountCode)
	}
	// The dedupe key must be stable and include the provider's event id.
	if ev.ID != "charge.success:302961" {
		t.Fatalf("dedupe id = %q", ev.ID)
	}
}

// The whole point of the endpoint's security. Each of these must be refused.
func TestVerifyWebhook_RejectsForgeries(t *testing.T) {
	p := NewPaystack(testSecret)
	good := sign(successBody, testSecret)

	cases := []struct {
		name string
		body string
		sig  string
	}{
		{"no signature at all", successBody, ""},
		{"signature from the wrong key", successBody, sign(successBody, "sk_test_attacker")},
		{"garbage signature", successBody, "not-a-signature"},
		{"empty signature hex", successBody, strings.Repeat("0", 128)},
		// The attack that matters most: a valid signature lifted from one
		// payload and replayed against a payload whose amount was inflated.
		{"tampered amount, original signature", strings.Replace(successBody, `"amount":2000000`, `"amount":999999999`, 1), good},
		{"tampered reference, original signature", strings.Replace(successBody, "civicos_abc123", "civicos_someone_else", 1), good},
		{"tampered subaccount, original signature", strings.Replace(successBody, "ACCT_xyz", "ACCT_attacker", 1), good},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := p.VerifyWebhook([]byte(c.body), c.sig); err == nil {
				t.Fatalf("forgery was ACCEPTED — this would let an attacker fabricate a donation")
			}
		})
	}
}

// A single flipped byte anywhere in the body must invalidate the signature.
func TestVerifyWebhook_AnyBodyMutationInvalidates(t *testing.T) {
	p := NewPaystack(testSecret)
	good := sign(successBody, testSecret)
	for i := 0; i < len(successBody); i += 17 {
		b := []byte(successBody)
		b[i] ^= 0x01
		if _, err := p.VerifyWebhook(b, good); err == nil {
			t.Fatalf("mutation at byte %d was accepted", i)
		}
	}
}

// Signature case must not matter; tampering still must.
func TestVerifyWebhook_SignatureCaseInsensitive(t *testing.T) {
	p := NewPaystack(testSecret)
	upper := strings.ToUpper(sign(successBody, testSecret))
	if _, err := p.VerifyWebhook([]byte(successBody), upper); err != nil {
		t.Fatalf("uppercase hex signature should verify: %v", err)
	}
}

// Fail closed: with no key configured, verification is impossible, so every
// call must be refused rather than waved through.
func TestVerifyWebhook_FailsClosedWithoutKey(t *testing.T) {
	p := NewPaystack("")
	if _, err := p.VerifyWebhook([]byte(successBody), sign(successBody, testSecret)); err != ErrProviderUnavailable {
		t.Fatalf("expected ErrProviderUnavailable, got %v", err)
	}
}

// A correctly-signed body that isn't a transaction event must be rejected
// rather than producing a zero-value event the handler might act on.
func TestVerifyWebhook_RejectsUnusablePayloads(t *testing.T) {
	p := NewPaystack(testSecret)
	for _, body := range []string{
		`{"event":"charge.success","data":{}}`, // no reference
		`not json at all`,
		`{}`,
	} {
		if _, err := p.VerifyWebhook([]byte(body), sign(body, testSecret)); err == nil {
			t.Fatalf("unusable payload accepted: %s", body)
		}
	}
}

func TestVerifyWebhook_FailedChargeIsNotSuccess(t *testing.T) {
	body := strings.Replace(successBody, `"status":"success"`, `"status":"failed"`, 1)
	p := NewPaystack(testSecret)
	ev, err := p.VerifyWebhook([]byte(body), sign(body, testSecret))
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if ev.Status.Succeeded {
		t.Fatalf("a failed charge must not read as succeeded")
	}
	if !ev.Status.Failed {
		t.Fatalf("a failed charge should be marked failed")
	}
}

// API calls must refuse rather than send an unauthenticated request when no
// key is configured.
func TestAPICalls_RequireAKey(t *testing.T) {
	p := NewPaystack("")
	if _, err := p.CreateSubaccount(context.Background(), CreateSubaccountInput{}); err != ErrProviderUnavailable {
		t.Fatalf("CreateSubaccount without a key: %v", err)
	}
	if _, err := p.InitializeTransaction(context.Background(), InitializeInput{}); err != ErrProviderUnavailable {
		t.Fatalf("InitializeTransaction without a key: %v", err)
	}
}
