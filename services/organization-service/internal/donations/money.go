// Package donations owns the Community Funding ledger: what a donor paid,
// which campaign it was for, how it split, and whether it settled.
//
// CivicOS is NOT the merchant of record. Each organization connects its own
// Paystack sub-account and Paystack settles directly to it, so nothing here
// represents a balance CivicOS owes anyone. These rows are a record of money
// that moved between other parties, which is exactly why they have to be
// right: the ledger is the only account of it we will ever have.
package donations

import "errors"

// ─── Fee math ───────────────────────────────────────────────────────────
//
// Every amount is an integer in the currency's minor unit (kobo for NGN).
// There is no float anywhere in this file, and there must never be one:
// binary floating point cannot represent 0.01 exactly, so `amount * 0.025`
// silently produces values like 249.99999999999997 that round unpredictably.
// Over thousands of donations that becomes real, unexplainable drift in
// somebody's flood-relief total.

// MaxBps is 100%. The platform fee is expressed in basis points — integer
// hundredths of a percent — so a 2.5% fee is 250 and needs no decimals.
const MaxBps int64 = 10_000

// maxAmountMinor caps a single donation at 1e13 minor units (₦100bn).
// Not a product limit so much as arithmetic headroom: Split multiplies
// amount by bps, and amount * 10_000 must stay well inside int64
// (max ≈ 9.2e18) for the intermediate product not to overflow.
const maxAmountMinor int64 = 10_000_000_000_000

var (
	ErrAmountNotPositive = errors.New("donation amount must be greater than zero")
	ErrAmountTooLarge    = errors.New("donation amount is implausibly large")
	ErrBpsOutOfRange     = errors.New("platform fee basis points must be between 0 and 10000")
)

// Split is how one donation divides. Gross is what the donor paid;
// PlatformFee is CivicOS's cut; Net is what Paystack settles to the
// organization's sub-account, before Paystack's own transaction charge.
//
// Paystack's charge is deliberately NOT modelled here. It is deducted by
// Paystack, varies by card/channel, and is only known once the transaction
// completes — so it is recorded from the webhook payload rather than
// predicted. Net below is "gross minus the CivicOS fee", which is the part
// we control and must be able to explain.
type Split struct {
	GrossMinor       int64
	PlatformFeeMinor int64
	NetMinor         int64
}

// ComputeSplit divides a donation by an integer basis-point rate.
//
// The fee is FLOOR-rounded, deliberately. Integer division truncates toward
// zero, so any fraction of a minor unit falls to the organization rather
// than the platform. On a single donation that is one kobo; the point is
// that the bias is fixed and in the beneficiary's favour, rather than a
// rounding mode nobody chose. It also guarantees PlatformFee <= Gross, so
// Net can never go negative.
func ComputeSplit(grossMinor, platformFeeBps int64) (Split, error) {
	if grossMinor <= 0 {
		return Split{}, ErrAmountNotPositive
	}
	if grossMinor > maxAmountMinor {
		return Split{}, ErrAmountTooLarge
	}
	if platformFeeBps < 0 || platformFeeBps > MaxBps {
		return Split{}, ErrBpsOutOfRange
	}

	// Safe from overflow: grossMinor <= 1e13 and platformFeeBps <= 1e4, so
	// the product is at most 1e17, comfortably inside int64.
	fee := (grossMinor * platformFeeBps) / MaxBps

	return Split{
		GrossMinor:       grossMinor,
		PlatformFeeMinor: fee,
		NetMinor:         grossMinor - fee,
	}, nil
}

// Valid re-checks the invariants a Split must always satisfy. Called before
// any ledger write, so a split that was constructed or deserialised by some
// future path still cannot persist as nonsense.
func (s Split) Valid() bool {
	return s.GrossMinor > 0 &&
		s.PlatformFeeMinor >= 0 &&
		s.NetMinor >= 0 &&
		s.PlatformFeeMinor+s.NetMinor == s.GrossMinor
}
