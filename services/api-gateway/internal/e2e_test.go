package internal_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/civicos/api-gateway/internal/middleware"
	"github.com/civicos/api-gateway/internal/proxy"
	"github.com/civicos/api-gateway/pkg/config"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func startGateway(cfg *config.Config) *httptest.Server {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	authMiddleware := middleware.JWTAuth(cfg)

	identityPublic := proxy.NewReverseProxy(cfg.IdentityServiceURL, "/api")
	identityProtected := proxy.NewReverseProxy(cfg.IdentityServiceURL, "/api")
	communityProxy := proxy.NewReverseProxy(cfg.CommunityServiceURL, "/api")

	r.POST("/api/v1/auth/register", identityPublic)
	r.POST("/api/v1/auth/login", identityPublic)
	r.POST("/api/v1/auth/refresh", identityPublic)
	r.GET("/api/v1/auth/me", authMiddleware, identityProtected)

	r.GET("/api/v1/communities", communityProxy)
	r.GET("/api/v1/communities/:id", communityProxy)
	r.POST("/api/v1/communities", authMiddleware, communityProxy)

	r.GET("/api/v1/issues", communityProxy)
	r.GET("/api/v1/issues/:id", communityProxy)
	r.POST("/api/v1/issues", authMiddleware, communityProxy)
	r.POST("/api/v1/issues/:id/upvote", authMiddleware, communityProxy)
	r.PATCH("/api/v1/issues/:id/status", authMiddleware, communityProxy)

	r.GET("/api/v1/petitions", communityProxy)
	r.GET("/api/v1/petitions/:id", communityProxy)
	r.POST("/api/v1/petitions", authMiddleware, communityProxy)
	r.POST("/api/v1/petitions/:id/sign", authMiddleware, communityProxy)

	return httptest.NewServer(r)
}

func TestRegisterProxiesToIdentity(t *testing.T) {
	// upstream identity server records the request
	var gotPath string
	identity := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer identity.Close()

	community := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer community.Close()

	cfg := &config.Config{Port: "", JWTSecret: strings.Repeat("x", 32), IdentityServiceURL: identity.URL, CommunityServiceURL: community.URL}
	gw := startGateway(cfg)
	defer gw.Close()

	resp, err := http.Post(gw.URL+"/api/v1/auth/register", "application/json", strings.NewReader(`{"email":"a@b"}`))
	if err != nil {
		t.Fatalf("post gateway: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 from gateway, got %d", resp.StatusCode)
	}
	if gotPath == "" {
		t.Fatalf("upstream did not receive request")
	}
}

func signedToken(t *testing.T, cfg *config.Config, subject string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{Subject: subject})
	signed, err := token.SignedString([]byte(cfg.JWTSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

func TestCommunityCreateRequiresAuthAndProxies(t *testing.T) {
	var gotPath, gotAuth string
	community := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer community.Close()

	identity := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer identity.Close()

	cfg := &config.Config{Port: "", JWTSecret: strings.Repeat("x", 32), IdentityServiceURL: identity.URL, CommunityServiceURL: community.URL}
	gw := startGateway(cfg)
	defer gw.Close()

	req, _ := http.NewRequest("POST", gw.URL+"/api/v1/communities", bytes.NewBufferString(`{"name":"A","slug":"a","state":"Lagos","lga":"Ikeja"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 when missing auth, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	req2, _ := http.NewRequest("POST", gw.URL+"/api/v1/communities", bytes.NewBufferString(`{"name":"A","slug":"a","state":"Lagos","lga":"Ikeja"}`))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+signedToken(t, cfg, "user-1"))
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("request2: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 when authorized, got %d", resp2.StatusCode)
	}
	if !strings.HasPrefix(gotAuth, "Bearer ") {
		t.Fatalf("expected Authorization header proxied, got %q", gotAuth)
	}
	if gotPath == "" {
		t.Fatalf("expected community upstream request path to be set")
	}
}

func TestIssueStatusPatchRequiresAuthAndProxies(t *testing.T) {
	var gotMethod, gotPath string
	community := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer community.Close()

	identity := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer identity.Close()

	cfg := &config.Config{Port: "", JWTSecret: strings.Repeat("x", 32), IdentityServiceURL: identity.URL, CommunityServiceURL: community.URL}
	gw := startGateway(cfg)
	defer gw.Close()

	req, _ := http.NewRequest("PATCH", gw.URL+"/api/v1/issues/issue-123/status", bytes.NewBufferString(`{"status":"RESOLVED"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 when missing auth, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	req2, _ := http.NewRequest("PATCH", gw.URL+"/api/v1/issues/issue-123/status", bytes.NewBufferString(`{"status":"RESOLVED"}`))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+signedToken(t, cfg, "user-2"))
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("request2: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 when authorized, got %d", resp2.StatusCode)
	}
	if gotMethod != http.MethodPatch {
		t.Fatalf("expected proxied method PATCH, got %s", gotMethod)
	}
	if gotPath == "" {
		t.Fatalf("expected issue upstream request path to be set")
	}
}

func TestPetitionsListProxiesToCommunity(t *testing.T) {
	var gotPath string
	community := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer community.Close()

	identity := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer identity.Close()

	cfg := &config.Config{Port: "", JWTSecret: strings.Repeat("x", 32), IdentityServiceURL: identity.URL, CommunityServiceURL: community.URL}
	gw := startGateway(cfg)
	defer gw.Close()

	resp, err := http.Get(gw.URL + "/api/v1/petitions")
	if err != nil {
		t.Fatalf("get petitions: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gateway, got %d", resp.StatusCode)
	}
	if gotPath == "" {
		t.Fatalf("expected petition upstream request path to be set")
	}
}

func TestPetitionSignRequiresAuthAndProxies(t *testing.T) {
	// community upstream will verify Authorization header
	var gotAuth string
	community := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"signed":true}`)
	}))
	defer community.Close()

	identity := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// identity not used in this test
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer identity.Close()

	cfg := &config.Config{Port: "", JWTSecret: strings.Repeat("x", 32), IdentityServiceURL: identity.URL, CommunityServiceURL: community.URL}
	gw := startGateway(cfg)
	defer gw.Close()

	// call without Authorization -> should be 401
	req, _ := http.NewRequest("POST", gw.URL+"/api/v1/petitions/123/sign", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 when missing auth, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// call with Authorization -> should proxy to community and forward header
	req2, _ := http.NewRequest("POST", gw.URL+"/api/v1/petitions/123/sign", nil)
	req2.Header.Set("Authorization", "Bearer "+signedToken(t, cfg, "user-1"))
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("request2: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 when authorized, got %d", resp2.StatusCode)
	}
	if !strings.HasPrefix(gotAuth, "Bearer ") {
		t.Fatalf("expected Authorization header proxied, got %q", gotAuth)
	}
}
