package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeaders(t *testing.T) {
	// Dummy handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with middleware
	secured := SecurityHeaders(handler)

	// Create request
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	// Execute
	secured.ServeHTTP(rr, req)

	// Check headers
	headers := rr.Header()

	checks := map[string]string{
		"X-Content-Type-Options":      "nosniff",
		"X-Frame-Options":             "DENY",
		"Referrer-Policy":             "no-referrer",
		"Strict-Transport-Security":   "max-age=31536000; includeSubDomains",
		"Content-Security-Policy":     "default-src 'self'; script-src 'self'; style-src 'self' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; img-src 'self' data:; connect-src 'self'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'; object-src 'none';",
	}

	for key, expected := range checks {
		if val := headers.Get(key); val != expected {
			t.Errorf("Header %s = %s, want %s", key, val, expected)
		}
	}
}
