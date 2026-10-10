package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/apirout/apirout/internal/core/models"
	"github.com/apirout/apirout/internal/gateway/apiclientauth"
	"github.com/apirout/apirout/internal/gateway/responder"
)

func TestEngine_PublicEndpoint(t *testing.T) {
	engine := NewEngine(nil, "test-secret-key-at-least-32-chars-long!!")
	req := httptest.NewRequest("GET", "/api/public", nil)
	ep := &models.Endpoint{
		ID:       "ep-1",
		Path:     "/api/public",
		IsPublic: true,
	}

	result, err := engine.Authenticate(req, ep, nil)
	if err != nil {
		t.Fatalf("expected nil error for public endpoint, got: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result for public endpoint, got: %v", result)
	}
}

func TestEngine_Fallback_MissingAuth(t *testing.T) {
	engine := NewEngine(nil, "test-secret-key-at-least-32-chars-long!!")
	req := httptest.NewRequest("GET", "/api/protected", nil)
	ep := &models.Endpoint{
		ID:       "ep-2",
		Path:     "/api/protected",
		IsPublic: false,
	}

	result, err := engine.Authenticate(req, ep, nil)
	if err == nil {
		t.Fatal("expected error for missing credentials, got nil")
	}
	if result != nil {
		t.Fatalf("expected nil result, got: %v", result)
	}

	authErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *auth.Error, got %T", err)
	}
	if authErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", authErr.StatusCode)
	}
	if authErr.Code != responder.ErrCodeUnauthorized {
		t.Errorf("expected %s, got %s", responder.ErrCodeUnauthorized, authErr.Code)
	}
}

func TestEngine_Fallback_JWTAuth(t *testing.T) {
	secret := "test-secret-key-at-least-32-chars-long!!"
	acService := apiclientauth.NewService(secret, nil, 1*time.Hour)
	engine := NewEngine(nil, secret, acService)

	client := &models.APIClient{
		ID:       "client-uuid-1",
		Username: "service-user",
	}
	token, _, err := acService.GenerateToken(client)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// 1. Valid JWT
	req := httptest.NewRequest("GET", "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	ep := &models.Endpoint{
		ID:       "ep-3",
		Path:     "/api/protected",
		IsPublic: false,
	}

	result, err := engine.Authenticate(req, ep, nil)
	if err != nil {
		t.Fatalf("expected success, got err: %v", err)
	}
	if result == nil || result.ConsumerID != "client-uuid-1" {
		t.Fatalf("expected consumer ID client-uuid-1, got: %+v", result)
	}
	if result.AuthType != "jwt" {
		t.Errorf("expected jwt auth type, got %s", result.AuthType)
	}

	// 2. Invalid JWT token
	badReq := httptest.NewRequest("GET", "/api/protected", nil)
	badReq.Header.Set("Authorization", "Bearer invalid.jwt.token")
	_, badErr := engine.Authenticate(badReq, ep, nil)
	if badErr == nil {
		t.Fatal("expected error for invalid JWT token")
	}
	authErr, ok := badErr.(*Error)
	if !ok {
		t.Fatalf("expected *auth.Error, got %T", badErr)
	}
	if authErr.Code != responder.ErrCodeExpiredToken {
		t.Errorf("expected %s, got %s", responder.ErrCodeExpiredToken, authErr.Code)
	}
}

func TestEngine_AbsorbedClientAuthService(t *testing.T) {
	secret := "test-secret-key-at-least-32-chars-long!!"
	acService := apiclientauth.NewService(secret, nil, 1*time.Hour)
	engine := NewEngine(nil, secret, acService)

	client := &models.APIClient{
		ID:       "client-123",
		Username: "deep-module-user",
	}

	token, expiresAt, err := engine.GenerateToken(client)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	if token == "" || expiresAt.IsZero() {
		t.Fatalf("expected non-empty token and expiration")
	}

	claims, err := engine.ValidateToken(token)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}
	if claims.APIClientID != "client-123" {
		t.Errorf("expected client-123, got %s", claims.APIClientID)
	}

	// Verify unconfigured service handling
	unconfiguredEngine := NewEngine(nil, "")
	_, err = unconfiguredEngine.ValidateToken(token)
	if err == nil {
		t.Fatal("expected error from unconfigured engine ValidateToken")
	}
}

