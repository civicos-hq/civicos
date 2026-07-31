package donations

import (
	"math"
	"testing"
)

// The fee split is the one piece of arithmetic in Community Funding where
// being wrong means someone's money is wrong. These tests are deliberately
// exhaustive about edges rather than representative.

func TestComputeSplit_Basic(t *testing.T) {
	cases := []struct {
		name             string
		gross, bps       int64
		wantFee, wantNet int64
	}{
		// ₦20,000 at 2.5% → ₦500 fee, ₦19,500 net.
		{"2.5% of 20k naira", 2_000_000, 250, 50_000, 1_950_000},
		{"zero fee is full pass-through", 2_000_000, 0, 0, 2_000_000},
		{"100% fee leaves nothing", 2_000_000, MaxBps, 2_000_000, 0},
		{"1% of 100 naira", 10_000, 100, 100, 9_900},
		{"smallest possible donation", 1, 250, 0, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ComputeSplit(c.gross, c.bps)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.PlatformFeeMinor != c.wantFee || got.NetMinor != c.wantNet {
				t.Fatalf("got fee=%d net=%d, want fee=%d net=%d",
					got.PlatformFeeMinor, got.NetMinor, c.wantFee, c.wantNet)
			}
			if !got.Valid() {
				t.Fatalf("split failed its own invariants: %+v", got)
			}
		})
	}
}

// Rounding must always favour the organization, never the platform.
func TestComputeSplit_RoundsDownInFavourOfTheOrg(t *testing.T) {
	// 1001 kobo at 2.5% is exactly 25.025 kobo. A kobo cannot be split, so
	// the fee must be 25 (floor), not 26.
	got, err := ComputeSplit(1001, 250)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.PlatformFeeMinor != 25 {
		t.Fatalf("fee should floor to 25, got %d", got.PlatformFeeMinor)
	}
	if got.NetMinor != 976 {
		t.Fatalf("net should be 976, got %d", got.NetMinor)
	}

	// Sweep a range and assert the invariant holds every time: the fee is
	// never more than the exact rational share.
	for gross := int64(1); gross <= 5000; gross++ {
		s, err := ComputeSplit(gross, 250)
		if err != nil {
			t.Fatalf("gross=%d: %v", gross, err)
		}
		if s.PlatformFeeMinor*MaxBps > gross*250 {
			t.Fatalf("gross=%d: fee %d exceeds exact share", gross, s.PlatformFeeMinor)
		}
		if !s.Valid() {
			t.Fatalf("gross=%d: invariants violated: %+v", gross, s)
		}
	}
}

// Whatever the inputs, the parts must reconstitute the whole. A split that
// loses or invents a single kobo is a reconciliation failure waiting to
// happen.
func TestComputeSplit_PartsAlwaysSumToGross(t *testing.T) {
	amounts := []int64{1, 2, 3, 7, 99, 100, 101, 999, 1_000, 12_345, 1_000_000, maxAmountMinor}
	rates := []int64{0, 1, 7, 100, 250, 333, 1_000, 9_999, MaxBps}
	for _, a := range amounts {
		for _, r := range rates {
			s, err := ComputeSplit(a, r)
			if err != nil {
				t.Fatalf("amount=%d bps=%d: %v", a, r, err)
			}
			if s.PlatformFeeMinor+s.NetMinor != a {
				t.Fatalf("amount=%d bps=%d: %d + %d != %d",
					a, r, s.PlatformFeeMinor, s.NetMinor, a)
			}
			if s.NetMinor < 0 || s.PlatformFeeMinor < 0 {
				t.Fatalf("amount=%d bps=%d: negative component %+v", a, r, s)
			}
		}
	}
}

func TestComputeSplit_Rejects(t *testing.T) {
	cases := []struct {
		name       string
		gross, bps int64
		want       error
	}{
		{"zero amount", 0, 250, ErrAmountNotPositive},
		{"negative amount", -1, 250, ErrAmountNotPositive},
		{"absurd amount", maxAmountMinor + 1, 250, ErrAmountTooLarge},
		{"negative bps", 1000, -1, ErrBpsOutOfRange},
		{"bps over 100%", 1000, MaxBps + 1, ErrBpsOutOfRange},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := ComputeSplit(c.gross, c.bps); err != c.want {
				t.Fatalf("got %v, want %v", err, c.want)
			}
		})
	}
}

// The amount cap exists so gross*bps cannot overflow int64. Prove the
// intermediate product stays in range at the very top of the allowed band,
// and that we are refusing values that would breach it.
func TestComputeSplit_NoOverflowAtTheCap(t *testing.T) {
	s, err := ComputeSplit(maxAmountMinor, MaxBps)
	if err != nil {
		t.Fatalf("max amount at max bps should be accepted: %v", err)
	}
	if !s.Valid() || s.PlatformFeeMinor != maxAmountMinor {
		t.Fatalf("unexpected split at the cap: %+v", s)
	}
	// The guarded product, computed in the widest available type, must be
	// far below the int64 ceiling — that is what the cap is buying us.
	if float64(maxAmountMinor)*float64(MaxBps) >= math.MaxInt64 {
		t.Fatalf("amount cap no longer protects against overflow")
	}
}

func TestSplit_ValidCatchesTampering(t *testing.T) {
	good, _ := ComputeSplit(1_000, 250)
	if !good.Valid() {
		t.Fatalf("well-formed split should be valid")
	}
	bad := good
	bad.NetMinor += 1 // parts no longer sum to gross
	if bad.Valid() {
		t.Fatalf("Valid() must reject a split whose parts do not sum to gross")
	}
	neg := Split{GrossMinor: 100, PlatformFeeMinor: 150, NetMinor: -50}
	if neg.Valid() {
		t.Fatalf("Valid() must reject negative components")
	}
}
