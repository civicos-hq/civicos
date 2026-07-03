package petitions

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// TestIntegration_PetitionConcurrentSign brings up a Postgres container via the
// repo's docker compose, starts the community-service, creates a petition and
// performs concurrent sign attempts from the same user to validate the unique
// signature constraint and idempotent handling.
func TestIntegration_PetitionConcurrentSign(t *testing.T) {
	// locate repo root by walking up until we find infrastructure/docker-compose.yml
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	repoRoot := ""
	cur := cwd
	for i := 0; i < 8; i++ {
		candidate := filepath.Join(cur, "infrastructure", "docker-compose.yml")
		if _, err := os.Stat(candidate); err == nil {
			repoRoot = cur
			break
		}
		cur = filepath.Dir(cur)
	}
	if repoRoot == "" {
		t.Fatalf("could not locate infrastructure folder from cwd %s", cwd)
	}

	// pick an available port for embedded Postgres to avoid collisions
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen tcp: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	cfgPg := embeddedpostgres.DefaultConfig().Port(uint32(port))
	pg := embeddedpostgres.NewDatabase(cfgPg)
	if err := pg.Start(); err != nil {
		t.Fatalf("start embedded postgres: %v", err)
	}
	defer func() { _ = pg.Stop() }()

	// DSN for the started embedded postgres
	dsn := fmt.Sprintf("postgres://postgres:postgres@localhost:%d/postgres?sslmode=disable", port)
	// wait for postgres to accept connections
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			t.Fatal("timed out waiting for postgres")
		default:
			db, err := sql.Open("pgx", dsn)
			if err == nil {
				if err = db.Ping(); err == nil {
					db.Close()
					goto DBReady
				}
				db.Close()
			}
			time.Sleep(250 * time.Millisecond)
		}
	}
DBReady:

	// start community-service with env pointing to embedded Postgres
	svcLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen service port: %v", err)
	}
	servicePort := svcLn.Addr().(*net.TCPAddr).Port
	_ = svcLn.Close()

	logFile, err := os.CreateTemp("/tmp", "community-service-*.log")
	if err != nil {
		t.Fatalf("create log file: %v", err)
	}
	defer func() {
		_ = logFile.Close()
	}()
	t.Logf("service logs: %s", logFile.Name())

	svcCmd := exec.Command("go", "run", "./cmd/server")
	svcCmd.Dir = filepath.Join(repoRoot, "services", "community-service")
	svcCmd.Stdout = logFile
	svcCmd.Stderr = logFile
	svcCmd.Env = append(os.Environ(), "DATABASE_URL="+dsn, "JWT_SECRET=integration-test-secret-which-is-very-long-0123456789", fmt.Sprintf("COMMUNITY_SERVICE_PORT=%d", servicePort))
	if err := svcCmd.Start(); err != nil {
		t.Fatalf("start service: %v", err)
	}
	done := make(chan error, 1)
	go func() {
		done <- svcCmd.Wait()
	}()
	defer func() {
		if svcCmd.Process != nil {
			_ = svcCmd.Process.Kill()
			select {
			case <-done:
			case <-time.After(5 * time.Second):
			}
		}
	}()

	// wait for service health
	hc := &http.Client{Timeout: 2 * time.Second}
	healthURL := fmt.Sprintf("http://localhost:%d/health", servicePort)
	ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel2()
	for {
		select {
		case <-ctx2.Done():
			t.Fatal("timed out waiting for service health")
		default:
			resp, err := hc.Get(healthURL)
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				goto ServiceReady
			}
			if resp != nil && resp.Body != nil {
				resp.Body.Close()
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
ServiceReady:

	// create auth token for the integration user — admin role so the test can
	// create a community (gated by requireAdminRole), and emailVerified=true so
	// the test can perform writes (gated by requireVerified).
	userID := "integration-user"
	type integrationClaims struct {
		UserID        string `json:"sub"`
		Role          string `json:"role"`
		EmailVerified bool   `json:"emailVerified"`
		jwt.RegisteredClaims
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, integrationClaims{
		UserID:        userID,
		Role:          "PLATFORM_ADMIN",
		EmailVerified: true,
	})
	signed, err := token.SignedString([]byte("integration-test-secret-which-is-very-long-0123456789"))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	// create a community through the service API so the route and schema are both exercised
	communityURL := fmt.Sprintf("http://localhost:%d/v1/communities", servicePort)
	communityBody := map[string]any{
		"name":  "Integration Community",
		"slug":  "integration-community-" + uuid.NewString(),
		"state": "Lagos",
		"lga":   "Ikeja",
	}
	cb, _ := json.Marshal(communityBody)
	creq, _ := http.NewRequest("POST", communityURL, bytes.NewReader(cb))
	creq.Header.Set("Content-Type", "application/json")
	creq.Header.Set("Authorization", "Bearer "+signed)
	cresp, err := hc.Do(creq)
	if err != nil {
		t.Fatalf("create community request: %v", err)
	}
	var body []byte
	if cresp.StatusCode != http.StatusCreated {
		body, _ = io.ReadAll(cresp.Body)
		cresp.Body.Close()
		t.Fatalf("unexpected status creating community: %d, body=%s", cresp.StatusCode, string(body))
	}
	body, _ = io.ReadAll(cresp.Body)
	cresp.Body.Close()
	var createCResp map[string]any
	if err := json.Unmarshal(body, &createCResp); err != nil {
		t.Fatalf("decode create community response: %v, body=%s", err, string(body))
	}
	dataVal, ok := createCResp["data"]
	if !ok || dataVal == nil {
		t.Fatalf("create community response missing 'data' field: %s", string(body))
	}
	dataMap, ok := dataVal.(map[string]any)
	if !ok {
		t.Fatalf("unexpected data type in create community response: %T, body=%s", dataVal, string(body))
	}
	commVal, ok := dataMap["community"]
	if !ok || commVal == nil {
		t.Fatalf("create community response missing 'community' field: %s", string(body))
	}
	comm, ok := commVal.(map[string]any)
	if !ok {
		t.Fatalf("unexpected community type in response: %T, body=%s", commVal, string(body))
	}
	communityID, ok := comm["id"].(string)
	if !ok || communityID == "" {
		t.Fatalf("community id not found or invalid in response: %s", string(body))
	}

	createURL := fmt.Sprintf("http://localhost:%d/v1/petitions", servicePort)

	createBody := map[string]any{"title": "Integration Petition", "description": "Test petition", "goal": 10, "communityId": communityID}
	b, _ := json.Marshal(createBody)
	req, _ := http.NewRequest("POST", createURL, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+signed)
	resp, err := hc.Do(req)
	if err != nil {
		t.Fatalf("create petition request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		body, _ = io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("unexpected status creating petition: %d, body=%s", resp.StatusCode, string(body))
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	var createResp map[string]any
	if err := json.Unmarshal(body, &createResp); err != nil {
		t.Fatalf("decode create response: %v, body=%s", err, string(body))
	}
	petitionData, ok := createResp["data"]
	if !ok || petitionData == nil {
		t.Fatalf("create petition response missing 'data' field: %s", string(body))
	}
	petitionMap, ok := petitionData.(map[string]any)
	if !ok {
		t.Fatalf("unexpected data type in create petition response: %T, body=%s", petitionData, string(body))
	}
	petVal, ok := petitionMap["petition"]
	if !ok || petVal == nil {
		t.Fatalf("create petition response missing 'petition' field: %s", string(body))
	}
	pet, ok := petVal.(map[string]any)
	if !ok {
		t.Fatalf("unexpected petition type in response: %T, body=%s", petVal, string(body))
	}
	petitionID, ok := pet["id"].(string)
	if !ok || petitionID == "" {
		t.Fatalf("petition id not found or invalid in response: %s", string(body))
	}

	// concurrently attempt to sign the petition multiple times as the same user
	var wg sync.WaitGroup
	concurrency := 10
	wg.Add(concurrency)
	signURL := fmt.Sprintf("http://localhost:%d/v1/petitions/%s/sign", servicePort, petitionID)
	errCh := make(chan error, concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			signReq, _ := http.NewRequest("POST", signURL, nil)
			signReq.Header.Set("Authorization", "Bearer "+signed)
			signResp, err := hc.Do(signReq)
			if err != nil {
				errCh <- err
				return
			}
			if signResp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(signResp.Body)
				errCh <- fmt.Errorf("unexpected sign status: %d, body=%s", signResp.StatusCode, string(body))
				signResp.Body.Close()
				return
			}
			if signResp.Body != nil {
				signResp.Body.Close()
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("sign petition request: %v", err)
		}
	}

	// fetch petition and assert signatureCount == 1
	getURL := fmt.Sprintf("http://localhost:%d/v1/petitions/%s", servicePort, petitionID)
	resp, err = hc.Get(getURL)
	if err != nil {
		t.Fatalf("get petition: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ = io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("unexpected get status: %d, body=%s", resp.StatusCode, string(body))
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	var getResp map[string]any
	if err := json.Unmarshal(body, &getResp); err != nil {
		t.Fatalf("decode get response: %v, body=%s", err, string(body))
	}
	dataVal, ok = getResp["data"]
	if !ok || dataVal == nil {
		t.Fatalf("get petition response missing 'data' field: %s", string(body))
	}
	dataMap, ok = dataVal.(map[string]any)
	if !ok {
		t.Fatalf("unexpected data type in get response: %T, body=%s", dataVal, string(body))
	}
	petVal, ok = dataMap["petition"]
	if !ok || petVal == nil {
		t.Fatalf("get petition response missing 'petition' field: %s", string(body))
	}
	pet2, ok := petVal.(map[string]any)
	if !ok {
		t.Fatalf("unexpected petition type in get response: %T, body=%s", petVal, string(body))
	}
	countFloat, ok := pet2["signatureCount"].(float64)
	if !ok {
		t.Fatalf("signatureCount missing or wrong type in response: %s", string(body))
	}
	if int(countFloat) != 1 {
		t.Fatalf("expected signatureCount 1, got %v", countFloat)
	}
}
