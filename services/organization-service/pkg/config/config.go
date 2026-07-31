package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string

	// ─── Community Funding (Phase 3) ───
	//
	// Paystack is optional at boot: the campaign lifecycle, admin review and
	// public pages all work without it, and a missing key must degrade
	// donations rather than take the whole service down. PaystackEnabled
	// reports whether the donation endpoints can actually function.
	PaystackSecretKey string
	PaystackPublicKey string

	// PlatformFeeBps is CivicOS's cut in integer basis points (250 = 2.5%).
	// Basis points, never a float — see internal/donations/money.go for why.
	PlatformFeeBps int64

	// DonationCallbackURL is where Paystack returns the donor after payment.
	// The redirect is only a UX hint — settlement is decided by the webhook,
	// never by the browser coming back, because a donor can close the tab or
	// craft the return URL themselves.
	DonationCallbackURL string

	// ReconcileIntervalMinutes is how often the background reconciliation
	// sweep runs. 0 disables it — the on-demand admin endpoint still works.
	//
	// This is the job that catches donations whose webhook never arrived.
	// Turning it off means trusting that Paystack's delivery never fails,
	// which is not a property any payment provider offers.
	ReconcileIntervalMinutes int64

	// ─── Mail (donation receipts) ───
	//
	// Optional. When SMTPHost is empty the service falls back to the console
	// mailer and receipts are printed to the log — a donation must never be
	// refused because mail is misconfigured.
	SMTPHost     string
	SMTPPort     int64
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string

	// AppURL is the public web origin, used to link a receipt back to the
	// campaign it funded.
	AppURL string
}

// PaystackEnabled reports whether donation endpoints can operate. Checked at
// request time so the service still starts (and serves campaigns) when the
// key is absent, e.g. in a dev environment that only needs the public pages.
func (c *Config) PaystackEnabled() bool { return c.PaystackSecretKey != "" }

// PaystackLive reports whether the configured key is a LIVE key. Callers use
// this to refuse destructive test fixtures against real money.
func (c *Config) PaystackLive() bool {
	return strings.HasPrefix(c.PaystackSecretKey, "sk_live_")
}

func Load() *Config {
	_ = godotenv.Load()
	cfg := &Config{
		// PORT wins when set — PaaS providers like Render dictate it.
		// Falls back to ORGANIZATION_SERVICE_PORT for local dev.
		Port:        getStr("PORT", getStr("ORGANIZATION_SERVICE_PORT", "3003")),
		DatabaseURL: require("DATABASE_URL"),
		JWTSecret:   require("JWT_SECRET"),

		PaystackSecretKey:   os.Getenv("PAYSTACK_SECRET_KEY"),
		PaystackPublicKey:   os.Getenv("PAYSTACK_PUBLIC_KEY"),
		PlatformFeeBps:      getInt64("PLATFORM_FEE_BPS", 0),
		DonationCallbackURL: getStr("DONATION_CALLBACK_URL", "http://localhost:5173/campaigns"),

		ReconcileIntervalMinutes: getInt64("RECONCILE_INTERVAL_MINUTES", 60),

		SMTPHost:     os.Getenv("SMTP_HOST"),
		SMTPPort:     getInt64("SMTP_PORT", 1025),
		SMTPUser:     os.Getenv("SMTP_USER"),
		SMTPPassword: os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:     getStr("SMTP_FROM", "CivicOS <no-reply@civicos.local>"),
		AppURL:       ensureScheme(getStr("APP_URL", "http://localhost:5173")),
	}
	if len(cfg.JWTSecret) < 32 {
		fatalf("JWT_SECRET must be at least 32 characters")
	}
	// A malformed fee is a silent money bug, so refuse to boot on one rather
	// than defaulting and quietly charging the wrong rate.
	if cfg.PlatformFeeBps < 0 || cfg.PlatformFeeBps > 10_000 {
		fatalf("PLATFORM_FEE_BPS must be between 0 and 10000 (basis points), got %d", cfg.PlatformFeeBps)
	}
	if cfg.ReconcileIntervalMinutes < 0 {
		fatalf("RECONCILE_INTERVAL_MINUTES must not be negative, got %d", cfg.ReconcileIntervalMinutes)
	}
	if cfg.PaystackSecretKey != "" && !strings.HasPrefix(cfg.PaystackSecretKey, "sk_") {
		fatalf("PAYSTACK_SECRET_KEY does not look like a Paystack secret key")
	}
	return cfg
}

// ensureScheme hardens APP_URL against the bare-host form Render's Blueprint
// injects via `fromService.host` — without a scheme, the campaign link in a
// receipt would be unclickable. A value with an explicit port is a dev URL.
func ensureScheme(u string) string {
	if u == "" || strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") {
		return u
	}
	if strings.Contains(u, ":") {
		return "http://" + u
	}
	return "https://" + u
}

func getInt64(key string, fallback int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		fatalf("%s must be an integer, got %q", key, v)
	}
	return n
}

func require(key string) string {
	v := os.Getenv(key)
	if v == "" {
		fatalf("missing required env var: %s", key)
	}
	return v
}

func getStr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "❌ config error: "+format+"\n", args...)
	os.Exit(1)
}
