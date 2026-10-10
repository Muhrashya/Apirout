package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/apirout/apirout/internal/core/crypto"
	"github.com/apirout/apirout/internal/core/models"
)

func TestOAuth2CCManager_TokenExchangeAndCache(t *testing.T) {
	var requestCount int32
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)

		if r.Method != "POST" {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		if r.FormValue("grant_type") != "client_credentials" {
			http.Error(w, "invalid grant_type", http.StatusBadRequest)
			return
		}

		u, p, ok := r.BasicAuth()
		if !ok || u != "test-client" || p != "test-secret" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "token-xyz-12345",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer mockServer.Close()

	manager := NewOAuth2CCManager(mockServer.Client())
	manager.AllowInsecureEgress = true

	// 1. Initial exchange
	token, err := manager.GetToken(mockServer.URL, "test-client", "test-secret", "read write")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "token-xyz-12345" {
		t.Errorf("got token %s, want token-xyz-12345", token)
	}
	if atomic.LoadInt32(&requestCount) != 1 {
		t.Errorf("expected 1 upstream call, got %d", requestCount)
	}

	// 2. Immediate second call should hit memory cache (no additional upstream request)
	cachedToken, err := manager.GetToken(mockServer.URL, "test-client", "test-secret", "read write")
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}
	if cachedToken != "token-xyz-12345" {
		t.Errorf("got cached token %s, want token-xyz-12345", cachedToken)
	}
	if atomic.LoadInt32(&requestCount) != 1 {
		t.Errorf("expected still 1 upstream call due to cache, got %d", requestCount)
	}
}

func TestOAuth2CCManager_ProactiveRefreshBuffer(t *testing.T) {
	var requestCount int32
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": fmt.Sprintf("token-v%d", count),
			"token_type":   "Bearer",
			"expires_in":   70, // 70 seconds total lifetime
		})
	}))
	defer mockServer.Close()

	manager := NewOAuth2CCManager(mockServer.Client())
	manager.AllowInsecureEgress = true

	token1, err := manager.GetToken(mockServer.URL, "client", "secret", "")
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	if token1 != "token-v1" {
		t.Errorf("got %s, want token-v1", token1)
	}

	// Manually age the cached token so only 30 seconds of validity remain (< 60s proactive buffer)
	cacheKey := fmt.Sprintf("%s|%s|%s", mockServer.URL, "client", "")
	manager.cacheMu.Lock()
	if c, ok := manager.cache[cacheKey]; ok {
		c.expiresAt = time.Now().Add(30 * time.Second)
	}
	manager.cacheMu.Unlock()

	// Calling GetToken must proactively refresh because remaining validity < 60s
	token2, err := manager.GetToken(mockServer.URL, "client", "secret", "")
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}
	if token2 != "token-v2" {
		t.Errorf("got %s, want proactively refreshed token-v2", token2)
	}
	if atomic.LoadInt32(&requestCount) != 2 {
		t.Errorf("expected 2 upstream calls due to proactive refresh, got %d", requestCount)
	}
}

func TestOAuth2CCManager_SingleflightDeduplication(t *testing.T) {
	var requestCount int32
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		// Introduce artificial delay to test concurrency coalescing
		time.Sleep(50 * time.Millisecond)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "token-singleflight-winner",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer mockServer.Close()

	manager := NewOAuth2CCManager(mockServer.Client())
	manager.AllowInsecureEgress = true

	concurrency := 20
	var wg sync.WaitGroup
	wg.Add(concurrency)

	tokens := make([]string, concurrency)
	errors := make([]error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(idx int) {
			defer wg.Done()
			tokens[idx], errors[idx] = manager.GetToken(mockServer.URL, "concurrent-client", "secret", "scope1")
		}(i)
	}

	wg.Wait()

	for i := 0; i < concurrency; i++ {
		if errors[i] != nil {
			t.Fatalf("goroutine %d failed: %v", i, errors[i])
		}
		if tokens[i] != "token-singleflight-winner" {
			t.Errorf("goroutine %d got token %s, want token-singleflight-winner", i, tokens[i])
		}
	}

	// Singleflight must have coalesced all concurrent calls into 1 upstream HTTP request
	if atomic.LoadInt32(&requestCount) != 1 {
		t.Errorf("expected exactly 1 upstream request due to singleflight deduplication, got %d", requestCount)
	}
}

func TestOutboundInjector_OAuth2CC_Negotiator(t *testing.T) {
	encKey := "test-32-byte-encryption-key-1234"
	normKey := crypto.NormalizeKey(encKey)
	injector := NewOutboundInjector(encKey)

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "token-injected-bearer-999",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer mockServer.Close()

	ccMgr := NewOAuth2CCManager(mockServer.Client())
	ccMgr.AllowInsecureEgress = true
	injector.SetOAuth2CCManager(ccMgr)

	plainSecret := "real-oauth-client-secret"
	encSecret, _ := crypto.EncryptPassword(normKey, plainSecret)

	auth := &models.ServiceOutboundAuth{
		AuthType: models.OutboundOAuth2CC,
		Enabled:  true,
		Config: fmt.Sprintf(`{
			"token_url": "%s",
			"client_id": "api-service-app",
			"client_secret_encrypted": "%s",
			"scopes": "analytics read"
		}`, mockServer.URL, encSecret),
	}

	req := httptest.NewRequest("GET", "http://backend.internal/resource", nil)
	// Add legacy header that must be purged
	req.Header.Set("X-Apirout-OAuth2-Client", "legacy-client-id")

	var negotiator OutboundNegotiator = injector
	if err := negotiator.Negotiate(req, auth); err != nil {
		t.Fatalf("negotiator.Negotiate error: %v", err)
	}

	// 1. Verify legacy pseudo-header purged
	if req.Header.Get("X-Apirout-OAuth2-Client") != "" {
		t.Errorf("expected X-Apirout-OAuth2-Client to be purged")
	}

	// 2. Verify authentic Authorization: Bearer <token> injected
	authHeader := req.Header.Get("Authorization")
	if authHeader != "Bearer token-injected-bearer-999" {
		t.Errorf("got %s, want Bearer token-injected-bearer-999", authHeader)
	}
}

func TestOAuth2CCManager_SSRFProtection(t *testing.T) {
	manager := NewOAuth2CCManager()

	blockedURLs := []string{
		"http://127.0.0.1:8080/token",
		"http://localhost:8080/token",
		"http://169.254.169.254/latest/meta-data/",
		"http://10.0.0.1/oauth/token",
		"http://172.16.0.1/oauth/token",
		"http://192.168.1.1/oauth/token",
	}

	for _, u := range blockedURLs {
		_, err := manager.GetToken(u, "client", "secret", "")
		if err == nil {
			t.Errorf("expected SSRF error for token URL %q, got nil", u)
		}
	}
}

