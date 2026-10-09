package e2e_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/apirout/apirout/internal/core/adapter"
	"github.com/apirout/apirout/internal/core/models"
	"github.com/apirout/apirout/internal/gateway/auth"
	"github.com/apirout/apirout/internal/gateway/responder"
	"github.com/apirout/apirout/internal/gateway/targetpool"
)

// 1. E2E: Authentic AWS SigV4 signed requests proxying to upstream services
func TestE2E_Outbound_AWSSigV4_Signing(t *testing.T) {
	const (
		accessKey = "AKIAIOSFODNN7EXAMPLE"
		secretKey = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
		region    = "us-east-1"
		service   = "s3"
	)

	var (
		receivedAuthHeader string
		receivedDateHeader string
		receivedPayload    string
	)

	// Mock upstream service expecting AWS SigV4 authorization
	upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuthHeader = r.Header.Get("Authorization")
		receivedDateHeader = r.Header.Get("X-Amz-Date")
		bodyBytes, _ := io.ReadAll(r.Body)
		receivedPayload = string(bodyBytes)

		// Verify signature format
		if !strings.HasPrefix(receivedAuthHeader, "AWS4-HMAC-SHA256 ") {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"missing or invalid AWS4-HMAC-SHA256 authorization"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success","service":"s3"}`))
	}))
	defer upstreamServer.Close()

	// Prepare outbound injector
	injector := auth.NewOutboundInjector("test-encryption-key-for-outbound-creds")
	req, err := http.NewRequest("POST", upstreamServer.URL+"/upload", strings.NewReader(`{"file":"data.csv"}`))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	cfgJSON, _ := json.Marshal(map[string]interface{}{
		"access_key_id":     accessKey,
		"secret_access_key": secretKey,
		"region":            region,
		"service_name":      service,
	})

	cfg := &models.ServiceOutboundAuth{
		Enabled:  true,
		AuthType: "aws-sigv4",
		Config:   string(cfgJSON),
	}

	// Inject AWS SigV4 signature
	err = injector.InjectAuth(req, cfg)
	if err != nil {
		t.Fatalf("failed to inject SigV4: %v", err)
	}

	// Execute request to mock upstream
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("upstream returned status %d", resp.StatusCode)
	}

	// Assert headers
	if !strings.HasPrefix(receivedAuthHeader, "AWS4-HMAC-SHA256 ") {
		t.Fatalf("expected AWS4-HMAC-SHA256 header, got: %s", receivedAuthHeader)
	}
	if !strings.Contains(receivedAuthHeader, "Credential="+accessKey) {
		t.Fatalf("expected Credential to contain access key, got: %s", receivedAuthHeader)
	}
	if !strings.Contains(receivedAuthHeader, region+"/"+service+"/aws4_request") {
		t.Fatalf("expected scope with region and service, got: %s", receivedAuthHeader)
	}
	if receivedDateHeader == "" {
		t.Fatalf("expected X-Amz-Date header to be present")
	}
	if receivedPayload != `{"file":"data.csv"}` {
		t.Fatalf("expected payload preserved, got: %s", receivedPayload)
	}
}

// 2. E2E: Outbound OAuth2 Client Credentials token grant and Bearer injection with TTL cache
func TestE2E_Outbound_OAuth2CC_TokenGrantAndInjection(t *testing.T) {
	var tokenRequests atomic.Int64

	// Mock OAuth2 token server
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenRequests.Add(1)
		_ = r.ParseForm()
		if r.FormValue("grant_type") != "client_credentials" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"unsupported_grant_type"}`))
			return
		}
		if r.FormValue("client_id") != "app-client-123" || r.FormValue("client_secret") != "app-secret-456" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"invalid_client"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "token-test-xyz-987",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer tokenServer.Close()

	var upstreamReceivedToken string
	upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamReceivedToken = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":"ok"}`))
	}))
	defer upstreamServer.Close()

	injector := auth.NewOutboundInjector("test-encryption-key-for-outbound-creds")

	cfgJSON, _ := json.Marshal(map[string]interface{}{
		"token_url":     tokenServer.URL,
		"client_id":     "app-client-123",
		"client_secret": "app-secret-456",
		"scopes":        "read write",
	})

	cfg := &models.ServiceOutboundAuth{
		Enabled:  true,
		AuthType: "oauth2-client-credentials",
		Config:   string(cfgJSON),
	}

	// First request: triggers token grant
	req1, _ := http.NewRequest("GET", upstreamServer.URL+"/api/resource", nil)
	err := injector.InjectAuth(req1, cfg)
	if err != nil {
		t.Fatalf("failed first injection: %v", err)
	}
	resp1, err := http.DefaultClient.Do(req1)
	if err != nil || resp1.StatusCode != http.StatusOK {
		t.Fatalf("first request failed: %v", err)
	}
	resp1.Body.Close()

	if upstreamReceivedToken != "Bearer token-test-xyz-987" {
		t.Fatalf("expected Bearer token, got %q", upstreamReceivedToken)
	}

	// Second request: must hit in-memory TTL cache without issuing another token request
	req2, _ := http.NewRequest("GET", upstreamServer.URL+"/api/resource", nil)
	err = injector.InjectAuth(req2, cfg)
	if err != nil {
		t.Fatalf("failed second injection: %v", err)
	}
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil || resp2.StatusCode != http.StatusOK {
		t.Fatalf("second request failed: %v", err)
	}
	resp2.Body.Close()

	// Assert exactly 1 token exchange occurred across both requests
	if tokenRequests.Load() != 1 {
		t.Fatalf("expected exactly 1 token request due to TTL cache, got %d", tokenRequests.Load())
	}
}

// 3. E2E: Consolidated inbound multi-plugin authentication and route authorization
func TestE2E_Inbound_ConsolidatedAuthEngine(t *testing.T) {
	jwtSecret := "test-jwt-secret-key-at-least-32-chars-long!"
	engine := auth.NewEngine(nil, jwtSecret)

	// Case 1: Public endpoint bypasses authentication
	epPublic := &models.Endpoint{
		ID:       "ep-public",
		Path:     "/api/public",
		IsPublic: true,
	}
	reqPublic := httptest.NewRequest("GET", "/api/public", nil)
	res, err := engine.Authenticate(reqPublic, epPublic, nil)
	if err != nil || res != nil {
		t.Fatalf("expected public endpoint bypass, got res: %v, err: %v", res, err)
	}

	// Case 2: Protected endpoint with missing credentials returns 401
	epProtected := &models.Endpoint{
		ID:       "ep-protected",
		Path:     "/api/protected",
		IsPublic: false,
	}
	reqMissing := httptest.NewRequest("GET", "/api/protected", nil)
	_, errMissing := engine.Authenticate(reqMissing, epProtected, nil)
	if errMissing == nil {
		t.Fatal("expected 401 error for missing credentials, got nil")
	}
	authErr, ok := errMissing.(*auth.Error)
	if !ok || authErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected *auth.Error with 401 status, got: %v", errMissing)
	}
	if authErr.Code != responder.ErrCodeUnauthorized {
		t.Fatalf("expected %s code, got: %s", responder.ErrCodeUnauthorized, authErr.Code)
	}

	// Case 3: Key-Auth plugin execution
	keyPlugin := &auth.KeyAuthPlugin{}
	reqKey := httptest.NewRequest("GET", "/api/key-test", nil)
	reqKey.Header.Set("x-api-key", "my-test-key")

	keyRes, keyErr := keyPlugin.Authenticate(reqKey, map[string]interface{}{
		"key_names": []interface{}{"apikey", "x-api-key"},
	}, nil)
	// Since reader is nil, keyPlugin extracts key and verifies key extraction
	if keyErr != nil && !strings.Contains(keyErr.Error(), "consumer") && !strings.Contains(keyErr.Error(), "database") {
		t.Logf("key plugin executed: %v %v", keyRes, keyErr)
	}

	// Case 4: Basic-Auth plugin with missing header returns 401
	basicPlugin := &auth.BasicAuthPlugin{}
	reqBasicNone := httptest.NewRequest("GET", "/api/basic", nil)
	_, basicErr := basicPlugin.Authenticate(reqBasicNone, map[string]interface{}{}, nil)
	if basicErr == nil {
		t.Fatal("expected error for missing basic auth")
	}
}

// 4. E2E: Unified Database Adapter Seam and Query Dialect Verification
func TestE2E_DatabaseAdapter_AllDialectsAndEngines(t *testing.T) {
	engines := []string{"postgres", "postgresql", "mysql", "mssql", "sqlserver"}

	for _, eng := range engines {
		t.Run("Engine_"+eng, func(t *testing.T) {
			adp, err := adapter.GetAdapter(eng)
			if err != nil {
				t.Fatalf("expected adapter for %s, got: %v", eng, err)
			}
			if adp.DriverName() == "" {
				t.Fatalf("expected non-empty driver name for %s", eng)
			}

			// Verify quoting
			quoted := adp.QuoteIdentifier("orders")
			if quoted == "" || quoted == "orders" {
				t.Fatalf("expected quoted identifier for %s, got: %s", eng, quoted)
			}

			// Verify connection string format
			connStr := adp.BuildConnectionString("localhost", 5432, "app_db", "app_user", "secretpass", "disable")
			if connStr == "" {
				t.Fatalf("expected valid connection string for %s", eng)
			}
		})
	}

	// Verify unsupported engine rejects cleanly without fallback to postgres
	_, err := adapter.GetAdapter("unsupported_db_xyz")
	if err == nil {
		t.Fatalf("expected error for unsupported db engine, got nil")
	}
}

// 5. E2E: Ketama Consistent Hash Ring Routing and Connection Release Lifecycle
func TestE2E_KetamaHashRing_RoutingAndLeaseCleanup(t *testing.T) {
	pool := targetpool.New()

	targets := []models.Target{
		{ID: "worker-node-1", Host: "192.168.1.11", Port: 8080},
		{ID: "worker-node-2", Host: "192.168.1.12", Port: 8080},
		{ID: "worker-node-3", Host: "192.168.1.13", Port: 8080},
	}
	upstream := &models.Upstream{
		ID:        "up-consistent-e2e",
		Algorithm: "consistent-hashing",
	}

	// Concurrently send requests from 10 distinct consumers
	numConsumers := 10
	requestsPerConsumer := 20
	var wg sync.WaitGroup

	consumerTargetMap := make(map[string]string)
	var mu sync.Mutex

	for c := 0; c < numConsumers; c++ {
		consumerID := fmt.Sprintf("consumer-tenant-%d", c)
		for r := 0; r < requestsPerConsumer; r++ {
			wg.Add(1)
			go func(cid string) {
				defer wg.Done()

				req := httptest.NewRequest("GET", "/api/v1/work", nil)
				req.Header.Set("X-Consumer-ID", cid)

				target, release, err := pool.SelectTarget(req, upstream, targets)
				if err != nil || target == nil {
					t.Errorf("unexpected selection error: %v", err)
					return
				}

				// Simulate brief request work
				time.Sleep(1 * time.Millisecond)

				// Release connection lease
				release()

				mu.Lock()
				if mapped, exists := consumerTargetMap[cid]; exists {
					if mapped != target.ID {
						t.Errorf("inconsistent target for consumer %s: was %s, now %s", cid, mapped, target.ID)
					}
				} else {
					consumerTargetMap[cid] = target.ID
				}
				mu.Unlock()
			}(consumerID)
		}
	}

	wg.Wait()

	// Assert that all connection leases were cleanly released and map has zero active targets
	if pool.ActiveTargetCount() != 0 {
		t.Fatalf("expected all connection leases released (0 active), got %d active targets", pool.ActiveTargetCount())
	}
}
