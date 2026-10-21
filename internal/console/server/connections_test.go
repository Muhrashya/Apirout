package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
	"github.com/apirout/apirout/internal/console/auth"
	"github.com/apirout/apirout/internal/console/registry"
	"github.com/apirout/apirout/internal/core/sqldialect"
)

func TestGenerateEndpoints_DialectIdentifierQuoting(t *testing.T) {
	keywords := []string{"order", "select", "user", "group", "limit"}

	dialects := []struct {
		dbType   string
		expected map[string]string
	}{
		{
			dbType: "postgres",
			expected: map[string]string{
				"order":  `"order"`,
				"select": `"select"`,
				"user":   `"user"`,
				"group":  `"group"`,
				"limit":  `"limit"`,
			},
		},
		{
			dbType: "mysql",
			expected: map[string]string{
				"order":  "`order`",
				"select": "`select`",
				"user":   "`user`",
				"group":  "`group`",
				"limit":  "`limit`",
			},
		},
		{
			dbType: "mssql",
			expected: map[string]string{
				"order":  `[order]`,
				"select": `[select]`,
				"user":   `[user]`,
				"group":  `[group]`,
				"limit":  `[limit]`,
			},
		},
		{
			dbType: "trino",
			expected: map[string]string{
				"order":  `"order"`,
				"select": `"select"`,
				"user":   `"user"`,
				"group":  `"group"`,
				"limit":  `"limit"`,
			},
		},
	}

	for _, d := range dialects {
		t.Run(d.dbType, func(t *testing.T) {
			gen, err := sqldialect.GetGenerator(d.dbType)
			if err != nil {
				t.Fatalf("failed to get generator for %s: %v", d.dbType, err)
			}

			for _, kw := range keywords {
				quoted := gen.QuoteIdentifier(kw)
				want := d.expected[kw]
				if quoted != want {
					t.Errorf("[%s] QuoteIdentifier(%q) = %q, want %q", d.dbType, kw, quoted, want)
				}

				sqlText := fmt.Sprintf("SELECT * FROM %s", quoted)
				expectedSQL := fmt.Sprintf("SELECT * FROM %s", want)
				if sqlText != expectedSQL {
					t.Errorf("[%s] generated SQL = %q, want %q", d.dbType, sqlText, expectedSQL)
				}
			}
		})
	}
}

func TestConnectionHandler_ReadOnlyFieldPersistence(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_conn_ro.db")

	reg, err := registry.NewWithSQLite(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test registry: %v", err)
	}
	defer reg.Close()

	if err := reg.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	handler := NewConnectionHandler(reg, "test-enc-key-32-chars-long!")

	claims := &auth.Claims{
		UserID: "super-admin-id",
		Email:  "admin@example.com",
		Role:   "SUPER_ADMIN",
	}

	// 1. Create a Read-Only Connection
	reqBody := CreateConnectionRequest{
		Name:        "Read Only Analytics DB",
		DBType:      "postgres",
		Host:        "localhost",
		Port:        5432,
		DBName:      "analytics",
		Username:    "ro_user",
		Password:    "secret",
		IsReadOnly:  true,
		SSLMode:     "disable",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/connections", bytes.NewReader(bodyBytes))
	req = req.WithContext(context.WithValue(req.Context(), auth.UserContextKey, claims))
	rec := httptest.NewRecorder()

	handler.Create(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	var createdConn registry.Connection
	if err := json.NewDecoder(rec.Body).Decode(&createdConn); err != nil {
		t.Fatalf("failed to decode created connection: %v", err)
	}

	if !createdConn.IsReadOnly {
		t.Errorf("expected created connection is_read_only to be true, got false")
	}

	// 2. Fetch the connection by ID and verify persistence
	reqGet := httptest.NewRequest("GET", "/api/v1/connections/"+createdConn.ID, nil)
	reqGet = reqGet.WithContext(context.WithValue(reqGet.Context(), auth.UserContextKey, claims))
	reqGet = mux.SetURLVars(reqGet, map[string]string{"id": createdConn.ID})
	recGet := httptest.NewRecorder()

	handler.Get(recGet, reqGet)
	if recGet.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", recGet.Code)
	}

	var fetchedConn registry.Connection
	if err := json.NewDecoder(recGet.Body).Decode(&fetchedConn); err != nil {
		t.Fatalf("failed to decode fetched connection: %v", err)
	}

	if !fetchedConn.IsReadOnly {
		t.Errorf("expected fetched connection is_read_only to be true from database, got false")
	}
}
