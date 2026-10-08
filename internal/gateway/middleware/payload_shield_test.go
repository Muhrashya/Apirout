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

func TestPayloadShield_WithinLimit(t *testing.T) {
	const limit = 1024
	handler := PayloadShield(limit)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))

	payload := bytes.Repeat([]byte("a"), 512)
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(payload))
	req.ContentLength = int64(len(payload))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got: %d", rec.Code)
	}
	if rec.Body.Len() != 512 {
		t.Fatalf("expected 512 bytes returned, got: %d", rec.Body.Len())
	}
}

func TestPayloadShield_ExceedsContentLength(t *testing.T) {
	const limit = 1024
	called := false
	handler := PayloadShield(limit)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	payload := bytes.Repeat([]byte("a"), 2048)
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(payload))
	req.ContentLength = int64(len(payload))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if called {
		t.Fatal("downstream handler should not have been called when Content-Length exceeds limit")
	}
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status 413, got: %d", rec.Code)
	}

	var env responder.ErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON error envelope: %v", err)
	}
	if env.Code != responder.ErrCodePayloadTooLarge {
		t.Fatalf("expected error code PAYLOAD_TOO_LARGE, got: %s", env.Code)
	}
}

func TestPayloadShield_ExceedsStreamReading(t *testing.T) {
	const limit = 1024
	handler := PayloadShield(limit)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handler tries to read full body
		_, err := io.ReadAll(r.Body)
		if err != nil {
			// Handler might try to write 400 Bad Request
			http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	// Chunked stream simulation without ContentLength
	payload := bytes.Repeat([]byte("a"), 2048)
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(payload))
	req.ContentLength = -1 // Unknown / streaming
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status 413 when stream exceeds limit, got: %d", rec.Code)
	}

	var env responder.ErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON error envelope: %v", err)
	}
	if env.Code != responder.ErrCodePayloadTooLarge {
		t.Fatalf("expected error code PAYLOAD_TOO_LARGE, got: %s", env.Code)
	}
}

func TestPayloadShield_NoLimit(t *testing.T) {
	handler := PayloadShield(0)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))

	payload := bytes.Repeat([]byte("a"), 5000)
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(payload))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 with limit 0, got: %d", rec.Code)
	}
}
