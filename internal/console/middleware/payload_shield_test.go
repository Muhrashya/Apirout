package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/apirout/apirout/internal/gateway/responder"
)

func TestConsolePayloadShield(t *testing.T) {
	const limit = 2048 // 2KB for test
	handler := PayloadShield(limit)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))

	// Case 1: Within limit
	smallPayload := bytes.Repeat([]byte("b"), 512)
	req1 := httptest.NewRequest("POST", "/admin/test", bytes.NewReader(smallPayload))
	req1.ContentLength = int64(len(smallPayload))
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got: %d", rec1.Code)
	}

	// Case 2: Exceeding Content-Length
	largePayload := bytes.Repeat([]byte("b"), 4096)
	req2 := httptest.NewRequest("POST", "/admin/test", bytes.NewReader(largePayload))
	req2.ContentLength = int64(len(largePayload))
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got: %d", rec2.Code)
	}
	var env responder.ErrorEnvelope
	if err := json.Unmarshal(rec2.Body.Bytes(), &env); err != nil {
		t.Fatalf("failed to decode error envelope: %v", err)
	}
	if env.Code != responder.ErrCodePayloadTooLarge {
		t.Fatalf("expected PAYLOAD_TOO_LARGE, got: %s", env.Code)
	}
}
