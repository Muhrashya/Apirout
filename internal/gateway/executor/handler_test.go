package executor

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/apirout/apirout/internal/core/models"
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
