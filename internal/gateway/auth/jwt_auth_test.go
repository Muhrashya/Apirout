package auth

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWTAuthPlugin_Name(t *testing.T) {
	p := &JWTAuthPlugin{}
	if p.Name() != "jwt-auth" {
		t.Errorf("got %s, want jwt-auth", p.Name())
	}
}

func TestJWTAuthPlugin_MissingToken(t *testing.T) {
	p := &JWTAuthPlugin{secret: []byte("my-test-secret")}
	req := httptest.NewRequest("GET", "/test", nil)

	_, err := p.Authenticate(req, map[string]interface{}{}, nil)
	if err == nil {
		t.Fatal("expected error for missing token, got nil")
	}
}

func TestJWTAuthPlugin_ValidToken(t *testing.T) {
	secret := []byte("super-secret-key-for-jwt-testing-123")
	p := &JWTAuthPlugin{secret: secret}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":           "consumer-user-123",
		"api_client_id": "client-456",
		"group_id":      "group-789",
		"exp":           time.Now().Add(time.Hour).Unix(),
	})

	tokenString, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	config := map[string]interface{}{
		"algorithms": []string{"HS256"},
	}

	result, err := p.Authenticate(req, config, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ConsumerID != "client-456" {
		t.Errorf("got ConsumerID %s, want client-456", result.ConsumerID)
	}
	if result.GroupID != "group-789" {
		t.Errorf("got GroupID %s, want group-789", result.GroupID)
	}
	if result.AuthType != "jwt" {
		t.Errorf("got AuthType %s, want jwt", result.AuthType)
	}
}

func TestJWTAuthPlugin_ExpiredToken(t *testing.T) {
	secret := []byte("super-secret-key-for-jwt-testing-123")
	p := &JWTAuthPlugin{secret: secret}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "expired-user",
		"exp": time.Now().Add(-time.Hour).Unix(), // expired 1 hour ago
	})

	tokenString, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	_, err = p.Authenticate(req, map[string]interface{}{"algorithms": []string{"HS256"}}, nil)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestJWTAuthPlugin_DisallowedAlgorithm(t *testing.T) {
	secret := []byte("super-secret-key-for-jwt-testing-123")
	p := &JWTAuthPlugin{secret: secret}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user-1",
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	tokenString, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	// Only allow RS256, while token was signed with HS256
	config := map[string]interface{}{
		"algorithms": []string{"RS256"},
	}

	_, err = p.Authenticate(req, config, nil)
	if err == nil {
		t.Fatal("expected error for disallowed algorithm, got nil")
	}
}

func TestJWTAuthPlugin_MissingAlgHeader(t *testing.T) {
	p := &JWTAuthPlugin{secret: []byte("secret")}
	req := httptest.NewRequest("GET", "/test", nil)
	// Raw token without alg header (e.g., {"typ":"JWT"}.{"sub":"1"}.signature)
	req.Header.Set("Authorization", "Bearer eyJ0eXAiOiJKV1QifQ.eyJzdWIiOiIxIn0.c2lnbmF0dXJl")

	_, err := p.Authenticate(req, map[string]interface{}{"algorithms": []string{"HS256"}}, nil)
	if err == nil {
		t.Fatal("expected error for token with missing alg, got nil")
	}
}

func TestJWTAuthPlugin_RejectAdminCrossPlaneToken(t *testing.T) {
	secret := []byte("shared-or-dedicated-secret-key-32ch")
	p := &JWTAuthPlugin{secret: secret}

	// 1. Admin token with typ: admin
	adminToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":     "admin-user-id",
		"user_id": "admin-user-id",
		"email":   "admin@example.com",
		"role":    "SUPER_ADMIN",
		"typ":     "admin",
	})
	adminTokenStr, err := adminToken.SignedString(secret)
	if err != nil {
		t.Fatalf("failed to sign admin token: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/protected", nil)
	req.Header.Set("Authorization", "Bearer "+adminTokenStr)

	_, err = p.Authenticate(req, map[string]interface{}{}, nil)
	if err == nil {
		t.Fatal("expected admin token to be rejected by gateway jwt auth, but it was accepted")
	}
	if !strings.Contains(err.Error(), "cross-plane token rejected") {
		t.Errorf("expected cross-plane token error, got: %v", err)
	}

	// 2. Legitimate consumer token with sub should succeed
	consumerToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "consumer-client-id",
	})
	consumerTokenStr, err := consumerToken.SignedString(secret)
	if err != nil {
		t.Fatalf("failed to sign consumer token: %v", err)
	}

	req2 := httptest.NewRequest("GET", "/api/v1/protected", nil)
	req2.Header.Set("Authorization", "Bearer "+consumerTokenStr)

	res, err := p.Authenticate(req2, map[string]interface{}{}, nil)
	if err != nil {
		t.Fatalf("expected consumer token to be accepted, got error: %v", err)
	}
	if res.ConsumerID != "consumer-client-id" {
		t.Errorf("expected consumer ID 'consumer-client-id', got %s", res.ConsumerID)
	}
}

