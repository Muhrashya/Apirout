package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDataPlaneSecurityHeaders(t *testing.T) {
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	secured := SecurityHeaders(dummyHandler)

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	rr := httptest.NewRecorder()

	secured.ServeHTTP(rr, req)

	if got := rr.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("expected X-Content-Type-Options: nosniff, got: %s", got)
	}

	if got := rr.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("expected X-Frame-Options: DENY, got: %s", got)
	}

	if got := rr.Header().Get("Referrer-Policy"); got != "no-referrer" {
		t.Errorf("expected Referrer-Policy: no-referrer, got: %s", got)
	}
}
