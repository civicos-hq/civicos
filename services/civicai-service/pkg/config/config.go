package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port         string
	JWTSecret    string
	GeminiAPIKey string
	// GeminiModel — override at deploy time when a newer flash tier ships.
	// Falls back to the last-known-good model so a missing env var doesn't
	// wedge the service.
	GeminiModel string
}

func Load() *Config {
	_ = godotenv.Load()
	cfg := &Config{
		// PORT wins when set — PaaS providers dictate it. Falls back to
		// CIVICAI_SERVICE_PORT for local dev, then a hardcoded default.
		Port:         getStr("PORT", getStr("CIVICAI_SERVICE_PORT", "3004")),
		JWTSecret:    require("JWT_SECRET"),
		GeminiAPIKey: require("GEMINI_API_KEY"),
		GeminiModel:  getStr("GEMINI_MODEL", "gemini-2.5-flash"),
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

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "config error: "+format+"\n", args...)
	os.Exit(1)
}
