package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/apirout/apirout/internal/console/auth"
	"github.com/apirout/apirout/internal/console/registry"
	"github.com/apirout/apirout/internal/core/crypto"
	"github.com/apirout/apirout/internal/core/egress"
	"github.com/apirout/apirout/internal/core/httputil"
	"github.com/apirout/apirout/internal/core/introspection"
	"github.com/apirout/apirout/internal/core/sqldialect"
	"github.com/apirout/apirout/internal/core/targets"

	"github.com/gorilla/mux"
)

// ConnectionHandler handles connection management
type ConnectionHandler struct {
	registry      *registry.Registry
	encryptionKey string
}

// NewConnectionHandler creates a new connection handler
func NewConnectionHandler(reg *registry.Registry, encryptionKey string) *ConnectionHandler {
	return &ConnectionHandler{
		registry:      reg,
		encryptionKey: encryptionKey,
	}
}

// List returns all connections (filtered by user's groups for non-SUPER_ADMIN)
func (h *ConnectionHandler) List(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		httputil.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var results interface{}
	var err error

	if claims.Role == "SUPER_ADMIN" {
		// SUPER_ADMIN sees all with health
		results, err = h.registry.ListConnectionsWithHealth()
	} else {
		// ADMIN/MEMBER sees only their groups (no health info for now)
		groupIDs, gErr := h.registry.GetUserGroupIDs(claims.UserID)
		if gErr != nil {
			httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to get user groups")
			return
		}
		results, err = h.registry.ListConnectionsByGroupIDs(groupIDs)
	}

	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to list connections")
		return
	}
	httputil.WriteJSONData(w, http.StatusOK, results)
}

// Get returns a connection by ID
func (h *ConnectionHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		httputil.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	conn, err := h.registry.GetConnectionByID(id)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to get connection")
		return
	}
	if conn == nil {
		httputil.WriteJSONError(w, http.StatusNotFound, "Connection not found")
		return
	}

	// Check access (Read)
	if err := h.checkConnectionAccess(claims, conn, registry.PermManageQueries); err != nil {
		httputil.WriteJSONError(w, http.StatusForbidden, "No permission for this connection")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, conn)
}

// CreateConnectionRequest represents connection creation request
type CreateConnectionRequest struct {
	GroupID       *string `json:"group_id"`
	Name          string  `json:"name"`
	DBType        string  `json:"db_type"`
	Host          string  `json:"host"`
	Port          int     `json:"port"`
	DBName        string  `json:"dbname"`
	Username      string  `json:"username"`
	Password      string  `json:"password"`
	SSLMode       string  `json:"sslmode"`
	IsReadOnly    bool    `json:"is_read_only"`
	SSHEnabled    bool    `json:"ssh_enabled"`
	SSHHost       *string `json:"ssh_host"`
	SSHPort       int     `json:"ssh_port"`
	SSHUser       *string `json:"ssh_user"`
	SSHAuthMethod *string `json:"ssh_auth_method"`
	SSHPassword   *string `json:"ssh_password"`
	SSHPrivateKey *string `json:"ssh_private_key"`
}

// Create creates a new connection
func (h *ConnectionHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		httputil.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req CreateConnectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// SECURITY: Verify permission on the group
	if req.GroupID != nil {
		if claims.Role != "SUPER_ADMIN" {
			canManage, err := h.registry.CanManageResource(claims.UserID, *req.GroupID, registry.PermManageConnections)
			if err != nil {
				httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to check permissions")
				return
			}
			if !canManage {
				httputil.WriteJSONError(w, http.StatusForbidden, "No permission for this group")
				return
			}
		}
	} else if claims.Role != "SUPER_ADMIN" {
		// Only SUPER_ADMIN can create global connections
		httputil.WriteJSONError(w, http.StatusForbidden, "Only Super Admin can create global connections")
		return
	}

	if req.DBType == "redis" {
		if req.Name == "" || req.Host == "" {
			httputil.WriteJSONError(w, http.StatusBadRequest, "Connection name and host are required for Redis")
			return
		}
		if req.DBName == "" {
			req.DBName = "0"
		}
		if req.Username == "" {
			req.Username = "default"
		}
	} else if req.DBType == "sqlite" {
		if req.Name == "" || req.Host == "" {
			httputil.WriteJSONError(w, http.StatusBadRequest, "Connection name and SQLite file path are required")
			return
		}
		if req.DBName == "" {
			req.DBName = "sqlite"
		}
	} else if req.DBType == "mongodb" || req.DBType == "mongo" {
		if req.Name == "" || req.Host == "" {
			httputil.WriteJSONError(w, http.StatusBadRequest, "Connection name and MongoDB host or URI are required")
			return
		}
		if req.DBName == "" {
			req.DBName = "admin"
		}
	} else if req.Name == "" || req.DBType == "" || req.Host == "" || req.DBName == "" || req.Username == "" {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Name, db_type, host, dbname, and username are required")
		return
	}

	if req.Port == 0 {
		switch req.DBType {
		case "postgres", "postgresql":
			req.Port = 5432
		case "mysql":
			req.Port = 3306
		case "mssql", "sqlserver":
			req.Port = 1433
		case "mongodb", "mongo":
			req.Port = 27017
		case "redis":
			req.Port = 6379
		case "trino":
			req.Port = 8080
		}
	}

	// Encrypt password
	encryptedPassword, err := encryptString(req.Password, h.encryptionKey)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to encrypt password")
		return
	}

	var sshPasswordEncrypted, sshPrivateKeyEncrypted *string
	if req.SSHPassword != nil && *req.SSHPassword != "" {
		encrypted, err := encryptString(*req.SSHPassword, h.encryptionKey)
		if err != nil {
			httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to encrypt SSH password")
			return
		}
		sshPasswordEncrypted = &encrypted
	}
	if req.SSHPrivateKey != nil && *req.SSHPrivateKey != "" {
		encrypted, err := encryptString(*req.SSHPrivateKey, h.encryptionKey)
		if err != nil {
			httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to encrypt SSH private key")
			return
		}
		sshPrivateKeyEncrypted = &encrypted
	}

	conn := &registry.Connection{
		GroupID:                req.GroupID,
		Name:                   req.Name,
		DBType:                 req.DBType,
		Host:                   req.Host,
		Port:                   req.Port,
		DBName:                 req.DBName,
		Username:               req.Username,
		PasswordEncrypted:      encryptedPassword,
		SSLMode:                req.SSLMode,
		IsReadOnly:             req.IsReadOnly,
		SSHEnabled:             req.SSHEnabled,
		SSHHost:                req.SSHHost,
		SSHPort:                req.SSHPort,
		SSHUser:                req.SSHUser,
		SSHAuthMethod:          req.SSHAuthMethod,
		SSHPasswordEncrypted:   sshPasswordEncrypted,
		SSHPrivateKeyEncrypted: sshPrivateKeyEncrypted,
	}

	if err := h.registry.CreateConnection(conn); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to create connection: "+err.Error())
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, conn)
}

// Update updates a connection
func (h *ConnectionHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		httputil.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	existing, err := h.registry.GetConnectionByID(id)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to get connection")
		return
	}
	if existing == nil {
		httputil.WriteJSONError(w, http.StatusNotFound, "Connection not found")
		return
	}

	// Check access (Update)
	if err := h.checkConnectionAccess(claims, existing, registry.PermManageConnections); err != nil {
		httputil.WriteJSONError(w, http.StatusForbidden, "No permission to update this connection")
		return
	}

	var req CreateConnectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Update fields
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.DBType != "" {
		existing.DBType = req.DBType
	}
	if req.Host != "" {
		existing.Host = req.Host
	}
	if req.Port != 0 {
		existing.Port = req.Port
	}
	if req.DBName != "" {
		existing.DBName = req.DBName
	}
	if req.Username != "" {
		existing.Username = req.Username
	}
	if req.Password != "" {
		encrypted, err := encryptString(req.Password, h.encryptionKey)
		if err != nil {
			httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to encrypt password")
			return
		}
		existing.PasswordEncrypted = encrypted
	}
	if req.SSLMode != "" {
		existing.SSLMode = req.SSLMode
	}
	existing.GroupID = req.GroupID
	existing.IsReadOnly = req.IsReadOnly
	existing.SSHEnabled = req.SSHEnabled
	existing.SSHHost = req.SSHHost
	existing.SSHPort = req.SSHPort
	existing.SSHUser = req.SSHUser
	existing.SSHAuthMethod = req.SSHAuthMethod

	if req.SSHPassword != nil && *req.SSHPassword != "" {
		encrypted, _ := encryptString(*req.SSHPassword, h.encryptionKey)
		existing.SSHPasswordEncrypted = &encrypted
	}
	if req.SSHPrivateKey != nil && *req.SSHPrivateKey != "" {
		encrypted, _ := encryptString(*req.SSHPrivateKey, h.encryptionKey)
		existing.SSHPrivateKeyEncrypted = &encrypted
	}

	if err := h.registry.UpdateConnection(existing); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to update connection: "+err.Error())
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]string{"message": "Connection updated successfully"})
}

// Delete deletes a connection
func (h *ConnectionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		httputil.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	existing, err := h.registry.GetConnectionByID(id)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to get connection")
		return
	}
	if existing == nil {
		httputil.WriteJSONError(w, http.StatusNotFound, "Connection not found")
		return
	}

	// Check access (Delete)
	if err := h.checkConnectionAccess(claims, existing, registry.PermManageConnections); err != nil {
		httputil.WriteJSONError(w, http.StatusForbidden, "No permission to delete this connection")
		return
	}

	if err := h.registry.DeleteConnection(id); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to delete connection: "+err.Error())
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]string{"message": "Connection deleted successfully"})
}

// TestConnectionResponse represents test connection response
type TestConnectionResponse struct {
	Success   bool    `json:"success"`
	LatencyMs int     `json:"latency_ms,omitempty"`
	Error     *string `json:"error,omitempty"`
}

// TestRawRequest represents payload to test an unsaved connection
type TestRawRequest struct {
	DBType   string `json:"db_type"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	DBName   string `json:"dbname"`
	Username string `json:"username"`
	Password string `json:"password"`
	SSLMode  string `json:"sslmode"`
}

// TestRaw tests an unsaved connection directly
func (h *ConnectionHandler) TestRaw(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		httputil.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req TestRawRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.DBType == "redis" {
		if req.DBName == "" {
			req.DBName = "0"
		}
	} else if req.DBType == "sqlite" {
		if req.DBName == "" {
			req.DBName = "sqlite"
		}
	} else if (req.DBType == "mongodb" || req.DBType == "mongo") && req.DBName == "" {
		req.DBName = "admin"
	}

	if req.Port == 0 {
		switch req.DBType {
		case "postgres", "postgresql":
			req.Port = 5432
		case "mysql":
			req.Port = 3306
		case "mssql", "sqlserver":
			req.Port = 1433
		case "mongodb", "mongo":
			req.Port = 27017
		case "redis":
			req.Port = 6379
		case "trino":
			req.Port = 8080
		}
	}

	// SSRF Protection: Prevent probing internal loopback, link-local metadata, or private IP ranges (SEC-15)
	if err := egress.ValidateSafeHost(req.Host); err != nil {
		errMsg := fmt.Sprintf("Security violation: %v", err)
		httputil.WriteJSON(w, http.StatusBadRequest, TestConnectionResponse{Success: false, Error: &errMsg})
		return
	}

	connString, err := targets.BuildConnectionString(req.DBType, req.Host, req.Port, req.DBName, req.Username, req.Password, req.SSLMode)
	if err != nil {
		errMsg := "Invalid configuration: " + err.Error()
		httputil.WriteJSON(w, http.StatusOK, TestConnectionResponse{Success: false, Error: &errMsg})
		return
	}

	connector, err := targets.GetConnector(req.DBType)
	if err != nil {
		errMsg := "Database driver not found: " + req.DBType
		httputil.WriteJSON(w, http.StatusOK, TestConnectionResponse{Success: false, Error: &errMsg})
		return
	}

	latency, err := connector.TestConnection(connString)
	if err != nil {
		errMsg := err.Error()
		httputil.WriteJSON(w, http.StatusOK, TestConnectionResponse{Success: false, Error: &errMsg})
		return
	}

	latencyMs := int(latency.Milliseconds())
	httputil.WriteJSON(w, http.StatusOK, TestConnectionResponse{Success: true, LatencyMs: latencyMs})
}

// Test tests a connection
func (h *ConnectionHandler) Test(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		httputil.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	conn, err := h.registry.GetConnectionByID(id)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to get connection: "+err.Error())
		return
	}
	if conn == nil {
		httputil.WriteJSONError(w, http.StatusNotFound, "Connection not found")
		return
	}

	// Check access (Read/Use)
	if err := h.checkConnectionAccess(claims, conn, registry.PermManageQueries); err != nil {
		httputil.WriteJSONError(w, http.StatusForbidden, "No permission for this connection")
		return
	}

	// Decrypt password
	password, err := decryptString(conn.PasswordEncrypted, h.encryptionKey)
	if err != nil {
		password = conn.PasswordEncrypted // fallback
	}

	// Build connection string
	connString, err := targets.BuildConnectionString(conn.DBType, conn.Host, conn.Port, conn.DBName, conn.Username, password, conn.SSLMode)
	if err != nil {
		genericMsg := "Invalid configuration"
		httputil.WriteJSON(w, http.StatusOK, TestConnectionResponse{Success: false, Error: &genericMsg})
		return
	}

	// Get connector and test
	connector, err := targets.GetConnector(conn.DBType)
	if err != nil {
		genericMsg := "Database driver not found"
		httputil.WriteJSON(w, http.StatusOK, TestConnectionResponse{Success: false, Error: &genericMsg})
		return
	}

	// Check access
	if err := h.checkConnectionAccess(claims, conn, registry.PermManageQueries); err != nil {
		httputil.WriteJSONError(w, http.StatusForbidden, "No permission for this connection")
		return
	}

	latency, err := connector.TestConnection(connString)
	now := time.Now()

	if err != nil {
		errStr := err.Error()

		// Update health status
		h.registry.UpsertConnectionHealth(&registry.ConnectionHealth{
			ConnectionID:     conn.ID,
			LastSuccess:      false,
			LastErrorMessage: &errStr,
			LastCheckedAt:    &now,
		})

		// SECURITY: Return generic error to client, log actual error
		// Log error (would ideally use a logger)
		// fmt.Printf("Connection test failed: %s\n", errStr)

		genericMsg := "Connection failed. Check server logs."
		httputil.WriteJSON(w, http.StatusOK, TestConnectionResponse{Success: false, Error: &genericMsg})
		return
	}

	latencyMs := int(latency.Milliseconds())

	// Update health status
	h.registry.UpsertConnectionHealth(&registry.ConnectionHealth{
		ConnectionID:  conn.ID,
		LastSuccess:   true,
		LastLatencyMs: &latencyMs,
		LastCheckedAt: &now,
	})

	httputil.WriteJSON(w, http.StatusOK, TestConnectionResponse{Success: true, LatencyMs: latencyMs})
}

func (h *ConnectionHandler) checkConnectionAccess(claims *auth.Claims, conn *registry.Connection, requiredPerm string) error {
	if claims.Role == "SUPER_ADMIN" {
		return nil
	}
	if conn.GroupID == nil {
		// Public/Global connections - logic TBD, for now assume read-only/use for non-superadmin
		if requiredPerm == registry.PermManageQueries { // Read/Use
			return nil
		}
		// Deny modification of global connections
		return fmt.Errorf("forbidden")
	}
	canAccess, err := h.registry.CanManageResource(claims.UserID, *conn.GroupID, requiredPerm)
	if err != nil {
		return err
	}
	if !canAccess {
		return fmt.Errorf("forbidden")
	}
	return nil
}

func encryptString(plaintext, key string) (string, error) {
	return crypto.EncryptPassword(crypto.NormalizeKey(key), plaintext)
}

func decryptString(encrypted, key string) (string, error) {
	return crypto.DecryptPassword(crypto.NormalizeKey(key), encrypted)
}

// Introspect introspects the tables and columns of a connection
func (h *ConnectionHandler) Introspect(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	conn, err := h.registry.GetConnectionByID(id)
	if err != nil || conn == nil {
		httputil.WriteJSONError(w, http.StatusNotFound, "Connection not found")
		return
	}

	password, err := decryptString(conn.PasswordEncrypted, h.encryptionKey)
	if err != nil {
		password = ""
	}

	connStr, err := targets.BuildConnectionString(conn.DBType, conn.Host, conn.Port, conn.DBName, conn.Username, password, conn.SSLMode)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, fmt.Sprintf("Introspection not supported for db type: %s", conn.DBType))
		return
	}

	schema, err := introspection.Introspect(r.Context(), conn.DBType, connStr)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, fmt.Sprintf("Introspection failed: %v", err))
		return
	}

	if pub := h.registry.GetPublisher(); pub != nil {
		_ = pub.PublishConnectionChanged("UPDATE", id)
	}

	httputil.WriteJSON(w, http.StatusOK, schema)
}

// GenerateEndpoints auto-generates queries and endpoints for all tables (Hasura style)
func (h *ConnectionHandler) GenerateEndpoints(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	conn, err := h.registry.GetConnectionByID(id)
	if err != nil || conn == nil {
		httputil.WriteJSONError(w, http.StatusNotFound, "Connection not found")
		return
	}

	// Parse optional is_public flag from request body (defaults to false for secure-by-default)
	type GenerateRequest struct {
		IsPublic *bool `json:"is_public"`
	}
	var genReq GenerateRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&genReq)
	}

	isPublic := false // Secure-by-default: protected with API key
	if genReq.IsPublic != nil {
		isPublic = *genReq.IsPublic
	}

	password, err := decryptString(conn.PasswordEncrypted, h.encryptionKey)
	if err != nil {
		password = ""
	}

	connStr, err := targets.BuildConnectionString(conn.DBType, conn.Host, conn.Port, conn.DBName, conn.Username, password, conn.SSLMode)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, fmt.Sprintf("Unsupported db type: %s", conn.DBType))
		return
	}

	schema, err := introspection.Introspect(r.Context(), conn.DBType, connStr)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, fmt.Sprintf("Introspection error: %v", err))
		return
	}

	gen, _ := sqldialect.GetGenerator(conn.DBType)

	createdCount := 0
	for _, table := range schema.Tables {
		quotedTable := table.Name
		if gen != nil {
			quotedTable = gen.QuoteIdentifier(table.Name)
		} else {
			quotedTable = fmt.Sprintf(`"%s"`, table.Name)
		}

		queryName := fmt.Sprintf("List %s (%s)", table.Name, conn.Name)
		sqlText := fmt.Sprintf("SELECT * FROM %s", quotedTable)
		countSQL := fmt.Sprintf("SELECT COUNT(*) FROM %s", quotedTable)
		desc := fmt.Sprintf("Auto-generated instant query for %s", table.Name)

		q := &registry.Query{
			GroupID:       conn.GroupID,
			ConnectionID:  &id,
			Name:          queryName,
			Description:   &desc,
			SQLText:       sqlText,
			CountSQLText:  &countSQL,
			ParamsSchema:  "{}",
			DefaultLimit:  20,
			PreviewLimit:  20,
			PreviewOffset: 0,
		}
		if err := h.registry.CreateQuery(q); err != nil {
			continue
		}

		path := fmt.Sprintf("/api/v1/%s/%s", strings.ToLower(conn.Name), strings.ToLower(table.Name))
		path = strings.ReplaceAll(path, " ", "-")
		epDesc := fmt.Sprintf("Instant REST API for table %s", table.Name)

		ep := &registry.Endpoint{
			GroupID:             conn.GroupID,
			Method:              "GET",
			Path:                path,
			QueryID:             &q.ID,
			Enabled:             true,
			IsPublic:            isPublic,
			RateLimitEnabled:    true,
			RateLimitRPM:        60,
			RateLimitBurst:      10,
			DefaultLimit:        20,
			MaxLimit:            100,
			ReturnTotalCount:    true,
			AllowDynamicFilters: true,
			AllowedDomains:      "*",
			CorsOrigins:         "*",
			Description:         &epDesc,
		}
		if err := h.registry.CreateEndpoint(ep); err == nil {
			createdCount++
		}
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"status":            true,
		"message":           fmt.Sprintf("Auto-generated %d REST endpoints for %s", createdCount, conn.Name),
		"endpoints_created": createdCount,
		"total_tables":      len(schema.Tables),
	})
}
