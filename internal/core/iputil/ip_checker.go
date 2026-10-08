package iputil

import (
	"net"
	"net/http"
	"strings"
	"sync/atomic"
)

// TrustedProxyChecker evaluates whether an IP address belongs to an authorized proxy network.
type TrustedProxyChecker struct {
	rules string
}

// NewTrustedProxyChecker creates a new checker with the given comma-separated CIDR/IP rules.
func NewTrustedProxyChecker(rules string) *TrustedProxyChecker {
	return &TrustedProxyChecker{rules: strings.TrimSpace(rules)}
}

// IsTrusted returns true if the candidate IP satisfies the trusted rules.
func (c *TrustedProxyChecker) IsTrusted(ip string) bool {
	if c == nil || c.rules == "" {
		return false
	}
	return IsIPAllowed(ip, c.rules)
}

var globalTrustedChecker atomic.Pointer[TrustedProxyChecker]

func init() {
	SetGlobalTrustedProxies("127.0.0.1/32,::1/128,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16")
}

// SetGlobalTrustedProxies configures the global proxy policy for ExtractClientIP and httputil.GetClientIP
func SetGlobalTrustedProxies(rules string) {
	globalTrustedChecker.Store(NewTrustedProxyChecker(rules))
}

// GetGlobalTrustedChecker retrieves the currently configured global checker
func GetGlobalTrustedChecker() *TrustedProxyChecker {
	return globalTrustedChecker.Load()
}

// ExtractClientIP extracts the real client IP from incoming HTTP request using the global trusted proxy policy.
func ExtractClientIP(r *http.Request) string {
	return ResolveClientIP(r, GetGlobalTrustedChecker())
}

// ResolveClientIP evaluates client IP against a specific TrustedProxyChecker.
// If RemoteAddr is not a trusted proxy, it strictly returns RemoteAddr and discards all proxy headers.
// If RemoteAddr is trusted, it traverses X-Forwarded-For from right to left, skipping trusted proxies
// to locate the first untrusted client IP.
func ResolveClientIP(r *http.Request, checker *TrustedProxyChecker) string {
	remoteHost, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil || remoteHost == "" {
		remoteHost = r.RemoteAddr
	}
	remoteHost = strings.TrimSpace(remoteHost)

	// If remote socket address is not a trusted proxy, proxy headers cannot be trusted.
	if checker == nil || !checker.IsTrusted(remoteHost) {
		return remoteHost
	}

	// 1. Right-to-left traversal of X-Forwarded-For
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		for i := len(parts) - 1; i >= 0; i-- {
			candidate := strings.TrimSpace(parts[i])
			if candidate == "" {
				continue
			}
			if parsed := net.ParseIP(candidate); parsed != nil {
				if !checker.IsTrusted(candidate) {
					return candidate
				}
			}
		}
	}

	// 2. Check X-Real-IP
	xrip := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	if xrip != "" {
		if parsed := net.ParseIP(xrip); parsed != nil {
			if !checker.IsTrusted(xrip) {
				return xrip
			}
		}
	}

	// 3. Check vendor-specific proxy headers
	for _, header := range []string{"CF-Connecting-IP", "Fastly-Client-IP", "True-Client-IP"} {
		if val := strings.TrimSpace(r.Header.Get(header)); val != "" {
			if parsed := net.ParseIP(val); parsed != nil {
				if !checker.IsTrusted(val) {
					return val
				}
			}
		}
	}

	return remoteHost
}

// IsIPAllowed verifies whether clientIP satisfies the allowedRules filter.
// allowedRules is a comma-separated list of individual IPs (e.g. 192.168.1.100),
// CIDR blocks (e.g. 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16), or "*" (allow all).
// If allowedRules is blank or "*", access is unconditionally allowed.
func IsIPAllowed(clientIP, allowedRules string) bool {
	cleanRules := strings.TrimSpace(allowedRules)
	if cleanRules == "" || cleanRules == "*" {
		return true
	}

	parsedClient := net.ParseIP(strings.TrimSpace(clientIP))
	if parsedClient == nil {
		return false
	}

	rules := strings.Split(cleanRules, ",")
	for _, rawRule := range rules {
		rule := strings.TrimSpace(rawRule)
		if rule == "" {
			continue
		}
		if rule == "*" {
			return true
		}

		// CIDR Notation
		if strings.Contains(rule, "/") {
			_, ipNet, err := net.ParseCIDR(rule)
			if err == nil && ipNet.Contains(parsedClient) {
				return true
			}
			continue
		}

		// Exact IP Match
		targetIP := net.ParseIP(rule)
		if targetIP != nil && targetIP.Equal(parsedClient) {
			return true
		}
	}

	return false
}
