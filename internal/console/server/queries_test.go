package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/apirout/apirout/internal/console/auth"
	"github.com/apirout/apirout/internal/console/executor"
	"github.com/apirout/apirout/internal/console/registry"
)

func TestQueryHandler_PreviewValidationAndReadOnlyGuards(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_queries_preview.db")

	reg, err := registry.NewWithSQLite(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test registry: %v", err)
	}
	defer reg.Close()

	if err := reg.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	execService := executor.NewService(reg, "test-enc-key-32-chars-long!")
	handler := NewQueryHandler(reg, execService)

	claims := &auth.Claims{
		UserID: "super-admin-id",
		Email:  "admin@example.com",
		Role:   "SUPER_ADMIN",
	}

	// 1. Create a regular read-write connection
	rwConn := &registry.Connection{
		Name:              "RW DB",
		DBType:            "postgres",
		Host:              "localhost",
		Port:              5432,
		DBName:            "rw_db",
		Username:          "postgres",
		PasswordEncrypted: "",
		SSLMode:           "disable",
		IsReadOnly:        false,
	}
	if err := reg.CreateConnection(rwConn); err != nil {
		t.Fatalf("failed to create RW connection: %v", err)
	}

	// 2. Create a read-only connection
	roConn := &registry.Connection{
		Name:              "RO DB",
		DBType:            "postgres",
		Host:              "localhost",
		Port:              5432,
		DBName:            "ro_db",
		Username:          "postgres",
		PasswordEncrypted: "",
		SSLMode:           "disable",
		IsReadOnly:        true,
	}
	if err := reg.CreateConnection(roConn); err != nil {
		t.Fatalf("failed to create RO connection: %v", err)
	}

	mutatingQueries := []struct {
		name string
		sql  string
	}{
		{"DROP TABLE", "DROP TABLE users;"},
		{"UPDATE", "UPDATE users SET active = 0 WHERE id = 1;"},
		{"DELETE", "DELETE FROM users WHERE id = 1;"},
		{"INSERT", "INSERT INTO users (id, name) VALUES (2, 'mallory');"},
		{"ALTER TABLE", "ALTER TABLE users ADD COLUMN hacked INT;"},
	}

	// Test 1: On RW connection, mutating statements are rejected with 400 Bad Request
	for _, tc := range mutatingQueries {
		t.Run("RW_Connection_"+tc.name, func(t *testing.T) {
			reqBody := PreviewRequest{
				ConnectionID: rwConn.ID,
				SQL:          tc.sql,
				Limit:        10,
				Offset:       0,
			}
			bodyBytes, _ := json.Marshal(reqBody)

			req := httptest.NewRequest("POST", "/api/v1/queries/preview", bytes.NewReader(bodyBytes))
			req = req.WithContext(context.WithValue(req.Context(), auth.UserContextKey, claims))
			rec := httptest.NewRecorder()

			handler.Preview(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("[%s on RW connection] expected 400 Bad Request, got %d (body: %s)", tc.name, rec.Code, rec.Body.String())
			}
		})
	}

	// Test 2: On Read-Only connection, mutating statements are rejected with 405 Method Not Allowed
	for _, tc := range mutatingQueries {
		t.Run("RO_Connection_"+tc.name, func(t *testing.T) {
			reqBody := PreviewRequest{
				ConnectionID: roConn.ID,
				SQL:          tc.sql,
				Limit:        10,
				Offset:       0,
			}
			bodyBytes, _ := json.Marshal(reqBody)

			req := httptest.NewRequest("POST", "/api/v1/queries/preview", bytes.NewReader(bodyBytes))
			req = req.WithContext(context.WithValue(req.Context(), auth.UserContextKey, claims))
			rec := httptest.NewRecorder()

			handler.Preview(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("[%s on RO connection] expected 405 Method Not Allowed, got %d (body: %s)", tc.name, rec.Code, rec.Body.String())
			}
			if rec.Header().Get("Allow") != "GET, HEAD, OPTIONS" {
				t.Errorf("[%s on RO connection] expected Allow header 'GET, HEAD, OPTIONS', got %q", tc.name, rec.Header().Get("Allow"))
			}
		})
	}
}
