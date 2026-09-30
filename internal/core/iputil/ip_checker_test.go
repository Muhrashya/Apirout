package iputil

import (
	"net/http/httptest"
	"testing"
)

func TestExtractClientIP(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"

	// Fallback to RemoteAddr
	if ip := ExtractClientIP(req); ip != "10.0.0.1" {
		t.Fatalf("expected 10.0.0.1, got %s", ip)
	}

	// X-Real-IP
	req.Header.Set("X-Real-IP", "192.168.1.50")
	if ip := ExtractClientIP(req); ip != "192.168.1.50" {
		t.Fatalf("expected 192.168.1.50, got %s", ip)
	}

	// X-Forwarded-For overrides X-Real-IP
	req.Header.Set("X-Forwarded-For", "203.0.113.195, 10.0.0.2")
	if ip := ExtractClientIP(req); ip != "203.0.113.195" {
		t.Fatalf("expected 203.0.113.195, got %s", ip)
	}
}

func TestIsIPAllowed(t *testing.T) {
	tests := []struct {
		clientIP string
		rules    string
		expected bool
	}{
		{"127.0.0.1", "", true},
		{"127.0.0.1", "*", true},
		{"192.168.1.5", "192.168.1.5", true},
		{"192.168.1.6", "192.168.1.5", false},
		{"192.168.1.42", "192.168.1.0/24", true},
		{"192.168.2.42", "192.168.1.0/24", false},
		{"10.20.30.40", "192.168.1.0/24, 10.0.0.0/8, 172.16.0.1", true},
		{"8.8.8.8", "192.168.1.0/24, 10.0.0.0/8", false},
	}

	for _, tt := range tests {
		got := IsIPAllowed(tt.clientIP, tt.rules)
		if got != tt.expected {
			t.Errorf("IsIPAllowed(%q, %q) = %v; want %v", tt.clientIP, tt.rules, got, tt.expected)
		}
	}
}
