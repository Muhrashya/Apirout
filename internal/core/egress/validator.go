package egress

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

var (
	ErrBlockedLoopback = errors.New("egress to loopback address is prohibited")
	ErrBlockedPrivate  = errors.New("egress to private network is prohibited")
	ErrBlockedMetadata = errors.New("egress to cloud metadata service is prohibited")
	ErrBlockedInvalid  = errors.New("invalid egress destination")
)

var (
	metadataIPs = []string{
		"169.254.169.254", // AWS, GCP, Azure, OpenStack
		"100.100.100.200", // Alibaba Cloud
	}
	metadataHosts = []string{
		"metadata.google.internal",
		"metadata.internal",
	}
)

// Policy configures egress restrictions. Defaults to blocking loopback, private, and metadata.
type Policy struct {
	AllowPrivate  bool
	AllowLoopback bool
}

// ValidateSafeEgress is an alias for ValidateSafeHost for egress boundary enforcement.
func ValidateSafeEgress(rawHost string, opts ...Policy) error {
	return ValidateSafeHost(rawHost, opts...)
}

// ValidateSafeHost parses and validates a hostname or IP string against SSRF rules.
func ValidateSafeHost(rawHost string, opts ...Policy) error {
	rawHost = strings.TrimSpace(rawHost)
	if rawHost == "" {
		return ErrBlockedInvalid
	}

	policy := Policy{}
	if len(opts) > 0 {
		policy = opts[0]
	}

	// Strip port if present
	host := rawHost
	if h, _, err := net.SplitHostPort(rawHost); err == nil {
		host = h
	}

	hostLower := strings.ToLower(host)
	if !policy.AllowLoopback && (hostLower == "localhost" || hostLower == "127.0.0.1" || hostLower == "::1" || hostLower == "0.0.0.0") {
		return ErrBlockedLoopback
	}

	for _, mh := range metadataHosts {
		if hostLower == mh {
			return ErrBlockedMetadata
		}
	}

	for _, mip := range metadataIPs {
		if host == mip {
			return ErrBlockedMetadata
		}
	}

	// Try parsing directly as IP
	if ip := net.ParseIP(host); ip != nil {
		return checkIP(ip, policy)
	}

	// Resolve hostname to IPs
	ips, err := net.LookupIP(host)
	if err != nil {
		if !policy.AllowPrivate && (strings.HasSuffix(hostLower, ".internal") || strings.HasSuffix(hostLower, ".local")) {
			return ErrBlockedPrivate
		}
		return nil
	}

	for _, ip := range ips {
		if err := checkIP(ip, policy); err != nil {
			return err
		}
	}

	return nil
}

// ValidateSafeURL parses a full URL and checks its destination host against SSRF rules.
func ValidateSafeURL(rawURL string, opts ...Policy) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ErrBlockedInvalid
	}

	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return fmt.Errorf("%w: cannot parse URL", ErrBlockedInvalid)
	}

	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("%w: unsupported scheme %q", ErrBlockedInvalid, u.Scheme)
	}

	return ValidateSafeHost(u.Host, opts...)
}

func checkIP(ip net.IP, policy Policy) error {
	if !policy.AllowLoopback && (ip.IsLoopback() || ip.IsUnspecified()) {
		return ErrBlockedLoopback
	}

	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return ErrBlockedMetadata
	}

	ipStr := ip.String()
	for _, mip := range metadataIPs {
		if ipStr == mip {
			return ErrBlockedMetadata
		}
	}

	if !policy.AllowPrivate && ip.IsPrivate() {
		return ErrBlockedPrivate
	}

	if !policy.AllowPrivate && isCGNAT(ip) {
		return ErrBlockedPrivate
	}

	return nil
}

func isCGNAT(ip net.IP) bool {
	if ip4 := ip.To4(); ip4 != nil {
		return ip4[0] == 100 && (ip4[1]&0xc0) == 64
	}
	return false
}
