package executor

import (
	"net/http"
	"net/http/httptest"
	"testing"
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
