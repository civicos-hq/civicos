package config

import (
	"log"
	"os"
	"strings"
)

// ensureScheme resolves scheme-less service URLs, which Render's Blueprint
// injects in two shapes:
//   - `hostport` (private network): "civicos-identity:10000" — plain HTTP
//   - `host` (public URL):          "civicos-identity.onrender.com" — must be
//     HTTPS, because onrender.com 301-redirects HTTP and the reverse proxy
//     would bounce that redirect back to the client
//
// A colon means an explicit port, i.e. the private-network/localhost form.
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

	// PORT wins if set — this is the env var PaaS providers like Render,
	// Fly, and Heroku dictate. Falls back to the service-specific var for
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
		RedisURL:               os.Getenv("REDIS_URL"),
	}
}
