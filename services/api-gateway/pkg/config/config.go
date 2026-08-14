package config

import (
	"log"
	"os"
	"strings"
)

// ensureScheme resolves scheme-less service URLs.
//
// Cloud Run sets full https:// URLs, so in the current deployment this is a
// no-op. It stays because a scheme-less value is a silent failure rather
// than a loud one: the gateway would treat "civicos-identity-xyz.run.app"
// as a relative path and every proxied route would 502 with nothing
// pointing at the missing "https://".
//
// A colon means an explicit port, which only happens for a local or
// private-network address — those are plain HTTP. Anything else is a
// public hostname and must be HTTPS, or a platform's 301 to HTTPS gets
// bounced back to the client by the reverse proxy.
func ensureScheme(u string) string {
	if u == "" {
		return u
	}
	if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") {
		return u
	}
	if strings.Contains(u, ":") {
		return "http://" + u
	}
	return "https://" + u
}

type Config struct {
	Port                   string
	JWTSecret              string
	IdentityServiceURL     string
	CommunityServiceURL    string
	OrganizationServiceURL string
	CivicAIServiceURL      string
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

	organizationURL := os.Getenv("ORGANIZATION_SERVICE_URL")
	if organizationURL == "" {
		organizationURL = "http://localhost:3003"
	}

	civicaiURL := os.Getenv("CIVICAI_SERVICE_URL")
	if civicaiURL == "" {
		civicaiURL = "http://localhost:3004"
	}

	// PORT wins if set — Cloud Run injects it and the container must
	// listen on exactly that. Falls back to the service-specific var for
	// local dev, then to a hardcoded default.
	port := os.Getenv("PORT")
	if port == "" {
		port = os.Getenv("API_GATEWAY_PORT")
	}
	if port == "" {
		port = "3000"
	}

	return &Config{
		Port:                   port,
		JWTSecret:              secret,
		IdentityServiceURL:     ensureScheme(identityURL),
		CommunityServiceURL:    ensureScheme(communityURL),
		OrganizationServiceURL: ensureScheme(organizationURL),
		CivicAIServiceURL:      ensureScheme(civicaiURL),
		RedisURL:               os.Getenv("REDIS_URL"),
	}
}
