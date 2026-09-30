package auth

import (
	"encoding/base64"
	"net/http/httptest"
	"testing"

	"github.com/apirout/apirout/internal/core/crypto"
	"github.com/apirout/apirout/internal/core/models"
)

func TestOutboundInjector_BasicAuth(t *testing.T) {
	encKey := "test-32-byte-encryption-key-1234"
	normKey := crypto.NormalizeKey(encKey)
	injector := NewOutboundInjector(encKey)

	plainPass := "secret-password-123"
	encPass, err := crypto.EncryptPassword(normKey, plainPass)
	if err != nil {
		t.Fatalf("failed to encrypt password: %v", err)
	}

	auth := &models.ServiceOutboundAuth{
		AuthType: models.OutboundBasic,
		Enabled:  true,
		Config:   `{"username":"admin","password_encrypted":"` + encPass + `"}`,
	}

	req := httptest.NewRequest("GET", "http://backend.internal/api", nil)
	if err := injector.InjectAuth(req, auth); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	authHeader := req.Header.Get("Authorization")
	expectedHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:"+plainPass))
	if authHeader != expectedHeader {
		t.Errorf("got %s, want %s", authHeader, expectedHeader)
	}
}

func TestOutboundInjector_BearerAuth(t *testing.T) {
	encKey := "test-32-byte-encryption-key-1234"
	normKey := crypto.NormalizeKey(encKey)
	injector := NewOutboundInjector(encKey)

	plainToken := "ey-token-test-value-12345"
	encToken, err := crypto.EncryptPassword(normKey, plainToken)
	if err != nil {
		t.Fatalf("failed to encrypt token: %v", err)
	}

	auth := &models.ServiceOutboundAuth{
		AuthType: models.OutboundBearer,
		Enabled:  true,
		Config:   `{"token_encrypted":"` + encToken + `","prefix":"Bearer"}`,
	}

	req := httptest.NewRequest("GET", "http://backend.internal/api", nil)
	if err := injector.InjectAuth(req, auth); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	authHeader := req.Header.Get("Authorization")
	expectedHeader := "Bearer " + plainToken
	if authHeader != expectedHeader {
		t.Errorf("got %s, want %s", authHeader, expectedHeader)
	}
}

func TestOutboundInjector_APIKeyHeader(t *testing.T) {
	encKey := "test-32-byte-encryption-key-1234"
	normKey := crypto.NormalizeKey(encKey)
	injector := NewOutboundInjector(encKey)

	plainKey := "upstream-api-key-999"
	encKeyVal, err := crypto.EncryptPassword(normKey, plainKey)
	if err != nil {
		t.Fatalf("failed to encrypt key: %v", err)
	}

	auth := &models.ServiceOutboundAuth{
		AuthType: models.OutboundAPIKey,
		Enabled:  true,
		Config:   `{"header_name":"X-Custom-Auth","key_encrypted":"` + encKeyVal + `","in_query":false}`,
	}

	req := httptest.NewRequest("GET", "http://backend.internal/api", nil)
	if err := injector.InjectAuth(req, auth); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val := req.Header.Get("X-Custom-Auth")
	if val != plainKey {
		t.Errorf("got %s, want %s", val, plainKey)
	}
}

func TestOutboundInjector_APIKeyQuery(t *testing.T) {
	encKey := "test-32-byte-encryption-key-1234"
	normKey := crypto.NormalizeKey(encKey)
	injector := NewOutboundInjector(encKey)

	plainKey := "upstream-api-key-888"
	encKeyVal, err := crypto.EncryptPassword(normKey, plainKey)
	if err != nil {
		t.Fatalf("failed to encrypt key: %v", err)
	}

	auth := &models.ServiceOutboundAuth{
		AuthType: models.OutboundAPIKey,
		Enabled:  true,
		Config:   `{"query_param":"key","key_encrypted":"` + encKeyVal + `","in_query":true}`,
	}

	req := httptest.NewRequest("GET", "http://backend.internal/api", nil)
	if err := injector.InjectAuth(req, auth); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	queryVal := req.URL.Query().Get("key")
	if queryVal != plainKey {
		t.Errorf("got %s, want %s", queryVal, plainKey)
	}
}

func TestOutboundInjector_Disabled(t *testing.T) {
	injector := NewOutboundInjector("any-key")
	auth := &models.ServiceOutboundAuth{
		AuthType: models.OutboundBearer,
		Enabled:  false,
		Config:   `{"token_encrypted":"abc"}`,
	}

	req := httptest.NewRequest("GET", "http://backend.internal/api", nil)
	if err := injector.InjectAuth(req, auth); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if h := req.Header.Get("Authorization"); h != "" {
		t.Errorf("expected no auth header when disabled, got %s", h)
	}
}
