package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string

	// ─── Flood forecasts (Google Flood Hub) ───
	//
	// All optional. With no API key the feature is simply off — CivicOS
	// runs exactly as before and no flood surface appears anywhere.
	//
	// FloodPollMinutes=0 is the kill switch. Google state the Flood
	// Forecasting API is in pilot and that breaking changes should be
	// expected, so an operator needs to stop consuming it without a
	// deploy.
	// GeocodingAPIKey powers the admin "suggest a location" button when
	// creating a community. Optional — without it admins type coordinates
	// by hand and the button is hidden.
	GeocodingAPIKey string

	FloodAPIKey        string
	FloodPollMinutes   int
	FloodRegionCode    string
	FloodMatchRadiusKm float64

	// NATSURL feeds the realtime bridge: notifications written by other
	// services (announcements, consultations, campaigns) are pushed to the
	// SSE hub instead of waiting for the user's next fetch. Optional.
	NATSURL string
}

func Load() *Config {
	_ = godotenv.Load()
	cfg := &Config{
		NATSURL: os.Getenv("NATS_URL"),
		// PORT wins when set — PaaS providers like Render dictate it.
		// Falls back to COMMUNITY_SERVICE_PORT for local dev.
		Port:        getStr("PORT", getStr("COMMUNITY_SERVICE_PORT", "3002")),
		DatabaseURL: require("DATABASE_URL"),
		JWTSecret:   require("JWT_SECRET"),

		GeocodingAPIKey:    os.Getenv("GOOGLE_GEOCODING_API_KEY"),
		FloodAPIKey:        os.Getenv("GOOGLE_FLOOD_API_KEY"),
		FloodPollMinutes:   getInt("FLOOD_POLL_INTERVAL_MINUTES", 60),
		FloodRegionCode:    getStr("FLOOD_REGION_CODE", "NG"),
		FloodMatchRadiusKm: getFloat("FLOOD_MATCH_RADIUS_KM", 50),
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

func getInt(key string, fallback int) int {
	if raw := os.Getenv(key); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			return n
		}
	}
	return fallback
}

func getFloat(key string, fallback float64) float64 {
	if raw := os.Getenv(key); raw != "" {
		if f, err := strconv.ParseFloat(raw, 64); err == nil {
			return f
		}
	}
	return fallback
}
