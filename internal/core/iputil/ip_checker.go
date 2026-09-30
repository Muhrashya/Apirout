package iputil

import (
	"net"
	"net/http"
	"strings"
)

// ExtractClientIP extracts the real client IP from incoming HTTP request.
// It checks standard proxy headers (X-Forwarded-For, X-Real-IP) and falls back to RemoteAddr.
func ExtractClientIP(r *http.Request) string {
	// 1. Check X-Forwarded-For (first IP is the originating client)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		clientIP := strings.TrimSpace(parts[0])
		if clientIP != "" {
			return clientIP
		}
	}

	// 2. Check X-Real-IP
	xrip := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	if xrip != "" {
		return xrip
	}

	// 3. Fallback to RemoteAddr
	remoteHost, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && remoteHost != "" {
		return remoteHost
	}

	return r.RemoteAddr
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
