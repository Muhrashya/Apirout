package responder_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/apirout/apirout/internal/gateway/responder"
)

func TestResponder_Error(t *testing.T) {
	t.Run("Emits hybrid envelope with legacy keys and machine code", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/protected", nil)
		start := time.Now().Add(-5 * time.Millisecond)
		ctx := responder.WithRequestContext(req.Context(), "req-test-trace-123", start)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		extra := map[string]string{
			"Allow": "GET, POST",
		}

		responder.Error(rr, req, http.StatusMethodNotAllowed, responder.ErrCodeMethodNotAllowed, "Method not allowed", extra)

		if rr.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
		}

		// Verify HTTP headers
		if rr.Header().Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json Content-Type, got %q", rr.Header().Get("Content-Type"))
		}
		if rr.Header().Get(responder.HeaderRequestID) != "req-test-trace-123" {
			t.Errorf("expected X-Request-ID req-test-trace-123, got %q", rr.Header().Get(responder.HeaderRequestID))
		}
		if rr.Header().Get("Allow") != "GET, POST" {
			t.Errorf("expected Allow header, got %q", rr.Header().Get("Allow"))
		}
		if rr.Header().Get(responder.HeaderGatewayLatency) == "" {
			t.Errorf("expected X-Gateway-Latency header")
		}
		if rr.Header().Get(responder.HeaderServerTiming) == "" {
			t.Errorf("expected Server-Timing header")
		}

		// Decode payload
		var envelope responder.ErrorEnvelope
		if err := json.Unmarshal(rr.Body.Bytes(), &envelope); err != nil {
			t.Fatalf("failed to decode error envelope: %v", err)
		}

		// Backward-compatible keys
		if envelope.Status != false {
			t.Errorf("expected status false, got %v", envelope.Status)
		}
		if envelope.Message != "Method not allowed" {
			t.Errorf("expected message 'Method not allowed', got %q", envelope.Message)
		}
		if envelope.Error != "Method not allowed" {
			t.Errorf("expected error 'Method not allowed', got %q", envelope.Error)
		}

		// Structured fields
		if envelope.Code != responder.ErrCodeMethodNotAllowed {
			t.Errorf("expected code %s, got %s", responder.ErrCodeMethodNotAllowed, envelope.Code)
		}
		if envelope.RequestID != "req-test-trace-123" {
			t.Errorf("expected request_id req-test-trace-123, got %q", envelope.RequestID)
		}
		if envelope.Timestamp == "" {
			t.Errorf("expected non-empty timestamp")
		}
	})
}

func TestResponder_JSON(t *testing.T) {
	t.Run("Emits standard data representation with telemetry headers", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/users", nil)
		ctx := responder.WithRequestContext(req.Context(), "req-success-456", time.Now().Add(-2*time.Millisecond))
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		payload := map[string]string{"result": "ok"}

		responder.JSON(rr, req, http.StatusOK, payload)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}
		if rr.Header().Get(responder.HeaderRequestID) != "req-success-456" {
			t.Errorf("expected X-Request-ID req-success-456, got %q", rr.Header().Get(responder.HeaderRequestID))
		}
		if rr.Header().Get(responder.HeaderServerTiming) == "" {
			t.Errorf("expected Server-Timing header")
		}

		var decoded map[string]string
		if err := json.Unmarshal(rr.Body.Bytes(), &decoded); err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}
		if decoded["result"] != "ok" {
			t.Errorf("expected result ok, got %q", decoded["result"])
		}
	})
}
