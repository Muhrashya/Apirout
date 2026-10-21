package executor

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"github.com/apirout/apirout/internal/core/adapter"
	"github.com/apirout/apirout/internal/core/cache"
	"github.com/apirout/apirout/internal/core/crypto"
	"github.com/apirout/apirout/internal/core/introspection"
	"github.com/apirout/apirout/internal/core/sqldialect"
	"github.com/apirout/apirout/internal/gateway/apiclientauth"
	"github.com/apirout/apirout/internal/gateway/auth"
	"github.com/apirout/apirout/internal/gateway/registry"
	"github.com/apirout/apirout/internal/gateway/responder"
)

var identifierRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// InstantCRUDHandler handles dynamic Hasura-style REST CRUD operations on target databases
type InstantCRUDHandler struct {
	reader        *registry.Reader
	encryptionKey string
	connPool      *cache.ManagedConnectionPool
	apiClientAuth *apiclientauth.Service
	adminSecret   string
	jwtSecret     string
}

// NewInstantCRUDHandler creates a new instant CRUD handler
func NewInstantCRUDHandler(reader *registry.Reader, encryptionKey string, connPool *cache.ManagedConnectionPool, apiClientAuth *apiclientauth.Service, adminSecret, jwtSecret string) *InstantCRUDHandler {
	return &InstantCRUDHandler{
		reader:        reader,
		encryptionKey: encryptionKey,
		connPool:      connPool,
		apiClientAuth: apiClientAuth,
		adminSecret:   adminSecret,
		jwtSecret:     jwtSecret,
	}
}

// AuthorizeConnectionAccess verifies authentication and enforces object-level authorization (BOLA defense)
// ensuring the caller has permissions to access the target connection ID.
func (h *InstantCRUDHandler) AuthorizeConnectionAccess(r *http.Request, connID string) (int, error) {
	// 1. Check Admin Secret (Constant-time comparison) - Grants full super admin bypass
	if h.adminSecret != "" {
		secretHeader := r.Header.Get("X-Apirout-Admin-Secret")
		if secretHeader == "" {
			secretHeader = r.Header.Get("X-Admin-Secret")
		}
		if secretHeader != "" && subtle.ConstantTimeCompare([]byte(secretHeader), []byte(h.adminSecret)) == 1 {
			return http.StatusOK, nil
		}
	}

	// 2. Fetch target connection if connID provided and reader configured
	var targetGroupID *string
	var connFound bool
	if h.reader != nil && connID != "" {
		conn, err := h.reader.GetConnectionByID(connID)
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("lookup connection %s failed: %w", connID, err)
		}
		if conn == nil {
			return http.StatusNotFound, fmt.Errorf("connection %s not found", connID)
		}
		targetGroupID = conn.GroupID
		connFound = true
	}

	// 3. Check API Key
	token, apiKey := auth.ExtractTokenOrKey(r)
	if apiKey != "" {
		if h.reader == nil {
			return http.StatusUnauthorized, fmt.Errorf("registry reader unavailable for authentication")
		}
		consumer, _, err := h.reader.ValidateAPIKey(apiKey)
		if err != nil || consumer == nil {
			return http.StatusUnauthorized, fmt.Errorf("invalid or revoked API key")
		}

		// Object-level authorization check:
		if connFound && (targetGroupID == nil || *targetGroupID == "") {
			slog.Warn("instant crud global connection access attempt blocked",
				"reason", "consumer cannot access ungrouped/global connection",
				"consumer_id", consumer.ID,
				"conn_id", connID,
			)
			return http.StatusForbidden, fmt.Errorf("access forbidden: global connection requires Super Admin or Master Admin Secret")
		}
		if targetGroupID != nil && *targetGroupID != "" {
			if consumer.GroupID == nil || *consumer.GroupID != *targetGroupID {
				slog.Warn("instant crud BOLA access attempt blocked",
					"reason", "consumer group mismatch",
					"consumer_id", consumer.ID,
					"conn_id", connID,
					"target_group_id", *targetGroupID,
				)
				return http.StatusForbidden, fmt.Errorf("access forbidden: consumer is not authorized for connection %s", connID)
			}
		}
		return http.StatusOK, nil
	}

	// 4. Check JWT token (API client token or Admin console token)
	if token != "" {
		if h.apiClientAuth != nil {
			claims, err := h.apiClientAuth.ValidateToken(token)
			if err == nil && claims != nil {
				// Object-level authorization check for API Client:
				if connFound && (targetGroupID == nil || *targetGroupID == "") {
					slog.Warn("instant crud global connection access attempt blocked",
						"reason", "api client cannot access ungrouped/global connection",
						"client_id", claims.APIClientID,
						"conn_id", connID,
					)
					return http.StatusForbidden, fmt.Errorf("access forbidden: global connection requires Super Admin or Master Admin Secret")
				}
				if targetGroupID != nil && *targetGroupID != "" {
					if claims.GroupID == "" || claims.GroupID != *targetGroupID {
						slog.Warn("instant crud BOLA access attempt blocked",
							"reason", "api client group mismatch",
							"client_id", claims.APIClientID,
							"conn_id", connID,
							"target_group_id", *targetGroupID,
						)
						return http.StatusForbidden, fmt.Errorf("access forbidden: api client is not authorized for connection %s", connID)
					}
				}
				return http.StatusOK, nil
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
				if connFound && (targetGroupID == nil || *targetGroupID == "") {
					if mapClaims, ok := parsed.Claims.(jwt.MapClaims); ok {
						role, _ := mapClaims["role"].(string)
						if role != "SUPER_ADMIN" {
							return http.StatusForbidden, fmt.Errorf("access forbidden: global connection requires Super Admin")
						}
					}
				}
				return http.StatusOK, nil
			}
		}
	}

	return http.StatusUnauthorized, fmt.Errorf("authentication required: provide valid X-API-Key, Bearer token, or X-Admin-Secret")
}

// AuthenticateRequest verifies that the request carries a valid API key, Bearer JWT, or Admin Secret
func (h *InstantCRUDHandler) AuthenticateRequest(r *http.Request) error {
	vars := mux.Vars(r)
	connID := vars["connId"]
	_, err := h.AuthorizeConnectionAccess(r, connID)
	return err
}

// getDB retrieves a database connection from the connection pool or falls back to direct connection
func (h *InstantCRUDHandler) getDB(connID, dbType, connStr string) (*sql.DB, func(), error) {
	ad, err := adapter.GetAdapter(dbType)
	if err != nil {
		return nil, nil, fmt.Errorf("unsupported database type: %s", dbType)
	}

	if h.connPool != nil {
		db, err := h.connPool.Get(connID, dbType, connStr)
		if err != nil {
			return nil, nil, err
		}
		return db, func() {}, nil
	}
	db, err := ad.Open(connStr)
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
		responder.Error(w, r, http.StatusNotFound, responder.ErrCodeNotFound, err.Error())
		return
	}

	schema, err := introspection.Introspect(r.Context(), dbType, connStr)
	if err != nil {
		responder.Error(w, r, http.StatusInternalServerError, responder.ErrCodeInternalError, fmt.Sprintf("Failed to introspect database: %v", err))
		return
	}

	responder.JSON(w, r, http.StatusOK, schema)
}

func parseInstantFilters(q url.Values) []sqldialect.FilterCondition {
	var filters []sqldialect.FilterCondition
	for key, values := range q {
		if key == "limit" || key == "offset" || key == "select" || key == "order" || key == "sort" || key == "page" {
			continue
		}
		if !identifierRegex.MatchString(key) {
			continue
		}

		for _, val := range values {
			if strings.HasPrefix(val, "eq.") {
				filters = append(filters, sqldialect.FilterCondition{Column: key, Operator: sqldialect.OpEq, Value: strings.TrimPrefix(val, "eq.")})
			} else if strings.HasPrefix(val, "neq.") {
				filters = append(filters, sqldialect.FilterCondition{Column: key, Operator: sqldialect.OpNeq, Value: strings.TrimPrefix(val, "neq.")})
			} else if strings.HasPrefix(val, "gt.") {
				filters = append(filters, sqldialect.FilterCondition{Column: key, Operator: sqldialect.OpGt, Value: strings.TrimPrefix(val, "gt.")})
			} else if strings.HasPrefix(val, "gte.") {
				filters = append(filters, sqldialect.FilterCondition{Column: key, Operator: sqldialect.OpGte, Value: strings.TrimPrefix(val, "gte.")})
			} else if strings.HasPrefix(val, "lt.") {
				filters = append(filters, sqldialect.FilterCondition{Column: key, Operator: sqldialect.OpLt, Value: strings.TrimPrefix(val, "lt.")})
			} else if strings.HasPrefix(val, "lte.") {
				filters = append(filters, sqldialect.FilterCondition{Column: key, Operator: sqldialect.OpLte, Value: strings.TrimPrefix(val, "lte.")})
			} else if strings.HasPrefix(val, "like.") {
				filters = append(filters, sqldialect.FilterCondition{Column: key, Operator: sqldialect.OpLike, Value: strings.TrimPrefix(val, "like.")})
			} else if strings.HasPrefix(val, "ilike.") {
				filters = append(filters, sqldialect.FilterCondition{Column: key, Operator: sqldialect.OpIlike, Value: strings.TrimPrefix(val, "ilike.")})
			} else if strings.HasPrefix(val, "in.(") && strings.HasSuffix(val, ")") {
				inner := strings.TrimSuffix(strings.TrimPrefix(val, "in.("), ")")
				filters = append(filters, sqldialect.FilterCondition{Column: key, Operator: sqldialect.OpIn, Value: strings.Split(inner, ",")})
			} else if val == "is.null" {
				filters = append(filters, sqldialect.FilterCondition{Column: key, Operator: sqldialect.OpIsNull})
			} else if val == "is.notnull" || val == "is.not_null" || val == "not.is.null" {
				filters = append(filters, sqldialect.FilterCondition{Column: key, Operator: sqldialect.OpNotNull})
			} else {
				filters = append(filters, sqldialect.FilterCondition{Column: key, Operator: sqldialect.OpEq, Value: val})
			}
		}
	}
	return filters
}

// QueryTable handles GET /api/v1/instant/{connId}/{table}
func (h *InstantCRUDHandler) QueryTable(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	connID := vars["connId"]
	tableName := vars["table"]

	if !identifierRegex.MatchString(tableName) {
		responder.Error(w, r, http.StatusBadRequest, responder.ErrCodeBadRequest, "Invalid table name")
		return
	}

	connStr, dbType, err := h.getTargetConn(r.Context(), connID)
	if err != nil {
		responder.Error(w, r, http.StatusNotFound, responder.ErrCodeNotFound, err.Error())
		return
	}

	db, closeDB, err := h.getDB(connID, dbType, connStr)
	if err != nil {
		responder.Error(w, r, http.StatusInternalServerError, responder.ErrCodeInternalError, "Failed to connect to database: "+err.Error())
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
	var validCols []string
	if sel := q.Get("select"); sel != "" {
		selectedCols := strings.Split(sel, ",")
		for _, col := range selectedCols {
			col = strings.TrimSpace(col)
			if identifierRegex.MatchString(col) {
				validCols = append(validCols, col)
			}
		}
	}

	gen, err := sqldialect.GetGenerator(dbType)
	if err != nil {
		responder.Error(w, r, http.StatusBadRequest, responder.ErrCodeBadRequest, "Unsupported database dialect: "+dbType)
		return
	}

	filters := parseInstantFilters(q)

	// Order by
	var orderBys []sqldialect.OrderBy
	orderParam := q.Get("order")
	if orderParam == "" {
		orderParam = q.Get("sort")
	}
	if orderParam != "" {
		parts := strings.Split(orderParam, ".")
		col := parts[0]
		direction := sqldialect.OrderAsc
		if strings.HasPrefix(col, "-") {
			col = strings.TrimPrefix(col, "-")
			direction = sqldialect.OrderDesc
		} else if len(parts) > 1 && strings.ToUpper(parts[1]) == "DESC" {
			direction = sqldialect.OrderDesc
		}
		if identifierRegex.MatchString(col) {
			orderBys = append(orderBys, sqldialect.OrderBy{Column: col, Direction: direction})
		}
	}

	// Count total
	countSpec := sqldialect.QuerySpec{
		Table:   tableName,
		Columns: []string{"COUNT(*)"},
		Filters: filters,
	}
	countSQL, countArgs, err := gen.BuildSelect(countSpec)
	if err != nil {
		responder.Error(w, r, http.StatusInternalServerError, responder.ErrCodeInternalError, "Failed to assemble count query: "+err.Error())
		return
	}
	var total int64
	_ = db.QueryRowContext(r.Context(), countSQL, countArgs...).Scan(&total)

	// Select rows
	querySpec := sqldialect.QuerySpec{
		Table:   tableName,
		Columns: validCols,
		Filters: filters,
		OrderBy: orderBys,
		Limit:   limit,
		Offset:  offset,
	}
	querySQL, args, err := gen.BuildSelect(querySpec)
	if err != nil {
		responder.Error(w, r, http.StatusInternalServerError, responder.ErrCodeInternalError, "Failed to assemble query: "+err.Error())
		return
	}

	rows, err := db.QueryContext(r.Context(), querySQL, args...)
	if err != nil {
		responder.Error(w, r, http.StatusInternalServerError, responder.ErrCodeInternalError, fmt.Sprintf("Query execution failed: %v", err))
		return
	}
	defer rows.Close()

	colNames, err := rows.Columns()
	if err != nil {
		responder.Error(w, r, http.StatusInternalServerError, responder.ErrCodeInternalError, "Failed to inspect columns")
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

	responder.JSON(w, r, http.StatusOK, map[string]interface{}{
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
		responder.Error(w, r, http.StatusBadRequest, responder.ErrCodeBadRequest, "Invalid table name")
		return
	}

	connStr, dbType, err := h.getTargetConn(r.Context(), connID)
	if err != nil {
		responder.Error(w, r, http.StatusNotFound, responder.ErrCodeNotFound, err.Error())
		return
	}

	schema, err := introspection.Introspect(r.Context(), dbType, connStr)
	if err != nil {
		responder.Error(w, r, http.StatusInternalServerError, responder.ErrCodeInternalError, "Failed to introspect table")
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
		responder.Error(w, r, http.StatusInternalServerError, responder.ErrCodeInternalError, "Database connection error: "+err.Error())
		return
	}
	defer closeDB()

	gen, err := sqldialect.GetGenerator(dbType)
	if err != nil {
		responder.Error(w, r, http.StatusBadRequest, responder.ErrCodeBadRequest, "Unsupported database: "+dbType)
		return
	}

	querySQL, args, err := gen.BuildSelect(sqldialect.QuerySpec{
		Table: tableName,
		Filters: []sqldialect.FilterCondition{
			{Column: pk, Operator: sqldialect.OpEq, Value: id},
		},
		Limit: 1,
	})
	if err != nil {
		responder.Error(w, r, http.StatusInternalServerError, responder.ErrCodeInternalError, "Failed to assemble query: "+err.Error())
		return
	}

	rows, err := db.QueryContext(r.Context(), querySQL, args...)
	if err != nil {
		responder.Error(w, r, http.StatusInternalServerError, responder.ErrCodeInternalError, fmt.Sprintf("Query error: %v", err))
		return
	}
	defer rows.Close()

	colNames, _ := rows.Columns()
	if !rows.Next() {
		responder.Error(w, r, http.StatusNotFound, responder.ErrCodeNotFound, fmt.Sprintf("Record with %s=%s not found", pk, id))
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

	responder.JSON(w, r, http.StatusOK, map[string]interface{}{
		"data": record,
	})
}

// InsertRow handles POST /api/v1/instant/{connId}/{table}
func (h *InstantCRUDHandler) InsertRow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	connID := vars["connId"]
	tableName := vars["table"]

	if h.reader != nil && connID != "" {
		conn, err := h.reader.GetConnectionByID(connID)
		if err == nil && conn != nil && conn.IsReadOnly {
			w.Header().Set("Allow", "GET, HEAD, OPTIONS")
			responder.Error(w, r, http.StatusMethodNotAllowed, responder.ErrCodeMethodNotAllowed, "Connection is configured as read-only: mutating operations are prohibited", map[string]string{"Allow": "GET, HEAD, OPTIONS"})
			return
		}
	}

	if !identifierRegex.MatchString(tableName) {
		responder.Error(w, r, http.StatusBadRequest, responder.ErrCodeBadRequest, "Invalid table name")
		return
	}

	connStr, dbType, err := h.getTargetConn(r.Context(), connID)
	if err != nil {
		responder.Error(w, r, http.StatusNotFound, responder.ErrCodeNotFound, err.Error())
		return
	}

	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		responder.Error(w, r, http.StatusBadRequest, responder.ErrCodeBadRequest, "Invalid JSON payload")
		return
	}

	if len(body) == 0 {
		responder.Error(w, r, http.StatusBadRequest, responder.ErrCodeBadRequest, "Empty payload")
		return
	}

	validValues := make(map[string]any)
	for k, v := range body {
		if identifierRegex.MatchString(k) {
			validValues[k] = v
		}
	}

	if len(validValues) == 0 {
		responder.Error(w, r, http.StatusBadRequest, responder.ErrCodeBadRequest, "No valid columns provided")
		return
	}

	db, closeDB, err := h.getDB(connID, dbType, connStr)
	if err != nil {
		responder.Error(w, r, http.StatusInternalServerError, responder.ErrCodeInternalError, "Database connection error: "+err.Error())
		return
	}
	defer closeDB()

	gen, err := sqldialect.GetGenerator(dbType)
	if err != nil {
		responder.Error(w, r, http.StatusBadRequest, responder.ErrCodeBadRequest, "Unsupported database: "+dbType)
		return
	}

	insertSQL, values, err := gen.BuildInsert(sqldialect.InsertSpec{
		Table:  tableName,
		Values: validValues,
	})
	if err != nil {
		responder.Error(w, r, http.StatusInternalServerError, responder.ErrCodeInternalError, fmt.Sprintf("Insert failed: %v", err))
		return
	}

	_, err = db.ExecContext(r.Context(), insertSQL, values...)
	if err != nil {
		responder.Error(w, r, http.StatusInternalServerError, responder.ErrCodeInternalError, fmt.Sprintf("Insert failed: %v", err))
		return
	}

	responder.JSON(w, r, http.StatusCreated, map[string]interface{}{
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

	if h.reader != nil && connID != "" {
		conn, err := h.reader.GetConnectionByID(connID)
		if err == nil && conn != nil && conn.IsReadOnly {
			w.Header().Set("Allow", "GET, HEAD, OPTIONS")
			responder.Error(w, r, http.StatusMethodNotAllowed, responder.ErrCodeMethodNotAllowed, "Connection is configured as read-only: mutating operations are prohibited", map[string]string{"Allow": "GET, HEAD, OPTIONS"})
			return
		}
	}

	if !identifierRegex.MatchString(tableName) {
		responder.Error(w, r, http.StatusBadRequest, responder.ErrCodeBadRequest, "Invalid table name")
		return
	}

	connStr, dbType, err := h.getTargetConn(r.Context(), connID)
	if err != nil {
		responder.Error(w, r, http.StatusNotFound, responder.ErrCodeNotFound, err.Error())
		return
	}

	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		responder.Error(w, r, http.StatusBadRequest, responder.ErrCodeBadRequest, "Invalid JSON body")
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

	validValues := make(map[string]any)
	for k, v := range body {
		if identifierRegex.MatchString(k) && k != pk {
			validValues[k] = v
		}
	}

	if len(validValues) == 0 {
		responder.Error(w, r, http.StatusBadRequest, responder.ErrCodeBadRequest, "No fields to update")
		return
	}

	db, closeDB, err := h.getDB(connID, dbType, connStr)
	if err != nil {
		responder.Error(w, r, http.StatusInternalServerError, responder.ErrCodeInternalError, "Database connection error: "+err.Error())
		return
	}
	defer closeDB()

	gen, err := sqldialect.GetGenerator(dbType)
	if err != nil {
		responder.Error(w, r, http.StatusBadRequest, responder.ErrCodeBadRequest, "Unsupported database: "+dbType)
		return
	}

	updateSQL, values, err := gen.BuildUpdate(sqldialect.UpdateSpec{
		Table:  tableName,
		Values: validValues,
		Filters: []sqldialect.FilterCondition{
			{Column: pk, Operator: sqldialect.OpEq, Value: id},
		},
	})
	if err != nil {
		if dbType == "trino" {
			responder.Error(w, r, http.StatusMethodNotAllowed, responder.ErrCodeMethodNotAllowed, err.Error())
			return
		}
		responder.Error(w, r, http.StatusInternalServerError, responder.ErrCodeInternalError, fmt.Sprintf("Update failed: %v", err))
		return
	}

	res, err := db.ExecContext(r.Context(), updateSQL, values...)
	if err != nil {
		responder.Error(w, r, http.StatusInternalServerError, responder.ErrCodeInternalError, fmt.Sprintf("Update failed: %v", err))
		return
	}

	rowsAff, _ := res.RowsAffected()
	if rowsAff == 0 {
		responder.Error(w, r, http.StatusNotFound, responder.ErrCodeNotFound, "Record not found")
		return
	}

	responder.JSON(w, r, http.StatusOK, map[string]interface{}{
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

	if h.reader != nil && connID != "" {
		conn, err := h.reader.GetConnectionByID(connID)
		if err == nil && conn != nil && conn.IsReadOnly {
			w.Header().Set("Allow", "GET, HEAD, OPTIONS")
			responder.Error(w, r, http.StatusMethodNotAllowed, responder.ErrCodeMethodNotAllowed, "Connection is configured as read-only: mutating operations are prohibited", map[string]string{"Allow": "GET, HEAD, OPTIONS"})
			return
		}
	}

	if !identifierRegex.MatchString(tableName) {
		responder.Error(w, r, http.StatusBadRequest, responder.ErrCodeBadRequest, "Invalid table name")
		return
	}

	connStr, dbType, err := h.getTargetConn(r.Context(), connID)
	if err != nil {
		responder.Error(w, r, http.StatusNotFound, responder.ErrCodeNotFound, err.Error())
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
		responder.Error(w, r, http.StatusInternalServerError, responder.ErrCodeInternalError, "Database connection error: "+err.Error())
		return
	}
	defer closeDB()

	gen, err := sqldialect.GetGenerator(dbType)
	if err != nil {
		responder.Error(w, r, http.StatusBadRequest, responder.ErrCodeBadRequest, "Unsupported database: "+dbType)
		return
	}

	deleteSQL, args, err := gen.BuildDelete(sqldialect.DeleteSpec{
		Table: tableName,
		Filters: []sqldialect.FilterCondition{
			{Column: pk, Operator: sqldialect.OpEq, Value: id},
		},
	})
	if err != nil {
		responder.Error(w, r, http.StatusInternalServerError, responder.ErrCodeInternalError, fmt.Sprintf("Delete failed: %v", err))
		return
	}

	res, err := db.ExecContext(r.Context(), deleteSQL, args...)
	if err != nil {
		responder.Error(w, r, http.StatusInternalServerError, responder.ErrCodeInternalError, fmt.Sprintf("Delete failed: %v", err))
		return
	}

	rowsAff, _ := res.RowsAffected()
	if rowsAff == 0 {
		responder.Error(w, r, http.StatusNotFound, responder.ErrCodeNotFound, "Record not found")
		return
	}

	responder.JSON(w, r, http.StatusOK, map[string]interface{}{
		"status":  "deleted",
		"message": "Record successfully deleted",
		"id":      id,
	})
}

func (h *InstantCRUDHandler) getTargetConn(ctx context.Context, connID string) (string, string, error) {
	conn, err := h.reader.GetConnectionByID(connID)
	if err != nil {
		return "", "", fmt.Errorf("connection %s lookup failed: %w", connID, err)
	}
	if conn == nil {
		return "", "", fmt.Errorf("connection %s not found", connID)
	}

	password, err := crypto.DecryptPassword(crypto.NormalizeKey(h.encryptionKey), conn.PasswordEncrypted)
	if err != nil {
		password = ""
	}

	ad, err := adapter.GetAdapter(conn.DBType)
	if err != nil {
		return "", "", fmt.Errorf("unsupported connection type: %s", conn.DBType)
	}

	connStr := ad.BuildConnectionString(conn.Host, conn.Port, conn.DBName, conn.Username, password, conn.SSLMode)
	return connStr, ad.DBType(), nil
}

func dbTypeDriver(dbType string) (string, error) {
	ad, err := adapter.GetAdapter(dbType)
	if err != nil {
		return "", fmt.Errorf("unsupported database driver: %s", dbType)
	}
	return ad.DriverName(), nil
}

func quoteIdentifier(dbType, ident string) string {
	ad, err := adapter.GetAdapter(dbType)
	if err == nil {
		return ad.QuoteIdentifier(ident)
	}
	gen, err := sqldialect.GetGenerator(dbType)
	if err != nil {
		return `"` + ident + `"`
	}
	return gen.QuoteIdentifier(ident)
}

// ExecuteTableCRUD dispatches a request directly to the table CRUD handlers for a given connection
func (h *InstantCRUDHandler) ExecuteTableCRUD(w http.ResponseWriter, r *http.Request, connID, tableName, id string) {
	if h.reader != nil && connID != "" {
		conn, err := h.reader.GetConnectionByID(connID)
		if err == nil && conn != nil && conn.IsReadOnly {
			if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch || r.Method == http.MethodDelete {
				w.Header().Set("Allow", "GET, HEAD, OPTIONS")
				responder.Error(w, r, http.StatusMethodNotAllowed, responder.ErrCodeMethodNotAllowed, "Connection is configured as read-only: mutating operations are prohibited", map[string]string{"Allow": "GET, HEAD, OPTIONS"})
				return
			}
		}
	}

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
		responder.Error(w, r, http.StatusMethodNotAllowed, responder.ErrCodeMethodNotAllowed, "Method not allowed for instant CRUD")
	}
}
