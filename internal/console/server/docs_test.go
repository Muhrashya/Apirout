package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/apirout/apirout/internal/console/auth"
	"github.com/apirout/apirout/internal/console/openapi"
	"github.com/apirout/apirout/internal/console/postman"
	"github.com/apirout/apirout/internal/console/registry"
)

func TestDocsHandler_RequiresAuth(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "docs_test.db")
	reg, err := registry.NewWithSQLite(dbPath)
	if err != nil {
		t.Fatalf("failed to init test registry: %v", err)
	}
	defer reg.Close()

	if err := reg.Migrate(); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	openapiSvc := openapi.NewService(reg, nil)
	postmanSvc := postman.NewService(reg, nil)
	docsHandler := NewDocsHandler(openapiSvc, postmanSvc, reg)

	// 1. Unauthenticated OpenAPI -> 401 Unauthorized
	reqUnauthOpenAPI := httptest.NewRequest("GET", "/api/v1/openapi.json", nil)
	rrOpenAPI := httptest.NewRecorder()
	docsHandler.OpenAPI(rrOpenAPI, reqUnauthOpenAPI)

	if rrOpenAPI.Code != http.StatusUnauthorized {
		t.Errorf("expected OpenAPI to return 401 unauthenticated, got %d", rrOpenAPI.Code)
	}

	// 2. Unauthenticated Postman -> 401 Unauthorized
	reqUnauthPostman := httptest.NewRequest("GET", "/api/v1/postman.json", nil)
	rrPostman := httptest.NewRecorder()
	docsHandler.Postman(rrPostman, reqUnauthPostman)

	if rrPostman.Code != http.StatusUnauthorized {
		t.Errorf("expected Postman to return 401 unauthenticated, got %d", rrPostman.Code)
	}

	// 3. Authenticated Super Admin -> 200 OK
	adminClaims := &auth.Claims{
		UserID: "admin-1",
		Email:  "admin@apirout.io",
		Role:   "SUPER_ADMIN",
		Type:   "admin",
	}
	ctx := context.WithValue(context.Background(), auth.UserContextKey, adminClaims)
	reqAuthOpenAPI := httptest.NewRequest("GET", "/api/v1/openapi.json", nil).WithContext(ctx)
	rrAuthOpenAPI := httptest.NewRecorder()
	docsHandler.OpenAPI(rrAuthOpenAPI, reqAuthOpenAPI)

	if rrAuthOpenAPI.Code != http.StatusOK {
		t.Errorf("expected authenticated OpenAPI to return 200, got %d", rrAuthOpenAPI.Code)
	}
}
