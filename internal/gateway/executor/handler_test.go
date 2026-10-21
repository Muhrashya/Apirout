package executor

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/apirout/apirout/internal/core/models"
	"github.com/apirout/apirout/internal/core/router"
	"github.com/apirout/apirout/internal/core/upstream"
	"github.com/apirout/apirout/internal/gateway/pipeline"
)

func TestExtractTableAndID(t *testing.T) {
	tests := []struct {
		name        string
		routePath   string
		requestPath string
		servicePath string
		wantTable   string
		wantID      string
	}{
		{
			name:        "Exact match single table",
			routePath:   "/api/v1/users",
			requestPath: "/api/v1/users",
			servicePath: "",
			wantTable:   "users",
			wantID:      "",
		},
		{
			name:        "Subpath with numeric ID",
			routePath:   "/api/v1/users",
			requestPath: "/api/v1/users/42",
			servicePath: "",
			wantTable:   "users",
			wantID:      "42",
		},
		{
			name:        "Subpath with string UUID",
			routePath:   "/api/v1/customers",
			requestPath: "/api/v1/customers/c8d45f31-89a1",
			servicePath: "",
			wantTable:   "customers",
			wantID:      "c8d45f31-89a1",
		},
		{
			name:        "Route with variable placeholder",
			routePath:   "/api/v1/orders/:id",
			requestPath: "/api/v1/orders/100",
			servicePath: "",
			wantTable:   "orders",
			wantID:      "100",
		},
		{
			name:        "Explicit service path override",
			routePath:   "/api/store/accounts",
			requestPath: "/api/store/accounts/99",
			servicePath: "/bank_accounts",
			wantTable:   "bank_accounts",
			wantID:      "99",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTable, gotID := extractTableAndID(tt.routePath, tt.requestPath, tt.servicePath)
			if gotTable != tt.wantTable {
				t.Errorf("extractTableAndID table = %q, want %q", gotTable, tt.wantTable)
			}
			if gotID != tt.wantID {
				t.Errorf("extractTableAndID id = %q, want %q", gotID, tt.wantID)
			}
		})
	}
}

func TestTokenBucketLimiter(t *testing.T) {
	limiter := newTokenBucketLimiter()

	key := "test-route-1"
	rpm := 60
	burst := 2

	// First request within burst
	if !limiter.Allow(key, rpm, burst) {
		t.Errorf("expected 1st request to be allowed")
	}

	// Second request within burst
	if !limiter.Allow(key, rpm, burst) {
		t.Errorf("expected 2nd request to be allowed")
	}

	// Third request exceeds burst (rpm=60 means 1 token/sec, so immediate 3rd call must be rejected)
	if limiter.Allow(key, rpm, burst) {
		t.Errorf("expected 3rd immediate request to be rejected")
	}
}

func TestStatusRecorder(t *testing.T) {
	rec := httptest.NewRecorder()
	sr := &statusRecorder{ResponseWriter: rec, statusCode: http.StatusOK}

	sr.WriteHeader(http.StatusCreated)
	if sr.statusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", sr.statusCode)
	}

	data := []byte(`{"created":true}`)
	n, err := sr.Write(data)
	if err != nil || n != len(data) {
		t.Errorf("write error or unexpected byte count")
	}
	if sr.bytesWritten != int64(len(data)) {
		t.Errorf("expected bytesWritten %d, got %d", len(data), sr.bytesWritten)
	}
}

func TestIsDomainAllowed(t *testing.T) {
	tests := []struct {
		host           string
		allowedDomains string
		want           bool
	}{
		{"api.example.com", "*", true},
		{"api.example.com", "", true},
		{"api.example.com", "api.example.com", true},
		{"api.example.com", "api.other.com, api.example.com", true},
		{"sub.api.example.com", "*.example.com", true},
		{"hacker.com", "api.example.com, internal.net", false},
	}

	for _, tt := range tests {
		got := isDomainAllowed(tt.host, tt.allowedDomains)
		if got != tt.want {
			t.Errorf("isDomainAllowed(%q, %q) = %v, want %v", tt.host, tt.allowedDomains, got, tt.want)
		}
	}
}

func TestIsOriginAllowed(t *testing.T) {
	tests := []struct {
		origin         string
		allowedOrigins string
		want           bool
	}{
		{"https://app.client.com", "*", true},
		{"https://app.client.com", "", true},
		{"https://app.client.com", "https://app.client.com", true},
		{"https://app.client.com", "https://other.com, https://app.client.com", true},
		{"https://evil.com", "https://app.client.com", false},
	}

	for _, tt := range tests {
		got := isOriginAllowed(tt.origin, tt.allowedOrigins)
		if got != tt.want {
			t.Errorf("isOriginAllowed(%q, %q) = %v, want %v", tt.origin, tt.allowedOrigins, got, tt.want)
		}
	}
}

func TestStageAuth_RateLimiterAnonymousIPIsolation(t *testing.T) {
	h := &Handler{
		rateLimiter: newTokenBucketLimiter(),
	}
	stage := newAuthStage(h)

	endpoint := &models.Endpoint{
		ID:               "ep-test-1",
		RateLimitEnabled: true,
		RateLimitRPM:     60,
		RateLimitBurst:   1, // Only 1 allowed immediately
	}

	// 1. First anonymous request from IP 192.168.1.10
	req1 := httptest.NewRequest("GET", "/api/v1/test", nil)
	rec1 := httptest.NewRecorder()
	ctx1 := pipeline.NewContext(rec1, req1)
	ctx1.ClientIP = "192.168.1.10"
	ctx1.Endpoint = endpoint
	_ = stage.Execute(ctx1)
	if ctx1.StatusCode == http.StatusTooManyRequests {
		t.Fatalf("first request from IP1 should be allowed, got 429")
	}

	// 2. Second immediate request from IP 192.168.1.10 should exceed burst=1
	req2 := httptest.NewRequest("GET", "/api/v1/test", nil)
	rec2 := httptest.NewRecorder()
	ctx2 := pipeline.NewContext(rec2, req2)
	ctx2.ClientIP = "192.168.1.10"
	ctx2.Endpoint = endpoint
	_ = stage.Execute(ctx2)
	if ctx2.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("second immediate request from IP1 should be rate-limited, got status %d", ctx2.StatusCode)
	}

	// 3. First request from different IP 192.168.1.20 should be ALLOWED (independent bucket)
	req3 := httptest.NewRequest("GET", "/api/v1/test", nil)
	rec3 := httptest.NewRecorder()
	ctx3 := pipeline.NewContext(rec3, req3)
	ctx3.ClientIP = "192.168.1.20"
	ctx3.Endpoint = endpoint
	_ = stage.Execute(ctx3)
	if ctx3.StatusCode == http.StatusTooManyRequests {
		t.Fatalf("request from distinct IP2 should have its own token bucket and be allowed, got 429")
	}
}

func TestStageIngress_PreflightShortCircuitAndIPv6(t *testing.T) {
	h := &Handler{}

	ep := models.Endpoint{
		ID:          "ep-orders-post",
		Path:        "/api/v1/orders",
		Method:      "POST",
		CorsOrigins: "*",
		Enabled:     true,
	}
	rtr := router.BuildFromEndpoints([]models.Endpoint{ep})
	h.SetRouter(rtr)

	stage := newIngressStage(h)

	// 1. Preflight OPTIONS request for a POST route with IPv6 Host literal
	req := httptest.NewRequest("OPTIONS", "/api/v1/orders", nil)
	req.Host = "[::1]:8000"
	rec := httptest.NewRecorder()
	ctx := pipeline.NewContext(rec, req)

	err := stage.Execute(ctx)
	if err != nil {
		t.Fatalf("unexpected error executing stage: %v", err)
	}

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected preflight status 204 No Content, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	allowMethods := rec.Header().Get("Access-Control-Allow-Methods")
	if !strings.Contains(allowMethods, "POST") {
		t.Errorf("expected Access-Control-Allow-Methods to include POST, got %q", allowMethods)
	}

	allowHeaders := rec.Header().Get("Access-Control-Allow-Headers")
	if allowHeaders == "" {
		t.Errorf("expected Access-Control-Allow-Headers to be set")
	}

	// 2. Non-existent route OPTIONS request should still return 404
	req404 := httptest.NewRequest("OPTIONS", "/api/v1/nonexistent", nil)
	rec404 := httptest.NewRecorder()
	ctx404 := pipeline.NewContext(rec404, req404)
	_ = stage.Execute(ctx404)
	if rec404.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown route preflight, got %d", rec404.Code)
	}
}

func TestStripStaticRoutePrefix(t *testing.T) {
	tests := []struct {
		name        string
		requestPath string
		routePath   string
		want        string
	}{
		{
			name:        "Wildcard trailing star",
			requestPath: "/api/v1/users/profile",
			routePath:   "/api/v1/users/*",
			want:        "/profile",
		},
		{
			name:        "Wildcard exact root match",
			requestPath: "/api/v1/users",
			routePath:   "/api/v1/users/*",
			want:        "/",
		},
		{
			name:        "Parameterized route with ID",
			requestPath: "/api/v1/orders/42",
			routePath:   "/api/v1/orders/:id",
			want:        "/42",
		},
		{
			name:        "Static route without tokens",
			requestPath: "/api/v1/checkout/pay",
			routePath:   "/api/v1/checkout",
			want:        "/pay",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripStaticRoutePrefix(tt.requestPath, tt.routePath)
			if got != tt.want {
				t.Errorf("stripStaticRoutePrefix(%q, %q) = %q, want %q", tt.requestPath, tt.routePath, got, tt.want)
			}
		})
	}
}

func TestDirectBackend_CircuitBreakerTripping(t *testing.T) {
	cb := upstream.NewPassiveCircuitBreaker(3, 10*time.Second)
	serviceID := "svc-direct-123"
	syntheticID := "direct:" + serviceID

	// 1. Initially healthy
	if !cb.Allow(syntheticID) {
		t.Fatalf("expected direct service to be allowed initially")
	}

	// 2. Record 2 failures (< threshold of 3)
	cb.RecordFailure(syntheticID)
	cb.RecordFailure(syntheticID)
	if !cb.Allow(syntheticID) {
		t.Fatalf("expected direct service to still be allowed after 2 failures")
	}

	// 3. 3rd failure reaches threshold, should trip circuit breaker
	cb.RecordFailure(syntheticID)
	if cb.Allow(syntheticID) {
		t.Fatalf("expected direct service circuit breaker to trip after 3 failures")
	}
}

func TestReverseProxy_WildcardPathStrippingEndToEnd(t *testing.T) {
	var receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	host, portStr, _ := net.SplitHostPort(u.Host)
	port := 80
	if p, err := strconv.Atoi(portStr); err == nil {
		port = p
	}

	h := &Handler{
		circuitBreaker: upstream.NewPassiveCircuitBreaker(3, 15*time.Second),
	}

	ep := &models.Endpoint{
		Name:      "Wildcard Route",
		Path:      "/api/v1/users/*",
		StripPath: true,
	}

	svc := &models.Service{
		ID:       "svc-direct-test",
		Name:     "Direct Service",
		Protocol: "http",
		Host:     host,
		Port:     port,
		Path:     "",
	}

	req := httptest.NewRequest("GET", "/api/v1/users/profile", nil)
	rec := httptest.NewRecorder()

	h.reverseProxyRequest(rec, req, ep, svc, host, port, "direct:svc-direct-test", nil, nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from reverse proxy, got %d", rec.Code)
	}

	if receivedPath != "/profile" {
		t.Errorf("expected upstream to receive path '/profile', got %q", receivedPath)
	}
}


