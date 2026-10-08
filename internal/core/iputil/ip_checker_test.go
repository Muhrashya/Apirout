package iputil

import (
	"net/http/httptest"
	"testing"
)

func TestExtractClientIP_TrustedAndUntrusted(t *testing.T) {
	// 1. Untrusted remote address: headers MUST be ignored
	untrustedChecker := NewTrustedProxyChecker("10.0.0.0/8")
	reqUntrusted := httptest.NewRequest("GET", "/", nil)
	reqUntrusted.RemoteAddr = "198.51.100.25:44321" // Untrusted public IP
	reqUntrusted.Header.Set("X-Forwarded-For", "10.0.0.99, 127.0.0.1")
	reqUntrusted.Header.Set("X-Real-IP", "10.0.0.99")

	if ip := ResolveClientIP(reqUntrusted, untrustedChecker); ip != "198.51.100.25" {
		t.Fatalf("expected direct socket IP 198.51.100.25 for untrusted remote, got: %s", ip)
	}

	// 2. Trusted proxy remote address: traverse X-Forwarded-For right-to-left
	trustedChecker := NewTrustedProxyChecker("10.0.0.0/8, 172.16.0.0/12")
	reqTrusted := httptest.NewRequest("GET", "/", nil)
	reqTrusted.RemoteAddr = "10.0.0.1:12345" // Trusted proxy socket
	// Chain: Client (203.0.113.50) -> Proxy1 (172.16.5.10) -> Ingress (10.0.0.1)
	reqTrusted.Header.Set("X-Forwarded-For", "203.0.113.50, 172.16.5.10")

	if ip := ResolveClientIP(reqTrusted, trustedChecker); ip != "203.0.113.50" {
		t.Fatalf("expected real untrusted client 203.0.113.50, got: %s", ip)
	}

	// 3. Trusted proxy with multiple trusted hops: should correctly skip all trusted hops
	reqMultiHop := httptest.NewRequest("GET", "/", nil)
	reqMultiHop.RemoteAddr = "10.0.0.1:12345"
	reqMultiHop.Header.Set("X-Forwarded-For", "198.51.100.77, 10.0.0.5, 172.16.1.1")
	if ip := ResolveClientIP(reqMultiHop, trustedChecker); ip != "198.51.100.77" {
		t.Fatalf("expected real client 198.51.100.77, got: %s", ip)
	}

	// 4. Fallback to X-Real-IP if X-Forwarded-For is missing
	reqRealIP := httptest.NewRequest("GET", "/", nil)
	reqRealIP.RemoteAddr = "10.0.0.1:12345"
	reqRealIP.Header.Set("X-Real-IP", "198.51.100.88")
	if ip := ResolveClientIP(reqRealIP, trustedChecker); ip != "198.51.100.88" {
		t.Fatalf("expected X-Real-IP 198.51.100.88, got: %s", ip)
	}

	// 5. Vendor-specific headers when trusted
	reqCF := httptest.NewRequest("GET", "/", nil)
	reqCF.RemoteAddr = "10.0.0.1:12345"
	reqCF.Header.Set("CF-Connecting-IP", "198.51.100.99")
	if ip := ResolveClientIP(reqCF, trustedChecker); ip != "198.51.100.99" {
		t.Fatalf("expected CF-Connecting-IP 198.51.100.99, got: %s", ip)
	}

	// 6. Vendor headers ignored when untrusted
	reqCFUntrusted := httptest.NewRequest("GET", "/", nil)
	reqCFUntrusted.RemoteAddr = "198.51.100.25:44321"
	reqCFUntrusted.Header.Set("CF-Connecting-IP", "198.51.100.99")
	if ip := ResolveClientIP(reqCFUntrusted, untrustedChecker); ip != "198.51.100.25" {
		t.Fatalf("expected socket IP 198.51.100.25, got: %s", ip)
	}

	// 7. Invalid/garbage IP in headers should be ignored
	reqGarbage := httptest.NewRequest("GET", "/", nil)
	reqGarbage.RemoteAddr = "10.0.0.1:12345"
	reqGarbage.Header.Set("X-Forwarded-For", "invalid-ip-string, 10.0.0.5")
	if ip := ResolveClientIP(reqGarbage, trustedChecker); ip != "10.0.0.1" {
		t.Fatalf("expected fallback to 10.0.0.1 when garbage IP in XFF, got: %s", ip)
	}

	// 8. Global helper ExtractClientIP using global checker
	SetGlobalTrustedProxies("10.0.0.0/8")
	reqGlobal := httptest.NewRequest("GET", "/", nil)
	reqGlobal.RemoteAddr = "10.0.0.2:5000"
	reqGlobal.Header.Set("X-Forwarded-For", "203.0.113.195")
	if ip := ExtractClientIP(reqGlobal); ip != "203.0.113.195" {
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
