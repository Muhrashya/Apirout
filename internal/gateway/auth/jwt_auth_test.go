package auth

import (
	"net/http/httptest"
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
