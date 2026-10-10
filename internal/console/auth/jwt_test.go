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

func TestService_VerifyUserActive_Revocation(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := tempDir + "/test_jwt_revocation.db"

	reg, err := registry.NewWithSQLite(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test registry: %v", err)
	}
	defer reg.Close()

	if err := reg.Migrate(); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	user, err := reg.CreateUser("user@example.com", "Password123!", "ADMIN")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	svc := NewService("test-secret-min-32-chars-long!!!!!", reg, 60)

	// Generate token
	tokenStr, _, err := svc.GenerateToken(user)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	claims, err := svc.ValidateToken(tokenStr)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	// 1. Valid user should pass verification
	if err := svc.VerifyUserActive(claims); err != nil {
		t.Errorf("Expected active user to pass, got: %v", err)
	}

	// 2. Change password -> old token should be revoked
	time.Sleep(1100 * time.Millisecond)
	if err := reg.UpdateUserPassword(user.ID, "NewPassword123!"); err != nil {
		t.Fatalf("Failed to update user password: %v", err)
	}
	svc.InvalidateUserCache(user.ID)

	if err := svc.VerifyUserActive(claims); err == nil {
		t.Error("Expected token to be revoked after password change, but passed")
	}

	// 3. Delete user -> token should be revoked
	_ = reg.DeleteUser(user.ID)
	svc.InvalidateUserCache(user.ID)

	if err := svc.VerifyUserActive(claims); err == nil {
		t.Error("Expected token to be revoked after user deletion, but passed")
	}
}

