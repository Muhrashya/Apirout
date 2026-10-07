package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/apirout/apirout/internal/console/registry"
	"github.com/apirout/apirout/internal/core/models"
	"github.com/gorilla/mux"
)

func setupTestRegistryAndHandler(t *testing.T) (*registry.Registry, *AuthPluginsHandler) {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_auth_plugins.db")

	reg, err := registry.NewWithSQLite(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test registry: %v", err)
	}

	if err := reg.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}
	if err := reg.MigrateSchema(); err != nil {
		t.Fatalf("Failed to migrate schema: %v", err)
	}

	handler := NewAuthPluginsHandler(reg, "test-secret-key-32-bytes-long-123")
	return reg, handler
}

func TestAuthPluginsHandler_DuplicatePrevention(t *testing.T) {
	reg, handler := setupTestRegistryAndHandler(t)
	defer reg.Close()

	// Create test route
	testRouteID := "route-test-123"
	ep := &models.Endpoint{
		ID:     testRouteID,
		Name:   "Test Users Route",
		Path:   "/api/v1/users",
		Method: "GET",
	}
	if err := reg.CreateEndpoint(ep); err != nil {
		t.Fatalf("Failed to create test endpoint: %v", err)
	}

	// 1. Create first key-auth plugin on route -> Should succeed
	body1 := CreateAuthPluginRequest{
		PluginName: "key-auth",
		Scope:      "route",
		RouteID:    &testRouteID,
		Config: map[string]interface{}{
			"key_names": []string{"X-API-Key"},
		},
	}
	bodyBytes1, _ := json.Marshal(body1)
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/auth-plugins", bytes.NewReader(bodyBytes1))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()

	handler.Create(w1, req1)
	if w1.Code != http.StatusOK && w1.Code != http.StatusCreated {
		t.Fatalf("Expected 200 or 201 for first key-auth plugin, got %d: %s", w1.Code, w1.Body.String())
	}

	// 2. Attempt to create second key-auth plugin on same route -> Must fail with HTTP 400
	body2 := CreateAuthPluginRequest{
		PluginName: "key-auth",
		Scope:      "route",
		RouteID:    &testRouteID,
	}
	bodyBytes2, _ := json.Marshal(body2)
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth-plugins", bytes.NewReader(bodyBytes2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()

	handler.Create(w2, req2)
	if w2.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 Bad Request for duplicate plugin, got %d: %s", w2.Code, w2.Body.String())
	}
	if !strings.Contains(w2.Body.String(), "already attached") {
		t.Errorf("Expected 'already attached' in error body, got: %s", w2.Body.String())
	}

	// 3. Create a heterogeneous plugin (jwt-auth) on same route -> Should succeed (Logical OR)
	body3 := CreateAuthPluginRequest{
		PluginName: "jwt-auth",
		Scope:      "route",
		RouteID:    &testRouteID,
		Config: map[string]interface{}{
			"algorithms": []string{"HS256"},
		},
	}
	bodyBytes3, _ := json.Marshal(body3)
	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/auth-plugins", bytes.NewReader(bodyBytes3))
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()

	handler.Create(w3, req3)
	if w3.Code != http.StatusOK && w3.Code != http.StatusCreated {
		t.Fatalf("Expected 200/201 for heterogeneous jwt-auth plugin, got %d: %s", w3.Code, w3.Body.String())
	}

	// Extract created jwt plugin ID from response
	var resData struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
		ID string `json:"id"`
	}
	_ = json.Unmarshal(w3.Body.Bytes(), &resData)
	jwtID := resData.Data.ID
	if jwtID == "" {
		jwtID = resData.ID
	}
	if jwtID == "" {
		// Fetch via list
		plugins, _ := reg.ListAuthPlugins("route", testRouteID)
		for _, p := range plugins {
			if p.PluginName == "jwt-auth" {
				jwtID = p.ID
				break
			}
		}
	}

	// 4. Update the jwt-auth plugin to key-auth -> Must fail with HTTP 400 (duplicate)
	updatePayload := map[string]interface{}{
		"plugin_name": "key-auth",
	}
	updateBytes, _ := json.Marshal(updatePayload)
	req4 := httptest.NewRequest(http.MethodPut, "/api/v1/auth-plugins/"+jwtID, bytes.NewReader(updateBytes))
	req4.Header.Set("Content-Type", "application/json")
	req4 = mux.SetURLVars(req4, map[string]string{"id": jwtID})
	w4 := httptest.NewRecorder()

	handler.Update(w4, req4)
	if w4.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 Bad Request when renaming plugin to duplicate name, got %d: %s", w4.Code, w4.Body.String())
	}
	if !strings.Contains(w4.Body.String(), "already attached") {
		t.Errorf("Expected 'already attached' in update error body, got: %s", w4.Body.String())
	}
}
