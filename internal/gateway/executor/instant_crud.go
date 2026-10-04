package executor

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"github.com/apirout/apirout/internal/core/crypto"
	"github.com/apirout/apirout/internal/gateway/apiclientauth"
	"github.com/apirout/apirout/internal/gateway/cache"
	"github.com/apirout/apirout/internal/core/httputil"
	"github.com/apirout/apirout/internal/core/introspection"
	"github.com/apirout/apirout/internal/gateway/registry"
)

var identifierRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// InstantCRUDHandler handles dynamic Hasura-style REST CRUD operations on target databases
type InstantCRUDHandler struct {
	reader        *registry.Reader
	encryptionKey string
	connPool      *cache.ConnectionPool
	apiClientAuth *apiclientauth.Service
	adminSecret   string
	jwtSecret     string
}

// NewInstantCRUDHandler creates a new instant CRUD handler
func NewInstantCRUDHandler(reader *registry.Reader, encryptionKey string, connPool *cache.ConnectionPool, apiClientAuth *apiclientauth.Service, adminSecret, jwtSecret string) *InstantCRUDHandler {
	return &InstantCRUDHandler{
		reader:        reader,
		encryptionKey: encryptionKey,
		connPool:      connPool,
		apiClientAuth: apiClientAuth,
		adminSecret:   adminSecret,
		jwtSecret:     jwtSecret,
	}
}

// AuthenticateRequest verifies that the request carries a valid API key, Bearer JWT, or Admin Secret
func (h *InstantCRUDHandler) AuthenticateRequest(r *http.Request) error {
	// 1. Check Admin Secret (Constant-time comparison)
	if h.adminSecret != "" {
		secretHeader := r.Header.Get("X-Apirout-Admin-Secret")
		if secretHeader == "" {
			secretHeader = r.Header.Get("X-Admin-Secret")
		}
		if secretHeader != "" && subtle.ConstantTimeCompare([]byte(secretHeader), []byte(h.adminSecret)) == 1 {
			return nil
		}
	}

	// 2. Check API Key
	token, apiKey := extractTokenOrKey(r)
	if apiKey != "" {
		consumer, _, err := h.reader.ValidateAPIKey(apiKey)
		if err == nil && consumer != nil {
			return nil
		}
	}

	// 3. Check JWT token (API client token or Admin console token)
	if token != "" {
		if h.apiClientAuth != nil {
			claims, err := h.apiClientAuth.ValidateToken(token)
			if err == nil && claims != nil {
				return nil
			}
		}
		if h.jwtSecret != "" {
			parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method")
				}
				return []byte(h.jwtSecret), nil
			})
			if err == nil && parsed.Valid {
				return nil
			}
		}
	}

	return fmt.Errorf("authentication required: provide valid X-API-Key, Bearer token, or X-Admin-Secret")
}

// getDB retrieves a database connection from the connection pool or falls back to direct connection
func (h *InstantCRUDHandler) getDB(connID, dbType, connStr string) (*sql.DB, func(), error) {
	if h.connPool != nil {
		db, err := h.connPool.Get(connID, dbType, connStr)
		if err != nil {
			return nil, nil, err
		}
		return db, func() {}, nil
	}
	db, err := sql.Open(dbTypeDriver(dbType), connStr)
	if err != nil {
		return nil, nil, err
	}
	return db, func() { _ = db.Close() }, nil
}

// GetSchema handles GET /api/v1/instant/{connId}/schema
func (h *InstantCRUDHandler) GetSchema(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	connID := vars["connId"]

	connStr, dbType, err := h.getTargetConn(r.Context(), connID)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusNotFound, err.Error())
		return
	}

	schema, err := introspection.Introspect(r.Context(), dbType, connStr)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to introspect database: %v", err))
		return
	}

	httputil.WriteJSON(w, http.StatusOK, schema)
}

// QueryTable handles GET /api/v1/instant/{connId}/{table}
func (h *InstantCRUDHandler) QueryTable(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	connID := vars["connId"]
	tableName := vars["table"]

	if !identifierRegex.MatchString(tableName) {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid table name")
		return
	}

	connStr, dbType, err := h.getTargetConn(r.Context(), connID)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusNotFound, err.Error())
		return
	}

	db, closeDB, err := h.getDB(connID, dbType, connStr)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to connect to database: "+err.Error())
		return
	}
	defer closeDB()

	// Parse query params
	q := r.URL.Query()
	limit := 20
	if l := q.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 500 {
			limit = parsed
		}
	}
	offset := 0
	if o := q.Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Columns selection
	columns := "*"
	if sel := q.Get("select"); sel != "" {
		selectedCols := strings.Split(sel, ",")
		validCols := make([]string, 0, len(selectedCols))
		for _, col := range selectedCols {
			col = strings.TrimSpace(col)
			if identifierRegex.MatchString(col) {
				validCols = append(validCols, quoteIdentifier(dbType, col))
			}
		}
		if len(validCols) > 0 {
			columns = strings.Join(validCols, ", ")
		}
	}

	// Filtering & Where clause
	whereClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	for key, values := range q {
		if key == "limit" || key == "offset" || key == "select" || key == "order" || key == "sort" || key == "page" {
			continue
		}
		if !identifierRegex.MatchString(key) {
			continue
		}

		colName := quoteIdentifier(dbType, key)
		for _, val := range values {
			placeholder := fmt.Sprintf("$%d", argIdx)
			if dbType == "mysql" {
				placeholder = "?"
			}

			if strings.HasPrefix(val, "eq.") {
				whereClauses = append(whereClauses, fmt.Sprintf("%s = %s", colName, placeholder))
				args = append(args, strings.TrimPrefix(val, "eq."))
				argIdx++
			} else if strings.HasPrefix(val, "neq.") {
				whereClauses = append(whereClauses, fmt.Sprintf("%s != %s", colName, placeholder))
				args = append(args, strings.TrimPrefix(val, "neq."))
				argIdx++
			} else if strings.HasPrefix(val, "gt.") {
				whereClauses = append(whereClauses, fmt.Sprintf("%s > %s", colName, placeholder))
				args = append(args, strings.TrimPrefix(val, "gt."))
				argIdx++
			} else if strings.HasPrefix(val, "gte.") {
				whereClauses = append(whereClauses, fmt.Sprintf("%s >= %s", colName, placeholder))
				args = append(args, strings.TrimPrefix(val, "gte."))
				argIdx++
			} else if strings.HasPrefix(val, "lt.") {
				whereClauses = append(whereClauses, fmt.Sprintf("%s < %s", colName, placeholder))
				args = append(args, strings.TrimPrefix(val, "lt."))
				argIdx++
			} else if strings.HasPrefix(val, "lte.") {
				whereClauses = append(whereClauses, fmt.Sprintf("%s <= %s", colName, placeholder))
				args = append(args, strings.TrimPrefix(val, "lte."))
				argIdx++
			} else if strings.HasPrefix(val, "like.") {
				whereClauses = append(whereClauses, fmt.Sprintf("%s LIKE %s", colName, placeholder))
				args = append(args, strings.TrimPrefix(val, "like."))
				argIdx++
			} else if strings.HasPrefix(val, "ilike.") {
				if dbType == "postgres" {
					whereClauses = append(whereClauses, fmt.Sprintf("%s ILIKE %s", colName, placeholder))
				} else {
					whereClauses = append(whereClauses, fmt.Sprintf("LOWER(%s) LIKE LOWER(%s)", colName, placeholder))
				}
				args = append(args, strings.TrimPrefix(val, "ilike."))
				argIdx++
			} else if val == "is.null" {
				whereClauses = append(whereClauses, fmt.Sprintf("%s IS NULL", colName))
			} else if val == "is.notnull" {
				whereClauses = append(whereClauses, fmt.Sprintf("%s IS NOT NULL", colName))
			} else {
				whereClauses = append(whereClauses, fmt.Sprintf("%s = %s", colName, placeholder))
				args = append(args, val)
				argIdx++
			}
		}
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Order by
	orderSQL := ""
	orderParam := q.Get("order")
	if orderParam == "" {
		orderParam = q.Get("sort")
	}
	if orderParam != "" {
		parts := strings.Split(orderParam, ".")
		col := parts[0]
		direction := "ASC"
		if strings.HasPrefix(col, "-") {
			col = strings.TrimPrefix(col, "-")
			direction = "DESC"
		} else if len(parts) > 1 && strings.ToUpper(parts[1]) == "DESC" {
			direction = "DESC"
		}
		if identifierRegex.MatchString(col) {
			orderSQL = fmt.Sprintf(" ORDER BY %s %s", quoteIdentifier(dbType, col), direction)
		}
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s%s", quoteIdentifier(dbType, tableName), whereSQL)
	var total int64
	_ = db.QueryRowContext(r.Context(), countQuery, args...).Scan(&total)

	// Build pagination placeholders
	limitPlaceholder := fmt.Sprintf("$%d", argIdx)
	offsetPlaceholder := fmt.Sprintf("$%d", argIdx+1)
	if dbType == "mysql" {
		limitPlaceholder = "?"
		offsetPlaceholder = "?"
	}
	args = append(args, limit, offset)

	querySQL := fmt.Sprintf(
		"SELECT %s FROM %s%s%s LIMIT %s OFFSET %s",
		columns, quoteIdentifier(dbType, tableName), whereSQL, orderSQL, limitPlaceholder, offsetPlaceholder,
	)

	rows, err := db.QueryContext(r.Context(), querySQL, args...)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, fmt.Sprintf("Query execution failed: %v", err))
		return
	}
	defer rows.Close()

	colNames, err := rows.Columns()
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to inspect columns")
		return
	}

	results := []map[string]interface{}{}
	for rows.Next() {
		colValues := make([]interface{}, len(colNames))
		colPtrs := make([]interface{}, len(colNames))
		for i := range colValues {
			colPtrs[i] = &colValues[i]
		}
		if err := rows.Scan(colPtrs...); err != nil {
			continue
		}
		record := make(map[string]interface{})
		for i, name := range colNames {
			val := colValues[i]
			if b, ok := val.([]byte); ok {
				record[name] = string(b)
			} else {
				record[name] = val
			}
		}
		results = append(results, record)
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"data": results,
		"meta": map[string]interface{}{
			"total":    total,
			"limit":    limit,
			"offset":   offset,
			"returned": len(results),
		},
	})
}

// GetRow handles GET /api/v1/instant/{connId}/{table}/{id}
func (h *InstantCRUDHandler) GetRow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	connID := vars["connId"]
	tableName := vars["table"]
	id := vars["id"]

	if !identifierRegex.MatchString(tableName) {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid table name")
		return
	}

	connStr, dbType, err := h.getTargetConn(r.Context(), connID)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusNotFound, err.Error())
		return
	}

	schema, err := introspection.Introspect(r.Context(), dbType, connStr)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to introspect table")
		return
	}

	pk := "id"
	for _, t := range schema.Tables {
		if t.Name == tableName && t.PrimaryKey != "" {
			pk = t.PrimaryKey
			break
		}
	}

	db, closeDB, err := h.getDB(connID, dbType, connStr)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Database connection error: "+err.Error())
		return
	}
	defer closeDB()

	paramPlaceholder := "$1"
	if dbType == "mysql" {
		paramPlaceholder = "?"
	}

	querySQL := fmt.Sprintf("SELECT * FROM %s WHERE %s = %s LIMIT 1",
		quoteIdentifier(dbType, tableName), quoteIdentifier(dbType, pk), paramPlaceholder)

	rows, err := db.QueryContext(r.Context(), querySQL, id)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, fmt.Sprintf("Query error: %v", err))
		return
	}
	defer rows.Close()

	colNames, _ := rows.Columns()
	if !rows.Next() {
		httputil.WriteJSONError(w, http.StatusNotFound, fmt.Sprintf("Record with %s=%s not found", pk, id))
		return
	}

	colValues := make([]interface{}, len(colNames))
	colPtrs := make([]interface{}, len(colNames))
	for i := range colValues {
		colPtrs[i] = &colValues[i]
	}
	_ = rows.Scan(colPtrs...)

	record := make(map[string]interface{})
	for i, name := range colNames {
		val := colValues[i]
		if b, ok := val.([]byte); ok {
			record[name] = string(b)
		} else {
			record[name] = val
		}
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"data": record,
	})
}

// InsertRow handles POST /api/v1/instant/{connId}/{table}
func (h *InstantCRUDHandler) InsertRow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	connID := vars["connId"]
	tableName := vars["table"]

	if !identifierRegex.MatchString(tableName) {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid table name")
		return
	}

	connStr, dbType, err := h.getTargetConn(r.Context(), connID)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusNotFound, err.Error())
		return
	}

	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if len(body) == 0 {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Empty payload")
		return
	}

	cols := []string{}
	placeholders := []string{}
	values := []interface{}{}
	idx := 1

	for k, v := range body {
		if !identifierRegex.MatchString(k) {
			continue
		}
		cols = append(cols, quoteIdentifier(dbType, k))
		if dbType == "mysql" {
			placeholders = append(placeholders, "?")
		} else {
			placeholders = append(placeholders, fmt.Sprintf("$%d", idx))
		}
		values = append(values, v)
		idx++
	}

	db, closeDB, err := h.getDB(connID, dbType, connStr)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Database connection error: "+err.Error())
		return
	}
	defer closeDB()

	insertSQL := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		quoteIdentifier(dbType, tableName),
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
	)

	_, err = db.ExecContext(r.Context(), insertSQL, values...)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, fmt.Sprintf("Insert failed: %v", err))
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, map[string]interface{}{
		"status":  "created",
		"message": "Row successfully inserted",
		"data":    body,
	})
}

// UpdateRow handles PUT/PATCH /api/v1/instant/{connId}/{table}/{id}
func (h *InstantCRUDHandler) UpdateRow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	connID := vars["connId"]
	tableName := vars["table"]
	id := vars["id"]

	if !identifierRegex.MatchString(tableName) {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid table name")
		return
	}

	connStr, dbType, err := h.getTargetConn(r.Context(), connID)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusNotFound, err.Error())
		return
	}

	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	schema, _ := introspection.Introspect(r.Context(), dbType, connStr)
	pk := "id"
	if schema != nil {
		for _, t := range schema.Tables {
			if t.Name == tableName && t.PrimaryKey != "" {
				pk = t.PrimaryKey
				break
			}
		}
	}

	setClauses := []string{}
	values := []interface{}{}
	idx := 1

	for k, v := range body {
		if !identifierRegex.MatchString(k) || k == pk {
			continue
		}
		if dbType == "mysql" {
			setClauses = append(setClauses, fmt.Sprintf("%s = ?", quoteIdentifier(dbType, k)))
		} else {
			setClauses = append(setClauses, fmt.Sprintf("%s = $%d", quoteIdentifier(dbType, k), idx))
		}
		values = append(values, v)
		idx++
	}

	if len(setClauses) == 0 {
		httputil.WriteJSONError(w, http.StatusBadRequest, "No fields to update")
		return
	}

	db, closeDB, err := h.getDB(connID, dbType, connStr)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Database connection error: "+err.Error())
		return
	}
	defer closeDB()

	pkPlaceholder := fmt.Sprintf("$%d", idx)
	if dbType == "mysql" {
		pkPlaceholder = "?"
	}
	values = append(values, id)

	updateSQL := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s = %s",
		quoteIdentifier(dbType, tableName),
		strings.Join(setClauses, ", "),
		quoteIdentifier(dbType, pk),
		pkPlaceholder,
	)

	res, err := db.ExecContext(r.Context(), updateSQL, values...)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, fmt.Sprintf("Update failed: %v", err))
		return
	}

	rowsAff, _ := res.RowsAffected()
	if rowsAff == 0 {
		httputil.WriteJSONError(w, http.StatusNotFound, "Record not found")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "updated",
		"message": "Row successfully updated",
		"id":      id,
	})
}

// DeleteRow handles DELETE /api/v1/instant/{connId}/{table}/{id}
func (h *InstantCRUDHandler) DeleteRow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	connID := vars["connId"]
	tableName := vars["table"]
	id := vars["id"]

	if !identifierRegex.MatchString(tableName) {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid table name")
		return
	}

	connStr, dbType, err := h.getTargetConn(r.Context(), connID)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusNotFound, err.Error())
		return
	}

	schema, _ := introspection.Introspect(r.Context(), dbType, connStr)
	pk := "id"
	if schema != nil {
		for _, t := range schema.Tables {
			if t.Name == tableName && t.PrimaryKey != "" {
				pk = t.PrimaryKey
				break
			}
		}
	}

	db, closeDB, err := h.getDB(connID, dbType, connStr)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Database connection error: "+err.Error())
		return
	}
	defer closeDB()

	placeholder := "$1"
	if dbType == "mysql" {
		placeholder = "?"
	}

	deleteSQL := fmt.Sprintf("DELETE FROM %s WHERE %s = %s",
		quoteIdentifier(dbType, tableName), quoteIdentifier(dbType, pk), placeholder)

	res, err := db.ExecContext(r.Context(), deleteSQL, id)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, fmt.Sprintf("Delete failed: %v", err))
		return
	}

	rowsAff, _ := res.RowsAffected()
	if rowsAff == 0 {
		httputil.WriteJSONError(w, http.StatusNotFound, "Record not found")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "deleted",
		"message": "Record successfully deleted",
		"id":      id,
	})
}

func (h *InstantCRUDHandler) getTargetConn(ctx context.Context, connID string) (string, string, error) {
	conn, err := h.reader.GetConnectionByID(connID)
	if err != nil || conn == nil {
		return "", "", fmt.Errorf("connection %s not found: %w", connID, err)
	}

	password, err := crypto.DecryptPassword(crypto.NormalizeKey(h.encryptionKey), conn.PasswordEncrypted)
	if err != nil {
		password = ""
	}

	var connStr string
	switch conn.DBType {
	case "postgres", "postgresql":
		ssl := conn.SSLMode
		if ssl == "" {
			ssl = "disable"
		}
		connStr = fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s&connect_timeout=5",
			conn.Username, password, conn.Host, conn.Port, conn.DBName, ssl)
	case "mysql":
		connStr = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?timeout=5s&parseTime=true",
			conn.Username, password, conn.Host, conn.Port, conn.DBName)
	default:
		return "", "", fmt.Errorf("unsupported connection type: %s", conn.DBType)
	}

	return connStr, conn.DBType, nil
}

func dbTypeDriver(dbType string) string {
	switch dbType {
	case "postgres", "postgresql":
		return "postgres"
	case "mysql":
		return "mysql"
	default:
		return "postgres"
	}
}

func quoteIdentifier(dbType, ident string) string {
	if dbType == "mysql" {
		return "`" + ident + "`"
	}
	return `"` + ident + `"`
}

// ExecuteTableCRUD dispatches a request directly to the table CRUD handlers for a given connection
func (h *InstantCRUDHandler) ExecuteTableCRUD(w http.ResponseWriter, r *http.Request, connID, tableName, id string) {
	vars := map[string]string{
		"connId": connID,
		"table":  tableName,
	}
	if id != "" {
		vars["id"] = id
	}
	r = mux.SetURLVars(r, vars)

	switch r.Method {
	case http.MethodGet:
		if id != "" {
			h.GetRow(w, r)
		} else {
			h.QueryTable(w, r)
		}
	case http.MethodPost:
		h.InsertRow(w, r)
	case http.MethodPut, http.MethodPatch:
		h.UpdateRow(w, r)
	case http.MethodDelete:
		h.DeleteRow(w, r)
	default:
		httputil.WriteJSONError(w, http.StatusMethodNotAllowed, "Method not allowed for instant CRUD")
	}
}
