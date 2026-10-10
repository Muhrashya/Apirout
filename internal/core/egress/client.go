package egress

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// NewSafeHTTPClient creates an http.Client configured with SSRF protection on connection dial
// and on every redirect hop (301/302/307/308).
func NewSafeHTTPClient(timeout time.Duration, opts ...Policy) *http.Client {
	policy := Policy{}
	if len(opts) > 0 {
		policy = opts[0]
	}

	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				host = addr
			}
			if err := ValidateSafeHost(host, policy); err != nil {
				return nil, fmt.Errorf("safe egress blocked destination %s: %w", host, err)
			}
			return dialer.DialContext(ctx, network, addr)
		},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			if err := ValidateSafeURL(req.URL.String(), policy); err != nil {
				return fmt.Errorf("safe egress blocked redirect to %s: %w", req.URL.String(), err)
			}
			return nil
		},
	}
}
