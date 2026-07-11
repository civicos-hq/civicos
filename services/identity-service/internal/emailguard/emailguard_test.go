package emailguard

import "testing"

func TestGuard_IsDisposable(t *testing.T) {
	g := NewGuard()

	if g.Size() < 100 {
		t.Fatalf("embedded blocklist looks empty (Size=%d) — did the go:embed break?", g.Size())
	}

	cases := []struct {
		name  string
		email string
		want  bool
	}{
		// Well-known disposables that must stay blocked. Losing coverage
		// here means the embedded list has regressed.
		{"mailinator", "someone@mailinator.com", true},
		{"guerrillamail", "user@guerrillamail.com", true},
		{"yopmail", "user@yopmail.com", true},
		{"tempmail_io", "hi@tempmail.io", true},

		// Case + surrounding whitespace: the blocklist is lowercase,
		// input may not be. Guard normalizes.
		{"upper_case", "Someone@MAILINATOR.COM", true},
		{"trailing_space", "someone@mailinator.com ", true},

		// Legit domains must pass through untouched.
		{"gmail", "user@gmail.com", false},
		{"yahoo", "user@yahoo.com", false},
		{"outlook", "user@outlook.com", false},
		{"custom_domain", "user@civicos.ng", false},

		// Malformed input isn't the guard's problem — the shape check
		// belongs to the caller. Missing @ or empty domain returns
		// false so a bad request 400s downstream instead of falsely
		// flagging as disposable.
		{"no_at", "not-an-email", false},
		{"trailing_at", "someone@", false},
		{"empty", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := g.IsDisposable(tc.email); got != tc.want {
				t.Errorf("IsDisposable(%q) = %v; want %v", tc.email, got, tc.want)
			}
		})
	}
}
