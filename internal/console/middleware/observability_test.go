package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestIDEchoesIncoming(t *testing.T) {
	var gotID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotID = GetRequestID(r.Context())
	})
	handler := RequestID(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(HeaderRequestID, "fixed-id")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if gotID != "fixed-id" {
		t.Errorf("context id = %q, want fixed-id", gotID)
	}
	if rr.Header().Get(HeaderRequestID) != "fixed-id" {
		t.Errorf("response header = %q", rr.Header().Get(HeaderRequestID))
	}
}

func TestRequestIDGeneratesID(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if GetRequestID(r.Context()) == "" {
			t.Error("expected generated request id in context")
		}
	})
	rr := httptest.NewRecorder()
	RequestID(next).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	if rr.Header().Get(HeaderRequestID) == "" {
		t.Error("expected generated request id in response header")
	}
}

func TestRecovererReturnsJSON500(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	rr := httptest.NewRecorder()
	Recoverer(logger)(next).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/panic", nil))

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rr.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON body %q: %v", rr.Body.String(), err)
	}
	if body["error"] != "Internal server error" {
		t.Errorf("error = %q", body["error"])
	}
	if !strings.Contains(buf.String(), "panic recovered") {
		t.Errorf("panic was not logged: %s", buf.String())
	}
}

func TestAccessLogCapturesStatus(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	rr := httptest.NewRecorder()
	AccessLog(logger)(next).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/teapot", nil))

	if rr.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want 418", rr.Code)
	}
	out := buf.String()
	if !strings.Contains(out, "status=418") {
		t.Errorf("access log did not capture status: %s", out)
	}
}

func TestRateLimiterStopIsIdempotent(t *testing.T) {
	rl := NewRateLimiter(10, 0, true)
	rl.Stop()
	rl.Stop()
}
