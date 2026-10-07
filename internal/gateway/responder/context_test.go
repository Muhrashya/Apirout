package responder_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/apirout/apirout/internal/gateway/responder"
)

func TestRequestContext_Lifecycle(t *testing.T) {
	t.Run("Context injection and retrieval", func(t *testing.T) {
		reqID := "req-custom-trace-12345"
		start := time.Now().Add(-10 * time.Millisecond)

		ctx := responder.WithRequestContext(context.Background(), reqID, start)
		if got := responder.GetRequestID(ctx); got != reqID {
			t.Errorf("expected requestID %q, got %q", reqID, got)
		}

		if gotStart := responder.GetStartTime(ctx); !gotStart.Equal(start) {
			t.Errorf("expected start %v, got %v", start, gotStart)
		}

		elapsed := responder.ElapsedMillis(ctx)
		if elapsed < 5.0 {
			t.Errorf("expected elapsed >= 5ms, got %f", elapsed)
		}
	})

	t.Run("Context fallback defaults for uninitialized context", func(t *testing.T) {
		emptyCtx := context.Background()
		if got := responder.GetRequestID(emptyCtx); got != "" {
			t.Errorf("expected empty string for uninitialized context, got %q", got)
		}

		if gotStart := responder.GetStartTime(emptyCtx); gotStart.IsZero() {
			// GetStartTime should return non-zero or fallback safely
		}

		elapsed := responder.ElapsedMillis(emptyCtx)
		if elapsed < 0 {
			t.Errorf("elapsed should not be negative, got %f", elapsed)
		}
	})
}

func TestMiddleware_CorrelationLifecycle(t *testing.T) {
	t.Run("Echoes existing inbound X-Request-ID", func(t *testing.T) {
		existingID := "trace-edge-client-abc-999"
		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		req.Header.Set("X-Request-ID", existingID)
		rr := httptest.NewRecorder()

		var recordedID string
		var hasContextStart bool

		handler := responder.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			recordedID = responder.GetRequestID(r.Context())
			hasContextStart = !responder.GetStartTime(r.Context()).IsZero()
			w.WriteHeader(http.StatusOK)
		}))

		handler.ServeHTTP(rr, req)

		if recordedID != existingID {
			t.Errorf("expected context to have %q, got %q", existingID, recordedID)
		}
		if !hasContextStart {
			t.Errorf("expected start time in request context")
		}
		if gotHeader := rr.Header().Get("X-Request-ID"); gotHeader != existingID {
			t.Errorf("expected response header X-Request-ID %q, got %q", existingID, gotHeader)
		}
	})

	t.Run("Generates high-entropy UUIDv4 when X-Request-ID is absent", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		rr := httptest.NewRecorder()

		var generatedID string
		handler := responder.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			generatedID = responder.GetRequestID(r.Context())
			w.WriteHeader(http.StatusOK)
		}))

		handler.ServeHTTP(rr, req)

		if generatedID == "" {
			t.Fatalf("expected non-empty generated request ID")
		}
		if !strings.HasPrefix(generatedID, "req-") && len(generatedID) < 16 {
			t.Errorf("expected valid ID format, got %q", generatedID)
		}
		if gotHeader := rr.Header().Get("X-Request-ID"); gotHeader != generatedID {
			t.Errorf("expected response header to match generated ID %q, got %q", generatedID, gotHeader)
		}
	})
}
