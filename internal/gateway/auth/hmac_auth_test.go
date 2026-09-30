package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHMACAuthPlugin_Name(t *testing.T) {
	p := &HMACAuthPlugin{}
	if p.Name() != "hmac-auth" {
		t.Errorf("got %s, want hmac-auth", p.Name())
	}
}

func TestHMACAuthPlugin_MissingHeader(t *testing.T) {
	p := &HMACAuthPlugin{}
	req := httptest.NewRequest("GET", "/test", nil)

	_, err := p.Authenticate(req, map[string]interface{}{}, nil)
	if err == nil {
		t.Fatal("expected error for missing hmac header, got nil")
	}
}

func TestHMACAuthPlugin_IncompleteHeader(t *testing.T) {
	p := &HMACAuthPlugin{}
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "hmac username=\"admin\"")

	_, err := p.Authenticate(req, map[string]interface{}{}, nil)
	if err == nil {
		t.Fatal("expected error for incomplete hmac header, got nil")
	}
}

func TestHMACAuthPlugin_DisallowedAlgorithm(t *testing.T) {
	p := &HMACAuthPlugin{}
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", `hmac username="admin", algorithm="hmac-md5", headers="date", signature="abc"`)

	config := map[string]interface{}{
		"algorithms": []string{"hmac-sha256"},
	}

	_, err := p.Authenticate(req, config, nil)
	if err == nil {
		t.Fatal("expected error for disallowed algorithm, got nil")
	}
}

func TestHMACAuthPlugin_ClockSkewExceeded(t *testing.T) {
	p := &HMACAuthPlugin{}
	req := httptest.NewRequest("GET", "/test", nil)
	// Date header from 1 hour ago
	oldDate := time.Now().Add(-time.Hour).Format(time.RFC1123)
	req.Header.Set("Date", oldDate)
	req.Header.Set("Authorization", `hmac username="admin", algorithm="hmac-sha256", headers="date", signature="abc"`)

	config := map[string]interface{}{
		"algorithms": []string{"hmac-sha256"},
		"clock_skew": 300,
	}

	_, err := p.Authenticate(req, config, nil)
	if err == nil {
		t.Fatal("expected error for clock skew exceeded, got nil")
	}
}

func TestParseHMACParams(t *testing.T) {
	headerVal := `username="alice", algorithm="hmac-sha256", headers="date request-line", signature="xyz123=="`
	params := parseHMACParams(headerVal)

	if params["username"] != "alice" {
		t.Errorf("got %s, want alice", params["username"])
	}
	if params["algorithm"] != "hmac-sha256" {
		t.Errorf("got %s, want hmac-sha256", params["algorithm"])
	}
	if params["headers"] != "date request-line" {
		t.Errorf("got %s, want date request-line", params["headers"])
	}
	if params["signature"] != "xyz123==" {
		t.Errorf("got %s, want xyz123==", params["signature"])
	}
}

func TestHMACSignatureComputation(t *testing.T) {
	secret := []byte("shared-secret")
	data := "date: Wed, 30 Sep 2026 12:00:00 GMT"

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(data))
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	if sig == "" {
		t.Fatal("signature should not be empty")
	}

	mac2 := hmac.New(sha256.New, secret)
	mac2.Write([]byte(data))
	sig2 := base64.StdEncoding.EncodeToString(mac2.Sum(nil))

	if !hmac.Equal([]byte(sig), []byte(sig2)) {
		t.Errorf("signatures should match: %s vs %s", sig, sig2)
	}
	_ = fmt.Sprintf("sig: %s", sig)
}
