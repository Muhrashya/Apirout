package egress

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSafeHTTPClient_BlocksRedirectToMetadata(t *testing.T) {
	// Setup test server that redirects to cloud metadata
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://169.254.169.254/latest/meta-data/", http.StatusFound)
	}))
	defer ts.Close()

	// Allow loopback for initial test server connection, but ensure metadata is strictly blocked
	client := NewSafeHTTPClient(2*time.Second, Policy{AllowLoopback: true})

	_, err := client.Get(ts.URL)
	if err == nil {
		t.Fatalf("expected redirect to metadata to fail, got nil error")
	}

	if !strings.Contains(err.Error(), "cloud metadata") && !strings.Contains(err.Error(), "safe egress blocked redirect") {
		t.Fatalf("expected error mentioning metadata SSRF block, got: %v", err)
	}
}

func TestSafeHTTPClient_BlocksRedirectToLoopback(t *testing.T) {
	// Setup test server that redirects to loopback
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://127.0.0.1:8001/api/v1/users", http.StatusFound)
	}))
	defer ts.Close()

	// Strict default policy (no loopback allowed)
	client := NewSafeHTTPClient(2 * time.Second)

	_, err := client.Get(ts.URL)
	if err == nil {
		t.Fatalf("expected connection to fail, got nil error")
	}

	if !strings.Contains(err.Error(), "loopback") && !strings.Contains(err.Error(), "safe egress") {
		t.Fatalf("expected error mentioning loopback SSRF block, got: %v", err)
	}
}
