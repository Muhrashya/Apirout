package responder_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/apirout/apirout/internal/core/upstream"
	"github.com/apirout/apirout/internal/gateway/responder"
)

func TestProxy_HandleProxyError(t *testing.T) {
	t.Run("Sanitizes internal network error and records failure", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		ctx := responder.WithRequestContext(req.Context(), "req-proxy-err-123", time.Now().Add(-15*time.Millisecond))
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		cb := upstream.NewPassiveCircuitBreaker(3, 10*time.Second)
		targetID := "tgt-dead-1"
		rawErr := errors.New("dial tcp 127.0.0.1:59199: connect: connection refused")

		var upDur int
		upStart := time.Now().Add(-10 * time.Millisecond)

		responder.HandleProxyError(rr, req, "PaymentsService", rawErr, targetID, cb, &upDur, upStart)

		if rr.Code != http.StatusBadGateway {
			t.Fatalf("expected 502 Bad Gateway, got %d", rr.Code)
		}

		body := rr.Body.String()
		if strings.Contains(body, "127.0.0.1:59199") {
			t.Errorf("expected sanitized error without internal IP, got: %s", body)
		}
		if !strings.Contains(body, "UPSTREAM_UNREACHABLE") {
			t.Errorf("expected UPSTREAM_UNREACHABLE error code, got: %s", body)
		}
		if !strings.Contains(body, "Backend service PaymentsService unreachable") {
			t.Errorf("expected user-facing service message, got: %s", body)
		}

		if upDur <= 0 {
			t.Errorf("expected upstream duration recorded, got %d", upDur)
		}
	})
}

func TestProxy_ModifyProxyResponse(t *testing.T) {
	t.Run("Strips sensitive server banners and adds Server-Timing", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		ctx := responder.WithRequestContext(req.Context(), "req-proxy-ok-456", time.Now().Add(-5*time.Millisecond))
		req = req.WithContext(ctx)

		res := &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
		}
		res.Header.Set("Server", "Apache/2.4.41 (Ubuntu)")
		res.Header.Set("X-Powered-By", "PHP/7.4.3")
		res.Header.Set("Content-Type", "application/json")

		cb := upstream.NewPassiveCircuitBreaker(3, 10*time.Second)
		targetID := "tgt-healthy-1"
		var upDur int
		upStart := time.Now().Add(-3 * time.Millisecond)

		err := responder.ModifyProxyResponse(req, res, targetID, cb, &upDur, upStart)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify stripped headers
		if res.Header.Get("Server") != "" {
			t.Errorf("expected Server header stripped, got %q", res.Header.Get("Server"))
		}
		if res.Header.Get("X-Powered-By") != "" {
			t.Errorf("expected X-Powered-By stripped, got %q", res.Header.Get("X-Powered-By"))
		}

		// Verify injected headers
		if res.Header.Get(responder.HeaderRequestID) != "req-proxy-ok-456" {
			t.Errorf("expected X-Request-ID req-proxy-ok-456, got %q", res.Header.Get(responder.HeaderRequestID))
		}
		if !strings.Contains(res.Header.Get(responder.HeaderServerTiming), "upstream;dur=") {
			t.Errorf("expected Server-Timing with upstream duration, got %q", res.Header.Get(responder.HeaderServerTiming))
		}
		if res.Header.Get(responder.HeaderGatewayLatency) == "" {
			t.Errorf("expected X-Gateway-Latency header")
		}
	})
}
