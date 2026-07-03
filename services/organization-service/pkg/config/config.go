package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string
}

func Load() *Config {
	_ = godotenv.Load()
	cfg := &Config{
		Port:        getStr("ORGANIZATION_SERVICE_PORT", "3003"),
		DatabaseURL: require("DATABASE_URL"),
		JWTSecret:   require("JWT_SECRET"),
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
	fmt.Fprintf(os.Stderr, "❌ config error: "+format+"\n", args...)
	os.Exit(1)
}
