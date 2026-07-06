package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                int
	DatabaseURL         string
	JWTSecret           string
	JWTExpiresIn        string
	JWTRefreshExpiresIn string

	// Public URL of the web frontend — used to build links inside emails.
	AppURL string

	// SMTP config. When SMTPHost is empty the service falls back to the
	// console mailer (verification links are printed to the service log).
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string
}

// Load validates and returns config from environment.
func Load() *Config {
	_ = godotenv.Load()

	cfg := &Config{
		// PORT wins when set — PaaS providers like Render dictate it.
		// Falls back to IDENTITY_SERVICE_PORT for local dev.
		Port:                getInt("PORT", getInt("IDENTITY_SERVICE_PORT", 3001)),
		DatabaseURL:         require("DATABASE_URL"),
		JWTSecret:           require("JWT_SECRET"),
		JWTExpiresIn:        getStr("JWT_EXPIRES_IN", "7d"),
		JWTRefreshExpiresIn: getStr("JWT_REFRESH_EXPIRES_IN", "30d"),
		AppURL:              getStr("APP_URL", "http://localhost:5173"),
		SMTPHost:            getStr("SMTP_HOST", ""),
		SMTPPort:            getInt("SMTP_PORT", 1025),
		SMTPUser:            getStr("SMTP_USER", ""),
		SMTPPassword:        getStr("SMTP_PASSWORD", ""),
		SMTPFrom:            getStr("SMTP_FROM", "CivicOS <no-reply@civicos.local>"),
	}

	if len(cfg.JWTSecret) < 32 {
		fatalf("JWT_SECRET must be at least 32 characters")
	}

	return cfg
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

func getInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			fatalf("env var %s must be an integer, got: %s", key, v)
		}
		return n
	}
	return fallback
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "❌ config error: "+format+"\n", args...)
	os.Exit(1)
}
