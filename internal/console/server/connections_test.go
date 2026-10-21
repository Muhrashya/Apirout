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
	"time"

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

func TestConnectionHandler_GroupRBACUpdateGuard(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_conn_rbac.db")

	reg, err := registry.NewWithSQLite(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test registry: %v", err)
	}
	defer reg.Close()

	if err := reg.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}
	if err := reg.MigrateSchema(); err != nil {
		t.Fatalf("Failed to migrate schema: %v", err)
	}

	encryptionKey := "01234567890123456789012345678901"
	handler := NewConnectionHandler(reg, encryptionKey)

	// Create Group A and Group B
	grpA, err := reg.CreateGroup("Group A")
	if err != nil {
		t.Fatalf("failed to create group A: %v", err)
	}
	grpB, err := reg.CreateGroup("Group B")
	if err != nil {
		t.Fatalf("failed to create group B: %v", err)
	}

	// Create user in Group A with manage permissions
	userA, err := reg.CreateUser("userA@example.com", "password123", "ADMIN")
	if err != nil {
		t.Fatalf("failed to create user A: %v", err)
	}
	err = reg.AssignUserToGroup(userA.ID, grpA.ID, registry.UserGroup{
		CanManageConnections: true,
		CanManageQueries:     true,
	})
	if err != nil {
		t.Fatalf("failed to assign user A to group A: %v", err)
	}

	claimsUserA := &auth.Claims{
		UserID: userA.ID,
		Email:  userA.Email,
		Role:   "ADMIN",
	}
	claimsSuperAdmin := &auth.Claims{
		UserID: "super-1",
		Email:  "root@example.com",
		Role:   "SUPER_ADMIN",
	}

	connA := &registry.Connection{
		Name:      "Group A DB",
		DBType:    "sqlite",
		Host:      filepath.Join(tempDir, "group_a.db"),
		DBName:    "main",
		Username:  "admin",
		GroupID:   &grpA.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := reg.CreateConnection(connA); err != nil {
		t.Fatalf("failed to create connection in group A: %v", err)
	}
	connID := connA.ID

	// 1. User A attempts to nullify GroupID (make connection global) -> 403 Forbidden
	reqBodyGlobal := CreateConnectionRequest{
		Name:    "Group A DB Global Attempt",
		DBType:  "sqlite",
		Host:    filepath.Join(tempDir, "group_a.db"),
		GroupID: nil,
	}
	bodyBytes, _ := json.Marshal(reqBodyGlobal)
	reqUpdate := httptest.NewRequest("PUT", "/api/v1/connections/"+connID, bytes.NewReader(bodyBytes))
	reqUpdate = reqUpdate.WithContext(context.WithValue(reqUpdate.Context(), auth.UserContextKey, claimsUserA))
	reqUpdate = mux.SetURLVars(reqUpdate, map[string]string{"id": connID})
	recUpdate := httptest.NewRecorder()

	handler.Update(recUpdate, reqUpdate)
	if recUpdate.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden when non-superadmin nullifies GroupID, got %d: %s", recUpdate.Code, recUpdate.Body.String())
	}

	// 2. User A attempts to reassign connection to Group B (unauthorized group) -> 403 Forbidden
	reqBodyGrpB := CreateConnectionRequest{
		Name:    "Group A DB Move to B",
		DBType:  "sqlite",
		Host:    filepath.Join(tempDir, "group_a.db"),
		GroupID: &grpB.ID,
	}
	bodyBytes, _ = json.Marshal(reqBodyGrpB)
	reqUpdate = httptest.NewRequest("PUT", "/api/v1/connections/"+connID, bytes.NewReader(bodyBytes))
	reqUpdate = reqUpdate.WithContext(context.WithValue(reqUpdate.Context(), auth.UserContextKey, claimsUserA))
	reqUpdate = mux.SetURLVars(reqUpdate, map[string]string{"id": connID})
	recUpdate = httptest.NewRecorder()

	handler.Update(recUpdate, reqUpdate)
	if recUpdate.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden when moving connection to unauthorized group B, got %d: %s", recUpdate.Code, recUpdate.Body.String())
	}

	// 3. User A updates connection staying within Group A -> 200 OK
	reqBodySameGrp := CreateConnectionRequest{
		Name:    "Group A DB Updated",
		DBType:  "sqlite",
		Host:    filepath.Join(tempDir, "group_a.db"),
		GroupID: &grpA.ID,
	}
	bodyBytes, _ = json.Marshal(reqBodySameGrp)
	reqUpdate = httptest.NewRequest("PUT", "/api/v1/connections/"+connID, bytes.NewReader(bodyBytes))
	reqUpdate = reqUpdate.WithContext(context.WithValue(reqUpdate.Context(), auth.UserContextKey, claimsUserA))
	reqUpdate = mux.SetURLVars(reqUpdate, map[string]string{"id": connID})
	recUpdate = httptest.NewRecorder()

	handler.Update(recUpdate, reqUpdate)
	if recUpdate.Code != http.StatusOK {
		t.Errorf("expected 200 OK when updating connection within same group, got %d: %s", recUpdate.Code, recUpdate.Body.String())
	}

	// 4. Super Admin can make connection global (GroupID = nil) -> 200 OK
	bodyBytes, _ = json.Marshal(reqBodyGlobal)
	reqUpdate = httptest.NewRequest("PUT", "/api/v1/connections/"+connID, bytes.NewReader(bodyBytes))
	reqUpdate = reqUpdate.WithContext(context.WithValue(reqUpdate.Context(), auth.UserContextKey, claimsSuperAdmin))
	reqUpdate = mux.SetURLVars(reqUpdate, map[string]string{"id": connID})
	recUpdate = httptest.NewRecorder()

	handler.Update(recUpdate, reqUpdate)
	if recUpdate.Code != http.StatusOK {
		t.Errorf("expected 200 OK when Super Admin nullifies GroupID, got %d: %s", recUpdate.Code, recUpdate.Body.String())
	}
}

func TestConnectionHandler_SSRFPolicy(t *testing.T) {
	claimsSuperAdmin := &auth.Claims{
		UserID: "super-1",
		Email:  "root@example.com",
		Role:   "SUPER_ADMIN",
	}

	handler := NewConnectionHandler(nil, "01234567890123456789012345678901")

	tests := []struct {
		name       string
		host       string
		wantBlocked bool
		errMsg     string
	}{
		{
			name:        "Cloud metadata 169.254.169.254",
			host:        "169.254.169.254",
			wantBlocked: true,
			errMsg:      "egress to cloud metadata service is prohibited",
		},
		{
			name:        "Loopback 127.0.0.1",
			host:        "127.0.0.1",
			wantBlocked: true,
			errMsg:      "egress to loopback address is prohibited",
		},
		{
			name:        "Private IP 10.16.3.172 (Internal Enterprise Network)",
			host:        "10.16.3.172",
			wantBlocked: false,
		},
		{
			name:        "Private IP 192.168.1.100",
			host:        "192.168.1.100",
			wantBlocked: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			payload := TestRawRequest{
				DBType:   "postgres",
				Host:     tc.host,
				Port:     5432,
				DBName:   "testdb",
				Username: "user",
			}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequest("POST", "/api/v1/connections/test", bytes.NewReader(body))
			req = req.WithContext(context.WithValue(req.Context(), auth.UserContextKey, claimsSuperAdmin))
			rec := httptest.NewRecorder()

			handler.TestRaw(rec, req)

			if tc.wantBlocked {
				if rec.Code != http.StatusBadRequest {
					t.Errorf("expected status 400 for %s, got %d", tc.host, rec.Code)
				}
				var resp TestConnectionResponse
				_ = json.Unmarshal(rec.Body.Bytes(), &resp)
				if resp.Error == nil || !strings.Contains(*resp.Error, tc.errMsg) {
					t.Errorf("expected error containing %q, got %+v", tc.errMsg, resp.Error)
				}
			} else {
				// Should pass egress validation and attempt connection (won't fail with security violation)
				if rec.Code == http.StatusBadRequest {
					var resp TestConnectionResponse
					_ = json.Unmarshal(rec.Body.Bytes(), &resp)
					if resp.Error != nil && strings.Contains(*resp.Error, "Security violation") {
						t.Errorf("unexpected security violation for safe private host %s: %s", tc.host, *resp.Error)
					}
				}
			}
		})
	}
}

func TestConnectionHandler_FieldAliasNormalization(t *testing.T) {
	// Test that JSON using snake_case (db_name, ssl_mode) is normalized to DBName and SSLMode
	rawJSON := []byte(`{
		"name": "Trino Lakehouse",
		"db_type": "trino",
		"host": "trino-internal.bgn.go.id",
		"port": 443,
		"db_name": "tpch",
		"username": "bgn_user",
		"ssl_mode": "require"
	}`)

	var createReq CreateConnectionRequest
	if err := json.Unmarshal(rawJSON, &createReq); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	createReq.normalize()

	if createReq.DBName != "tpch" {
		t.Errorf("expected DBName normalized to 'tpch', got %q", createReq.DBName)
	}
	if createReq.SSLMode != "require" {
		t.Errorf("expected SSLMode normalized to 'require', got %q", createReq.SSLMode)
	}

	var testReq TestRawRequest
	if err := json.Unmarshal(rawJSON, &testReq); err != nil {
		t.Fatalf("failed to unmarshal JSON into TestRawRequest: %v", err)
	}
	testReq.normalize()

	if testReq.DBName != "tpch" {
		t.Errorf("expected TestRawRequest DBName normalized to 'tpch', got %q", testReq.DBName)
	}
	if testReq.SSLMode != "require" {
		t.Errorf("expected TestRawRequest SSLMode normalized to 'require', got %q", testReq.SSLMode)
	}
}



