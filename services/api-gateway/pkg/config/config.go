package config

import (
	"log"
	"os"
)

type Config struct {
	Port                string
	JWTSecret           string
	IdentityServiceURL  string
	CommunityServiceURL string
	// RedisURL is used by the rate limiter. Empty means "no Redis" — the
	// limiter fails open (every request allowed) so a dev with no local
	// Redis can still run the gateway.
	RedisURL string
}

func Load() *Config {
	secret := os.Getenv("JWT_SECRET")
	if len(secret) < 32 {
		log.Fatal("JWT_SECRET must be at least 32 characters")
	}

	identityURL := os.Getenv("IDENTITY_SERVICE_URL")
	if identityURL == "" {
		identityURL = "http://localhost:3001"
	}

	communityURL := os.Getenv("COMMUNITY_SERVICE_URL")
	if communityURL == "" {
		communityURL = "http://localhost:3002"
	}

	port := os.Getenv("API_GATEWAY_PORT")
	if port == "" {
		port = "3000"
	}

	return &Config{
		Port:                port,
		JWTSecret:           secret,
		IdentityServiceURL:  identityURL,
		CommunityServiceURL: communityURL,
		RedisURL:            os.Getenv("REDIS_URL"),
	}
}
