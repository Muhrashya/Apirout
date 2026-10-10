package executor

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"

	"github.com/apirout/apirout/internal/core/adapter"
	"github.com/apirout/apirout/internal/core/models"
	"github.com/apirout/apirout/internal/core/sqldialect"
	"github.com/apirout/apirout/internal/gateway/apiclientauth"
	"github.com/apirout/apirout/internal/gateway/registry"
)

func TestInstantCRUD_DbTypeDriver(t *testing.T) {
	tests := []struct {
		dbType     string
		wantDriver string
		wantErr    bool
	}{
		{dbType: "postgres", wantDriver: "postgres"},
		{dbType: "postgresql", wantDriver: "postgres"},
		{dbType: "mysql", wantDriver: "mysql"},
		{dbType: "mssql", wantDriver: "sqlserver"},
		{dbType: "sqlserver", wantDriver: "sqlserver"},
		{dbType: "sqlite", wantErr: true},
		{dbType: "oracle", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.dbType, func(t *testing.T) {
			driver, err := dbTypeDriver(tt.dbType)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for dbType %s, got nil", tt.dbType)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.dbType, err)
			}
			if driver != tt.wantDriver {
				t.Errorf("got driver %s, want %s", driver, tt.wantDriver)
			}
		})
	}
}

func TestInstantCRUD_QuoteIdentifier(t *testing.T) {
	if quoteIdentifier("postgres", "users") != `"users"` {
		t.Errorf("postgres quote: got %s, want %s", quoteIdentifier("postgres", "users"), `"users"`)
	}
	if quoteIdentifier("mysql", "users") != "`users`" {
		t.Errorf("mysql quote: got %s, want %s", quoteIdentifier("mysql", "users"), "`users`")
	}
	if quoteIdentifier("mssql", "users") != "[users]" {
		t.Errorf("mssql quote: got %s, want %s", quoteIdentifier("mssql", "users"), "[users]")
	}
	if quoteIdentifier("sqlserver", "orders") != "[orders]" {
		t.Errorf("sqlserver quote: got %s, want %s", quoteIdentifier("sqlserver", "orders"), "[orders]")
	}
}

func TestInstantCRUD_DialectGenerators_AllEngines(t *testing.T) {
	engines := []string{"postgres", "mysql", "mssql"}

	for _, eng := range engines {
		t.Run(eng, func(t *testing.T) {
			ad, err := adapter.GetAdapter(eng)
			if err != nil {
				t.Fatalf("failed to get adapter for %s: %v", eng, err)
			}
			gen := ad.GetQueryGenerator()

			// 1. SELECT query
			selQuery, selArgs, err := gen.BuildSelect(sqldialect.QuerySpec{
				Table:   "customers",
				Columns: []string{"id", "name", "email"},
				Filters: []sqldialect.FilterCondition{
					{Column: "status", Operator: sqldialect.OpEq, Value: "active"},
				},
				Limit:  10,
				Offset: 0,
			})
			if err != nil {
				t.Fatalf("BuildSelect failed: %v", err)
			}
			if !strings.Contains(selQuery, "customers") || len(selArgs) != 1 {
				t.Errorf("invalid select query %s with args %+v", selQuery, selArgs)
			}

			// 2. INSERT query
			insQuery, insArgs, err := gen.BuildInsert(sqldialect.InsertSpec{
				Table: "customers",
				Values: map[string]any{
					"name":  "Alice",
					"email": "alice@example.com",
				},
			})
			if err != nil {
				t.Fatalf("BuildInsert failed: %v", err)
			}
			if !strings.Contains(insQuery, "INSERT INTO") || len(insArgs) != 2 {
				t.Errorf("invalid insert query %s with args %+v", insQuery, insArgs)
			}

			// 3. UPDATE query
			updQuery, updArgs, err := gen.BuildUpdate(sqldialect.UpdateSpec{
				Table: "customers",
				Values: map[string]any{
					"name": "Alice Cooper",
				},
				Filters: []sqldialect.FilterCondition{
					{Column: "id", Operator: sqldialect.OpEq, Value: "42"},
				},
			})
			if err != nil {
				t.Fatalf("BuildUpdate failed: %v", err)
			}
			if !strings.Contains(updQuery, "UPDATE") || len(updArgs) != 2 {
				t.Errorf("invalid update query %s with args %+v", updQuery, updArgs)
			}

			// 4. DELETE query
			delQuery, delArgs, err := gen.BuildDelete(sqldialect.DeleteSpec{
				Table: "customers",
				Filters: []sqldialect.FilterCondition{
					{Column: "id", Operator: sqldialect.OpEq, Value: "42"},
				},
			})
			if err != nil {
				t.Fatalf("BuildDelete failed: %v", err)
			}
			if !strings.Contains(delQuery, "DELETE FROM") || len(delArgs) != 1 {
				t.Errorf("invalid delete query %s with args %+v", delQuery, delArgs)
			}
		})
	}
}

func TestInstantCRUD_AuthorizeConnectionAccess_BOLA(t *testing.T) {
	db, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}
	defer db.Close()

	schema := `
		CREATE TABLE connections (
			id TEXT PRIMARY KEY,
			group_id TEXT,
			name TEXT NOT NULL,
			db_type TEXT NOT NULL,
			host TEXT NOT NULL,
			port INT NOT NULL,
			db_name TEXT NOT NULL,
			username TEXT NOT NULL,
			password_encrypted TEXT NOT NULL,
			ssl_mode TEXT NOT NULL,
			ssh_enabled INT DEFAULT 0,
			ssh_host TEXT,
			ssh_port INT DEFAULT 22,
			ssh_user TEXT,
			ssh_auth_method TEXT,
			ssh_password_encrypted TEXT,
			ssh_private_key_encrypted TEXT,
			created_at DATETIME,
			updated_at DATETIME
		);
		CREATE TABLE api_clients (
			id TEXT PRIMARY KEY,
			group_id TEXT,
			username TEXT NOT NULL,
			info TEXT,
			password_hash TEXT NOT NULL,
			status TEXT DEFAULT 'active',
			created_at DATETIME,
			updated_at DATETIME
		);
		CREATE TABLE consumer_credentials (
			id TEXT PRIMARY KEY,
			consumer_id TEXT NOT NULL,
			name TEXT NOT NULL,
			key_type TEXT NOT NULL,
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

	// Insert connection for Group A, Group B, and an ungrouped/global connection (NULL group_id)
	_, err = db.Exec(`
		INSERT INTO connections (id, group_id, name, db_type, host, port, db_name, username, password_encrypted, ssl_mode, created_at, updated_at)
		VALUES ('conn-a', 'group-a', 'DB A', 'postgres', 'localhost', 5432, 'db_a', 'user', '', 'disable', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
		       ('conn-b', 'group-b', 'DB B', 'postgres', 'localhost', 5432, 'db_b', 'user', '', 'disable', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
		       ('conn-global', NULL, 'DB Global', 'postgres', 'localhost', 5432, 'db_global', 'user', '', 'disable', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`)
	if err != nil {
		t.Fatalf("failed to insert connections: %v", err)
	}

	reader := registry.NewReaderWithDB(db, "sqlite")
	apiClientAuth := apiclientauth.NewService("test-jwt-secret-key-32-chars-long!", reader, 10*time.Minute)
	handler := NewInstantCRUDHandler(reader, "test-enc-key-32-chars-long!", nil, apiClientAuth, "admin-master-secret", "")

	// 1. Unauthenticated request -> 401 Unauthorized
	reqUnauth := httptest.NewRequest("GET", "/api/v1/instant/conn-a/users", nil)
	code, err := handler.AuthorizeConnectionAccess(reqUnauth, "conn-a")
	if err == nil || code != http.StatusUnauthorized {
		t.Errorf("unauthenticated request expected 401, got code=%d err=%v", code, err)
	}

	// 2. Admin secret bypass -> 200 OK
	reqAdmin := httptest.NewRequest("GET", "/api/v1/instant/conn-b/users", nil)
	reqAdmin.Header.Set("X-Apirout-Admin-Secret", "admin-master-secret")
	code, err = handler.AuthorizeConnectionAccess(reqAdmin, "conn-b")
	if err != nil || code != http.StatusOK {
		t.Errorf("admin secret expected 200, got code=%d err=%v", code, err)
	}

	// 3. API Client in Group A generates token
	groupA := "group-a"
	tokenA, _, err := apiClientAuth.GenerateToken(&models.APIClient{
		ID:       "client-a",
		Username: "alice",
		GroupID:  &groupA,
	})
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// 3a. Client in Group A accesses Connection in Group A -> 200 OK
	reqAuthorized := httptest.NewRequest("GET", "/api/v1/instant/conn-a/users", nil)
	reqAuthorized.Header.Set("Authorization", "Bearer "+tokenA)
	code, err = handler.AuthorizeConnectionAccess(reqAuthorized, "conn-a")
	if err != nil || code != http.StatusOK {
		t.Errorf("authorized group access expected 200, got code=%d err=%v", code, err)
	}

	// 3b. Client in Group A attempts to access Connection in Group B -> 403 Forbidden (BOLA defense!)
	reqForbidden := httptest.NewRequest("GET", "/api/v1/instant/conn-b/users", nil)
	reqForbidden.Header.Set("Authorization", "Bearer "+tokenA)
	code, err = handler.AuthorizeConnectionAccess(reqForbidden, "conn-b")
	if err == nil || code != http.StatusForbidden {
		t.Errorf("BOLA mismatch expected 403, got code=%d err=%v", code, err)
	}

	// 3c. Client in Group A attempts to access Global/Ungrouped Connection -> 403 Forbidden (Global Isolation!)
	reqGlobalForbidden := httptest.NewRequest("GET", "/api/v1/instant/conn-global/users", nil)
	reqGlobalForbidden.Header.Set("Authorization", "Bearer "+tokenA)
	code, err = handler.AuthorizeConnectionAccess(reqGlobalForbidden, "conn-global")
	if err == nil || code != http.StatusForbidden {
		t.Errorf("global connection access by client expected 403, got code=%d err=%v", code, err)
	}

	// 3d. Admin secret accesses Global Connection -> 200 OK
	reqGlobalAdmin := httptest.NewRequest("GET", "/api/v1/instant/conn-global/users", nil)
	reqGlobalAdmin.Header.Set("X-Apirout-Admin-Secret", "admin-master-secret")
	code, err = handler.AuthorizeConnectionAccess(reqGlobalAdmin, "conn-global")
	if err != nil || code != http.StatusOK {
		t.Errorf("global connection access by admin secret expected 200, got code=%d err=%v", code, err)
	}

	// 4. Unknown connection ID -> 404 Not Found
	reqNotFound := httptest.NewRequest("GET", "/api/v1/instant/conn-unknown/users", nil)
	reqNotFound.Header.Set("Authorization", "Bearer "+tokenA)
	code, err = handler.AuthorizeConnectionAccess(reqNotFound, "conn-unknown")
	if err == nil || code != http.StatusNotFound {
		t.Errorf("unknown connection expected 404, got code=%d err=%v", code, err)
	}
}

