package e2e_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"

	"github.com/apirout/apirout/internal/config"
	consoleAuth "github.com/apirout/apirout/internal/console/auth"
	"github.com/apirout/apirout/internal/console/openapi"
	"github.com/apirout/apirout/internal/console/postman"
	consoleRegistry "github.com/apirout/apirout/internal/console/registry"
	"github.com/apirout/apirout/internal/console/server"
	"github.com/apirout/apirout/internal/core/egress"
	"github.com/apirout/apirout/internal/core/models"
	gatewayApp "github.com/apirout/apirout/internal/gateway/app"
	"github.com/apirout/apirout/internal/gateway/apiclientauth"
	gatewayExecutor "github.com/apirout/apirout/internal/gateway/executor"
	gatewayMiddleware "github.com/apirout/apirout/internal/gateway/middleware"
	gatewayRegistry "github.com/apirout/apirout/internal/gateway/registry"
)

// 1. E2E: Instant CRUD cross-connection BOLA 403 enforcement
func TestE2E_Phase2_InstantCRUD_BOLA(t *testing.T) {
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}
	defer db.Close()

	schema := `
		CREATE TABLE connections (
			id TEXT PRIMARY KEY,
			group_id TEXT,
			name TEXT,
			db_type TEXT,
			host TEXT,
			port INT,
			db_name TEXT,
			username TEXT,
			password_encrypted TEXT,
			ssl_mode TEXT,
			created_at DATETIME,
			updated_at DATETIME
		);
		CREATE TABLE api_clients (
			id TEXT PRIMARY KEY,
			group_id TEXT,
			username TEXT,
			role TEXT,
			rate_limit INT,
			is_active INT DEFAULT 1,
			expires_at DATETIME,
			created_at DATETIME
		);
		CREATE TABLE api_client_keys (
			id TEXT PRIMARY KEY,
			client_id TEXT,
			key_id TEXT NOT NULL,
			secret_hash TEXT NOT NULL,
			masked_key TEXT NOT NULL,
			enabled INT DEFAULT 1,
			expires_at DATETIME,
			last_used_at DATETIME,
			created_at DATETIME
		);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO connections (id, group_id, name, db_type, host, port, db_name, username, password_encrypted, ssl_mode, created_at, updated_at)
		VALUES ('conn-a', 'group-a', 'Tenant A DB', 'postgres', 'localhost', 5432, 'db_a', 'user', '', 'disable', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
		       ('conn-b', 'group-b', 'Tenant B DB', 'postgres', 'localhost', 5432, 'db_b', 'user', '', 'disable', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`)
	if err != nil {
		t.Fatalf("failed to seed connections: %v", err)
	}

	reader := gatewayRegistry.NewReaderWithDB(db, "sqlite")
	apiClientAuth := apiclientauth.NewService("test-jwt-secret-key-32-chars-long!", reader, 10*time.Minute)
	handler := gatewayExecutor.NewInstantCRUDHandler(reader, "test-enc-key-32-chars-long!", nil, apiClientAuth, "admin-master-secret", "")

	// 1a. Unauthenticated consumer request -> 401 Unauthorized
	reqUnauth := httptest.NewRequest("GET", "/api/v1/instant/conn-a/users", nil)
	code, err := handler.AuthorizeConnectionAccess(reqUnauth, "conn-a")
	if err == nil || code != http.StatusUnauthorized {
		t.Errorf("unauthenticated request expected 401, got code=%d err=%v", code, err)
	}

	// 1b. API client scoped to Group A
	grpA := "group-a"
	tokenA, _, err := apiClientAuth.GenerateToken(&models.APIClient{
		ID:       "client-a",
		Username: "tenant-a-user",
		GroupID:  &grpA,
	})
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Group A accessing Connection A -> 200 OK
	reqAuthorized := httptest.NewRequest("GET", "/api/v1/instant/conn-a/users", nil)
	reqAuthorized.Header.Set("Authorization", "Bearer "+tokenA)
	code, err = handler.AuthorizeConnectionAccess(reqAuthorized, "conn-a")
	if err != nil || code != http.StatusOK {
		t.Errorf("authorized group access expected 200, got code=%d err=%v", code, err)
	}

	// Group A attempting to access Connection B -> 403 Forbidden (BOLA protected)
	reqForbidden := httptest.NewRequest("GET", "/api/v1/instant/conn-b/users", nil)
	reqForbidden.Header.Set("Authorization", "Bearer "+tokenA)
	code, err = handler.AuthorizeConnectionAccess(reqForbidden, "conn-b")
	if err == nil || code != http.StatusForbidden {
		t.Errorf("BOLA mismatch expected 403, got code=%d err=%v", code, err)
	}
}

// 2. E2E: Outbound HTTP requests reject 301/302 redirects to internal loopback and cloud metadata
func TestE2E_Phase2_SafeHTTPClient_SSRFRedirectProtection(t *testing.T) {
	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/redirect-metadata":
			http.Redirect(w, r, "http://169.254.169.254/latest/meta-data/", http.StatusFound)
		case "/redirect-loopback":
			http.Redirect(w, r, "http://127.0.0.1:8001/api/v1/admin", http.StatusMovedPermanently)
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		}
	}))
	defer redirectServer.Close()

	// 2a. Client allowing loopback dial to reach test server, but blocking cloud metadata redirect
	clientWithLoopback := egress.NewSafeHTTPClient(2*time.Second, egress.Policy{AllowLoopback: true})
	_, err := clientWithLoopback.Get(redirectServer.URL + "/redirect-metadata")
	if err == nil {
		t.Fatal("expected SafeHTTPClient to block redirect to AWS metadata, but succeeded")
	}
	if !strings.Contains(err.Error(), "cloud metadata") && !strings.Contains(err.Error(), "safe egress blocked redirect") {
		t.Errorf("expected cloud metadata redirect block error, got: %v", err)
	}

	// 2b. Default strict client rejects connecting to loopback endpoints
	strictClient := egress.NewSafeHTTPClient(2 * time.Second)
	_, err = strictClient.Get(redirectServer.URL + "/redirect-loopback")
	if err == nil {
		t.Fatal("expected strict SafeHTTPClient to block connection to 127.0.0.1 loopback, but succeeded")
	}
	if !strings.Contains(err.Error(), "loopback") && !strings.Contains(err.Error(), "safe egress") {
		t.Errorf("expected loopback block error, got: %v", err)
	}
}

// 3. E2E: Documentation endpoints require authentication and reject unauthenticated requests
func TestE2E_Phase2_DocumentationAuthAndScoping(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "docs_e2e.db")
	reg, err := consoleRegistry.NewWithSQLite(dbPath)
	if err != nil {
		t.Fatalf("failed to init test registry: %v", err)
	}
	defer reg.Close()

	if err := reg.Migrate(); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	openapiSvc := openapi.NewService(reg, nil)
	postmanSvc := postman.NewService(reg, nil)
	docsHandler := server.NewDocsHandler(openapiSvc, postmanSvc, reg)

	// 3a. Anonymous call to /api/v1/openapi.json returns 401 Unauthorized
	reqAnonOpenAPI := httptest.NewRequest("GET", "/api/v1/openapi.json", nil)
	recAnonOpenAPI := httptest.NewRecorder()
	docsHandler.OpenAPI(recAnonOpenAPI, reqAnonOpenAPI)
	if recAnonOpenAPI.Code != http.StatusUnauthorized {
		t.Errorf("expected OpenAPI to return 401 unauthenticated, got %d", recAnonOpenAPI.Code)
	}

	// 3b. Anonymous call to /api/v1/postman.json returns 401 Unauthorized
	reqAnonPostman := httptest.NewRequest("GET", "/api/v1/postman.json", nil)
	recAnonPostman := httptest.NewRecorder()
	docsHandler.Postman(recAnonPostman, reqAnonPostman)
	if recAnonPostman.Code != http.StatusUnauthorized {
		t.Errorf("expected Postman to return 401 unauthenticated, got %d", recAnonPostman.Code)
	}

	// 3c. Authenticated Super Admin call returns 200 OK
	adminClaims := &consoleAuth.Claims{
		UserID: "admin-user",
		Role:   "SUPER_ADMIN",
		Type:   "admin",
	}
	ctx := context.WithValue(context.Background(), consoleAuth.UserContextKey, adminClaims)
	reqAuthOpenAPI := httptest.NewRequest("GET", "/api/v1/openapi.json", nil).WithContext(ctx)
	recAuthOpenAPI := httptest.NewRecorder()
	docsHandler.OpenAPI(recAuthOpenAPI, reqAuthOpenAPI)
	if recAuthOpenAPI.Code != http.StatusOK {
		t.Errorf("expected authenticated OpenAPI to return 200, got %d", recAuthOpenAPI.Code)
	}
}

// 4. E2E: Metrics port isolation and public data plane security headers
func TestE2E_Phase2_MetricsIsolation_And_SecurityHeaders(t *testing.T) {
	// 4a. Defensive security headers are set on public data plane responses
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data plane response"))
	})
	secured := gatewayMiddleware.SecurityHeaders(dummyHandler)

	req := httptest.NewRequest("GET", "/api/v1/orders", nil)
	rec := httptest.NewRecorder()
	secured.ServeHTTP(rec, req)

	expectedHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "no-referrer",
	}
	for header, expectedVal := range expectedHeaders {
		if got := rec.Header().Get(header); got != expectedVal {
			t.Errorf("expected header %s to be %q, got %q", header, expectedVal, got)
		}
	}

	// 4b. Metrics port isolation:
	// Public Data Plane ProxyHandler (Port 8000) returns 404 for /metrics
	// Dedicated StatusHandler (Port 8100) serves /metrics (200 OK)
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	reader := gatewayRegistry.NewReaderWithDB(db, "sqlite")
	cfg := &config.Config{
		Role:               config.RoleDataPlane,
		ProxyPort:          "8000",
		StatusPort:         "8100",
		GatewayJWTSecret:   "test-gateway-secret-key-min-32-chars!",
		EncryptionKey:      "test-encryption-key-min-32-chars-long!",
		MaxRequestBodyMB:   10,
		GlobalRateLimitRPM: 600,
	}

	gwApp, err := gatewayApp.NewWithReader(cfg, nil, reader)
	if err != nil {
		t.Fatalf("failed to create gateway app: %v", err)
	}
	defer gwApp.Close()

	// Request /metrics on Public ProxyHandler -> 404 Not Found
	reqMetrics := httptest.NewRequest("GET", "/metrics", nil)
	recMetrics := httptest.NewRecorder()
	gwApp.ProxyHandler().ServeHTTP(recMetrics, reqMetrics)
	if recMetrics.Code != http.StatusNotFound {
		t.Errorf("expected /metrics on public ProxyHandler to return 404 Not Found, got %d", recMetrics.Code)
	}

	// Request /metrics on StatusHandler -> 200 OK
	reqStatusMetrics := httptest.NewRequest("GET", "/metrics", nil)
	recStatusMetrics := httptest.NewRecorder()
	gwApp.StatusHandler().ServeHTTP(recStatusMetrics, reqStatusMetrics)
	if recStatusMetrics.Code != http.StatusOK {
		t.Errorf("expected /metrics on StatusHandler to return 200 OK, got %d", recStatusMetrics.Code)
	}
}

// 5. E2E: Stateful JWT revocation on password rotation & account deletion
func TestE2E_Phase2_StatefulJWTRevocation(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "jwt_revocation_e2e.db")
	reg, err := consoleRegistry.NewWithSQLite(dbPath)
	if err != nil {
		t.Fatalf("failed to init test registry: %v", err)
	}
	defer reg.Close()

	if err := reg.Migrate(); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	user, err := reg.CreateUser("security_admin@example.com", "CurrentSecurePassword123!", "ADMIN")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	authSvc := consoleAuth.NewService("strong-test-jwt-secret-key-32chars!", reg, 60)
	tokenStr, _, err := authSvc.GenerateToken(user)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	claims, err := authSvc.ValidateToken(tokenStr)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}

	// Initially active token is accepted
	if err := authSvc.VerifyUserActive(claims); err != nil {
		t.Fatalf("expected valid token to pass user verification, got: %v", err)
	}

	// Password rotated after token was issued -> token revoked
	time.Sleep(1100 * time.Millisecond)
	if err := reg.UpdateUserPassword(user.ID, "NewlyRotatedPassword456!"); err != nil {
		t.Fatalf("failed to rotate password: %v", err)
	}
	authSvc.InvalidateUserCache(user.ID)

	if err := authSvc.VerifyUserActive(claims); err == nil {
		t.Fatal("expected token to be revoked after password update, but verification succeeded")
	} else if !strings.Contains(err.Error(), "token revoked") {
		t.Errorf("expected token revoked error message, got: %v", err)
	}
}

// 6. E2E: Super Admin self-deletion and lockout guardrails
func TestE2E_Phase2_SuperAdminLockoutGuardrails(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "lockout_e2e.db")
	reg, err := consoleRegistry.NewWithSQLite(dbPath)
	if err != nil {
		t.Fatalf("failed to init test registry: %v", err)
	}
	defer reg.Close()

	if err := reg.Migrate(); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	admin1, err := reg.CreateUser("super1@apirout.io", "Pass12345!", "SUPER_ADMIN")
	if err != nil {
		t.Fatalf("failed to create super admin 1: %v", err)
	}

	userHandler := server.NewUserHandler(reg)

	// 6a. Super Admin 1 attempts self-deletion -> 400 Bad Request
	reqSelfDelete := httptest.NewRequest("DELETE", "/api/v1/users/"+admin1.ID, nil)
	reqSelfDelete = mux.SetURLVars(reqSelfDelete, map[string]string{"id": admin1.ID})
	ctxSelf := context.WithValue(reqSelfDelete.Context(), consoleAuth.UserContextKey, &consoleAuth.Claims{
		UserID: admin1.ID,
		Role:   "SUPER_ADMIN",
	})
	reqSelfDelete = reqSelfDelete.WithContext(ctxSelf)
	recSelfDelete := httptest.NewRecorder()
	userHandler.Delete(recSelfDelete, reqSelfDelete)

	if recSelfDelete.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for self-deletion, got %d", recSelfDelete.Code)
	}
	if !strings.Contains(recSelfDelete.Body.String(), "Self-deletion is prohibited") {
		t.Errorf("expected self-deletion prohibited message, got: %s", recSelfDelete.Body.String())
	}

	// 6b. When admin1 is the sole Super Admin, another caller cannot delete admin1 -> 400 Bad Request
	reqDeleteSole := httptest.NewRequest("DELETE", "/api/v1/users/"+admin1.ID, nil)
	reqDeleteSole = mux.SetURLVars(reqDeleteSole, map[string]string{"id": admin1.ID})
	ctxOther := context.WithValue(reqDeleteSole.Context(), consoleAuth.UserContextKey, &consoleAuth.Claims{
		UserID: "another-admin-caller",
		Role:   "SUPER_ADMIN",
	})
	reqDeleteSole = reqDeleteSole.WithContext(ctxOther)
	recDeleteSole := httptest.NewRecorder()
	userHandler.Delete(recDeleteSole, reqDeleteSole)

	if recDeleteSole.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when attempting to delete last super admin, got %d", recDeleteSole.Code)
	}
	if !strings.Contains(recDeleteSole.Body.String(), "cannot delete the last active Super Admin") {
		t.Errorf("expected last super admin deletion protection message, got: %s", recDeleteSole.Body.String())
	}
}
