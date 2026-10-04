package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/apirout/apirout/internal/console/registry"
)

func setupTestAuthHandler(t *testing.T) (*Handler, *registry.Registry) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_auth.db")

	reg, err := registry.NewWithSQLite(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize test registry: %v", err)
	}

	if err := reg.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}
	if err := reg.MigrateSchema(); err != nil {
		t.Fatalf("Failed to run schema migrations: %v", err)
	}

	svc := NewService("test-secret-key-32-chars-long!", reg, 60, "master-admin-secret")
	handler := NewHandler(svc, reg)

	return handler, reg
}

func TestSetupLifecycle_Seam(t *testing.T) {
	handler, reg := setupTestAuthHandler(t)
	defer reg.Close()

	// 1. Initial State: Zero Super Admins -> Setup Required
	req := httptest.NewRequest("GET", "/api/v1/auth/setup-status", nil)
	rr := httptest.NewRecorder()
	handler.SetupStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d: %s", rr.Code, rr.Body.String())
	}

	var statusResp SetupStatusResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &statusResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if !statusResp.SetupRequired || statusResp.HasSuperAdmin {
		t.Fatalf("Expected setup_required=true, has_super_admin=false, got %+v", statusResp)
	}

	// 2. Validation failures
	cases := []struct {
		name    string
		payload SetupRequest
	}{
		{
			name: "missing email",
			payload: SetupRequest{
				Email:           "",
				Password:        "Password123!",
				ConfirmPassword: "Password123!",
			},
		},
		{
			name: "invalid email format",
			payload: SetupRequest{
				Email:           "notanemail",
				Password:        "Password123!",
				ConfirmPassword: "Password123!",
			},
		},
		{
			name: "short password",
			payload: SetupRequest{
				Email:           "admin@apirout.io",
				Password:        "short",
				ConfirmPassword: "short",
			},
		},
		{
			name: "password mismatch",
			payload: SetupRequest{
				Email:           "admin@apirout.io",
				Password:        "Password123!",
				ConfirmPassword: "DifferentPassword123!",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.payload)
			postReq := httptest.NewRequest("POST", "/api/v1/auth/setup", bytes.NewReader(body))
			postRec := httptest.NewRecorder()
			handler.Setup(postRec, postReq)

			if postRec.Code != http.StatusBadRequest {
				t.Errorf("Expected 400 Bad Request for %s, got %d", tc.name, postRec.Code)
			}
		})
	}

	// 3. Successful Setup Execution
	validPayload := SetupRequest{
		Email:           "root-admin@apirout.io",
		Password:        "RootSuperSecret2026!",
		ConfirmPassword: "RootSuperSecret2026!",
	}
	validBody, _ := json.Marshal(validPayload)
	setupReq := httptest.NewRequest("POST", "/api/v1/auth/setup", bytes.NewReader(validBody))
	setupRec := httptest.NewRecorder()
	handler.Setup(setupRec, setupReq)

	if setupRec.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created, got %d: %s", setupRec.Code, setupRec.Body.String())
	}

	var loginResp LoginResponse
	if err := json.Unmarshal(setupRec.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("Failed to decode login response: %v", err)
	}
	if loginResp.Token == "" {
		t.Fatal("Expected JWT token to be generated and returned")
	}
	if loginResp.User.Email != "root-admin@apirout.io" || loginResp.User.Role != "SUPER_ADMIN" {
		t.Fatalf("Expected root-admin@apirout.io with SUPER_ADMIN role, got %+v", loginResp.User)
	}

	// Verify token validity
	claims, err := handler.service.ValidateToken(loginResp.Token)
	if err != nil {
		t.Fatalf("Generated token failed validation: %v", err)
	}
	if claims.Role != "SUPER_ADMIN" || claims.Email != "root-admin@apirout.io" {
		t.Fatalf("Token claims mismatch: %+v", claims)
	}

	// Verify audit log entry
	logs, err := reg.ListAuditLogs("super_admin_setup_completed", 10)
	if err != nil {
		t.Fatalf("Failed to fetch audit logs: %v", err)
	}
	if len(logs) == 0 {
		t.Fatal("Expected super_admin_setup_completed audit log entry")
	}

	// 4. Permanent Lock: Subsequent invocation must return 409 Conflict
	replayReq := httptest.NewRequest("POST", "/api/v1/auth/setup", bytes.NewReader(validBody))
	replayRec := httptest.NewRecorder()
	handler.Setup(replayRec, replayReq)

	if replayRec.Code != http.StatusConflict {
		t.Fatalf("Expected 409 Conflict after setup completed, got %d: %s", replayRec.Code, replayRec.Body.String())
	}

	// 5. Subsequent Status Check: setup_required=false, has_super_admin=true
	statusReq2 := httptest.NewRequest("GET", "/api/v1/auth/setup-status", nil)
	statusRec2 := httptest.NewRecorder()
	handler.SetupStatus(statusRec2, statusReq2)

	if statusRec2.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", statusRec2.Code)
	}
	var statusResp2 SetupStatusResponse
	json.Unmarshal(statusRec2.Body.Bytes(), &statusResp2)
	if statusResp2.SetupRequired || !statusResp2.HasSuperAdmin {
		t.Fatalf("Expected setup_required=false, has_super_admin=true, got %+v", statusResp2)
	}
}
