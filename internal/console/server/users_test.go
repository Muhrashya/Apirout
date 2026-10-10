package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/apirout/apirout/internal/console/auth"
	"github.com/apirout/apirout/internal/console/registry"
)

func TestUserHandler_List_RBAC(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_users_rbac.db")

	reg, err := registry.NewWithSQLite(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test registry: %v", err)
	}
	defer reg.Close()

	if err := reg.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	handler := NewUserHandler(reg)

	// 1. Non-super-admin (DEVELOPER) should be rejected with 403 Forbidden
	reqDev := httptest.NewRequest("GET", "/api/v1/users", nil)
	devClaims := &auth.Claims{
		UserID: "dev-user-id",
		Email:  "dev@example.com",
		Role:   "DEVELOPER",
	}
	ctxDev := context.WithValue(reqDev.Context(), auth.UserContextKey, devClaims)
	reqDev = reqDev.WithContext(ctxDev)

	recDev := httptest.NewRecorder()
	handler.List(recDev, reqDev)

	if recDev.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for DEVELOPER listing users, got %d", recDev.Code)
	}

	// 2. Unauthenticated request should be rejected with 403 Forbidden
	reqAnon := httptest.NewRequest("GET", "/api/v1/users", nil)
	recAnon := httptest.NewRecorder()
	handler.List(recAnon, reqAnon)

	if recAnon.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for anonymous caller listing users, got %d", recAnon.Code)
	}

	// 3. SUPER_ADMIN should be allowed with 200 OK
	reqSuper := httptest.NewRequest("GET", "/api/v1/users", nil)
	superClaims := &auth.Claims{
		UserID: "super-user-id",
		Email:  "admin@example.com",
		Role:   "SUPER_ADMIN",
	}
	ctxSuper := context.WithValue(reqSuper.Context(), auth.UserContextKey, superClaims)
	reqSuper = reqSuper.WithContext(ctxSuper)

	recSuper := httptest.NewRecorder()
	handler.List(recSuper, reqSuper)

	if recSuper.Code != http.StatusOK {
		t.Errorf("expected 200 OK for SUPER_ADMIN listing users, got %d", recSuper.Code)
	}
}
