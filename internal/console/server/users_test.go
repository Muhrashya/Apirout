package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/mux"
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

func TestUserHandler_LockoutGuardrails(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_users_lockout.db")

	reg, err := registry.NewWithSQLite(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test registry: %v", err)
	}
	defer reg.Close()

	if err := reg.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	authSvc := auth.NewService("test-secret-key-32-chars-long!", reg, 60)
	handler := NewUserHandler(reg, authSvc)

	// Create root super admin
	admin, err := reg.CreateUser("admin@apirout.io", "Pass12345!", auth.RoleSuperAdmin)
	if err != nil {
		t.Fatalf("Failed to create super admin: %v", err)
	}

	// 1. Test Self-Deletion Guardrail
	reqSelfDelete := httptest.NewRequest("DELETE", "/api/v1/users/"+admin.ID, nil)
	reqSelfDelete = mux.SetURLVars(reqSelfDelete, map[string]string{"id": admin.ID})
	adminClaims := &auth.Claims{
		UserID: admin.ID,
		Email:  admin.Email,
		Role:   admin.Role,
	}
	ctxAdmin := context.WithValue(reqSelfDelete.Context(), auth.UserContextKey, adminClaims)
	reqSelfDelete = reqSelfDelete.WithContext(ctxAdmin)

	recSelfDelete := httptest.NewRecorder()
	handler.Delete(recSelfDelete, reqSelfDelete)

	if recSelfDelete.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for self-deletion, got %d", recSelfDelete.Code)
	}

	// 2. Test Deleting Last Super Admin (by different caller, e.g. system/another admin)
	reqLastDelete := httptest.NewRequest("DELETE", "/api/v1/users/"+admin.ID, nil)
	reqLastDelete = mux.SetURLVars(reqLastDelete, map[string]string{"id": admin.ID})
	otherCallerClaims := &auth.Claims{
		UserID: "different-admin-id",
		Email:  "other@apirout.io",
		Role:   auth.RoleSuperAdmin,
	}
	ctxOther := context.WithValue(reqLastDelete.Context(), auth.UserContextKey, otherCallerClaims)
	reqLastDelete = reqLastDelete.WithContext(ctxOther)

	recLastDelete := httptest.NewRecorder()
	handler.Delete(recLastDelete, reqLastDelete)

	if recLastDelete.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for deleting last super admin, got %d", recLastDelete.Code)
	}

	// 3. Test Demoting Last Super Admin
	reqDemote := httptest.NewRequest("PUT", "/api/v1/users/"+admin.ID, strings.NewReader(`{"email":"admin@apirout.io","role":"DEVELOPER"}`))
	reqDemote = mux.SetURLVars(reqDemote, map[string]string{"id": admin.ID})
	reqDemote = reqDemote.WithContext(ctxOther)

	recDemote := httptest.NewRecorder()
	handler.Update(recDemote, reqDemote)

	if recDemote.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for demoting last super admin, got %d", recDemote.Code)
	}
}

