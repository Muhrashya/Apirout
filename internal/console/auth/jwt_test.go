package auth

import (
	"testing"
	"time"

	"github.com/apirout/apirout/internal/console/registry"

	"github.com/golang-jwt/jwt/v5"
)

func TestValidateToken_AlgNone(t *testing.T) {
	// 1. Create a "none" alg token manually
	token := jwt.New(jwt.SigningMethodNone)
	claims := token.Claims.(jwt.MapClaims)
	claims["foo"] = "bar"
	claims["exp"] = time.Now().Add(time.Hour).Unix()

	// SignedString with UnsafeAllowNoneSignatureType
	tokenString, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)

	// Service setup
	svc := &Service{
		jwtSecret: []byte("secret"),
	}

	// 2. Try to validate it
	_, err := svc.ValidateToken(tokenString)
	if err == nil {
		t.Error("ValidateToken() accepted alg: none token")
	}
}

func TestValidateToken_HS256(t *testing.T) {
	svc := &Service{
		jwtSecret: []byte("secret"),
		ttl:       time.Hour,
	}

	// Create valid token
	validToken, _, _ := svc.GenerateToken(&registry.User{
		ID:    "1",
		Email: "test@test.com",
		Role:  "ADMIN",
	})

	claims, err := svc.ValidateToken(validToken)
	if err != nil {
		t.Errorf("ValidateToken() failed for valid token: %v", err)
	}
	if claims == nil || claims.UserID != "1" {
		t.Error("Claims mismatch")
	}
}
