package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/apirout/apirout/internal/console/cache"
	"github.com/apirout/apirout/internal/console/registry"
	"github.com/apirout/apirout/internal/core/crypto"
	"github.com/apirout/apirout/internal/core/targets"
	"github.com/apirout/apirout/internal/core/tunnel"
)

const (
	PreviewLimit      = 200
	APILimit          = 5000
	QueryTimeout      = 15 * time.Second
	TotalCountTimeout = 30 * time.Second
)

// Service handles query execution with caching
type Service struct {
	registry       *registry.Registry
	encryptionKey  []byte
	endpointCache  *cache.EndpointCache
	connectionPool *cache.ConnectionPool
	countCache     *cache.CountCache
}

// NewService creates a new executor service with caching
func NewService(reg *registry.Registry, encryptionKey string) *Service {
	// Pad or truncate key to 32 bytes for AES-256
	key := make([]byte, 32)
	copy(key, []byte(encryptionKey))

	// Create endpoint cache with 5 minute TTL
	epCache := cache.NewEndpointCache(5 * time.Minute)
	epCache.StartCleanupRoutine(1 * time.Minute)

	// Create connection pool with 30 minute idle timeout
	connPool := cache.NewConnectionPool(30*time.Minute, 50)

	// Create count cache with 60 second TTL
	countCache := cache.NewCountCache(60 * time.Second)
	countCache.StartCleanupRoutine(5 * time.Minute)

	return &Service{
		registry:       reg,
		encryptionKey:  key,
		endpointCache:  epCache,
		connectionPool: connPool,
		countCache:     countCache,
	}
}

// GetEndpointCache returns the endpoint cache for invalidation
func (s *Service) GetEndpointCache() *cache.EndpointCache {
	return s.endpointCache
}

// GetConnectionPool returns the connection pool for invalidation
func (s *Service) GetConnectionPool() *cache.ConnectionPool {
	return s.connectionPool
}

// decryptPassword decrypts an encrypted password
func (s *Service) decryptPassword(encrypted string) (string, error) {
	return crypto.DecryptPassword(s.encryptionKey, encrypted)
}

// EncryptPassword encrypts a password
func (s *Service) EncryptPassword(password string) (string, error) {
	return crypto.EncryptPassword(s.encryptionKey, password)
}

// PreviewQuery executes a query with preview limits
func (s *Service) PreviewQuery(connectionID, sql string, limit, offset int) ([]targets.Row, error) {
	// Validate SQL
	if err := ValidateSQL(sql); err != nil {
		return nil, err
	}

	// Get connection
	conn, err := s.registry.GetConnectionByID(connectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}
	if conn == nil {
		return nil, fmt.Errorf("connection not found")
	}

	// Decrypt password
	password, err := s.decryptPassword(conn.PasswordEncrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt password: %w", err)
	}

	// Build connection string
	connString, err := s.buildConnectionString(conn, password)
	if err != nil {
		return nil, err
	}

	// Get connector
	connector, err := targets.GetConnector(conn.DBType)
	if err != nil {
		return nil, err
	}

	// Apply safe limits
	effectiveLimit := limit
	if effectiveLimit <= 0 || effectiveLimit > PreviewLimit {
		effectiveLimit = PreviewLimit
	}

	effectiveOffset := offset
	if effectiveOffset < 0 {
		effectiveOffset = 0
	}

	// Wrap with limit and offset
	wrappedSQL := WrapWithPagination(sql, effectiveLimit, effectiveOffset)

	// Execute with timeout
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()

	return connector.Execute(ctx, connString, wrappedSQL, nil, effectiveLimit)
}

// ExecuteEndpoint executes a saved query for an API endpoint with caching
// ExecuteEndpoint executes a saved query for an API endpoint with caching
func (s *Service) ExecuteEndpoint(endpointID string, params map[string]interface{}) (interface{}, error) {
	// Get endpoint
	endpoint, err := s.registry.GetEndpointByID(endpointID)
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoint: %w", err)
	}
	if endpoint == nil {
		return nil, fmt.Errorf("endpoint not found")
	}

	if endpoint.QueryID == nil {
		return nil, fmt.Errorf("endpoint has no associated query")
	}

	// Get query
	query, err := s.registry.GetQueryByID(*endpoint.QueryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get query: %w", err)
	}
	if query == nil {
		return nil, fmt.Errorf("query not found")
	}

	// Build config
	config := &cache.EndpointConfig{
		EndpointID:        endpoint.ID,
		Method:            endpoint.Method,
		Path:              endpoint.Path,
		QueryID:           endpoint.QueryID,
		QuerySQL:          query.SQLText,
		QueryOrderBy:      safeString(query.OrderBy),
		ParamsSchema:      query.ParamsSchema,
		ConnectionID:      query.ConnectionID,
		Enabled:           endpoint.Enabled,
		IsPublic:          endpoint.IsPublic,
		GroupID:           endpoint.GroupID,
		HasLimit:          HasLimit(query.SQLText),
		DefaultLimit:      endpoint.DefaultLimit,
		MaxLimit:            endpoint.MaxLimit,
		ReturnTotalCount:    endpoint.ReturnTotalCount,
		AllowDynamicFilters: endpoint.AllowDynamicFilters,
		CountSQLText:        safeString(query.CountSQLText),
		QueryDefaultLimit:   query.DefaultLimit,
	}

	return s.executeCachedEndpoint(config, params)
}

// ExecuteEndpointByPath executes endpoint by path with caching
func (s *Service) ExecuteEndpointByPath(path string, params map[string]interface{}) (interface{}, error) {
	// Try to get from cache first
	if cached, found := s.endpointCache.Get(path); found {
		fmt.Printf("Executor: Cache hit for %s\n", path)
		return s.executeCachedEndpoint(cached, params)
	}
	fmt.Printf("Executor: Cache miss for %s\n", path)

	// Get endpoint from database
	endpoint, err := s.registry.GetEndpointByPath(path)
	if err != nil {
		fmt.Printf("Executor: GetEndpointByPath error: %v\n", err)
		return nil, fmt.Errorf("failed to get endpoint: %w", err)
	}
	if endpoint == nil {
		return nil, fmt.Errorf("endpoint not found")
	}

	if endpoint.QueryID == nil {
		return nil, fmt.Errorf("endpoint has no associated query")
	}

	// Get query
	query, err := s.registry.GetQueryByID(*endpoint.QueryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get query: %w", err)
	}
	if query == nil {
		return nil, fmt.Errorf("query not found")
	}

	// Cache the endpoint config
	cachedConfig := &cache.EndpointConfig{
		EndpointID:        endpoint.ID,
		Method:            endpoint.Method,
		Path:              endpoint.Path,
		QueryID:           endpoint.QueryID,
		QuerySQL:          query.SQLText,
		QueryOrderBy:      safeString(query.OrderBy),
		ParamsSchema:      query.ParamsSchema,
		ConnectionID:      query.ConnectionID,
		Enabled:           endpoint.Enabled,
		IsPublic:          endpoint.IsPublic,
		GroupID:           endpoint.GroupID,
		HasLimit:          HasLimit(query.SQLText),
		DefaultLimit:      endpoint.DefaultLimit,
		MaxLimit:          endpoint.MaxLimit,
		ReturnTotalCount:  endpoint.ReturnTotalCount,
		AllowDynamicFilters: endpoint.AllowDynamicFilters,
		CountSQLText:      safeString(query.CountSQLText),
		QueryDefaultLimit: query.DefaultLimit,
	}
	s.endpointCache.Set(path, cachedConfig)

	return s.executeCachedEndpoint(cachedConfig, params)
}

// ExecuteEndpointCount executes only the count query for an endpoint (ignoring ReturnTotalCount flag)
func (s *Service) ExecuteEndpointCount(endpointID string, params map[string]interface{}) (int64, error) {
	// Get endpoint
	endpoint, err := s.registry.GetEndpointByID(endpointID)
	if err != nil {
		return 0, fmt.Errorf("failed to get endpoint: %w", err)
	}
	if endpoint == nil {
		return 0, fmt.Errorf("endpoint not found")
	}

	if endpoint.QueryID == nil {
		return 0, fmt.Errorf("endpoint has no associated query")
	}

	// Get query
	query, err := s.registry.GetQueryByID(*endpoint.QueryID)
	if err != nil {
		return 0, fmt.Errorf("failed to get query: %w", err)
	}
	if query == nil {
		return 0, fmt.Errorf("query not found")
	}

	// 1. Check Cache
	cacheKey := s.countCache.GenerateKey(endpointID, params)
	if cachedTotal, found := s.countCache.Get(cacheKey); found {
		// Cache Hit
		return cachedTotal, nil
	}

	// We need to build a config similar to executeCachedEndpoint but specific for counting.
	// We can reuse the logic but force ReturnTotalCount=true for our purpose here,
	// except executeCachedEndpoint does both data + count. We want ONLY count.
	
	// Create temporary config
	config := &cache.EndpointConfig{
		EndpointID:          endpoint.ID,
		Method:              endpoint.Method,
		Path:                endpoint.Path,
		QueryID:             endpoint.QueryID,
		QuerySQL:            query.SQLText,
		ConnectionID:        query.ConnectionID,
		Enabled:             endpoint.Enabled,
		IsPublic:            endpoint.IsPublic,
		AllowDynamicFilters: endpoint.AllowDynamicFilters,
		CountSQLText:        safeString(query.CountSQLText),
	}

	// Execute
	total, err := s.executeCountOnly(config, params)
	if err != nil {
		return 0, err
	}

	// 2. Update Cache
	s.countCache.Set(cacheKey, total)

	return total, nil
}

// executeCountOnly executes only the count query
func (s *Service) executeCountOnly(config *cache.EndpointConfig, params map[string]interface{}) (int64, error) {
	if config.ConnectionID == nil {
		return 0, fmt.Errorf("query has no associated connection")
	}

	if params == nil {
		params = make(map[string]interface{})
	}

	// Handle Dynamic Filters (needed for accurate count)
	var dynamicWhere []string
	if config.AllowDynamicFilters {
		// Build whitelist from ParamsSchema (stored as lowercase -> original name)
		var validParams map[string]string
		if config.ParamsSchema != "" {
			validParams = make(map[string]string)
			var schema []struct {
				Name string `json:"name"`
			}
			if err := json.Unmarshal([]byte(config.ParamsSchema), &schema); err == nil {
				for _, p := range schema {
					validParams[strings.ToLower(p.Name)] = p.Name
				}
			}
		}

		for key := range params {
			if isReservedParam(key) {
				continue
			}
			if !isValidColumnName(key) {
				continue
			}

			// Determine column name to use
			colName := key
			// Validate against ParamsSchema if available
			if validParams != nil {
				// Check lowercase key against whitelist
				if originalName, ok := validParams[strings.ToLower(key)]; ok {
					colName = originalName
				} else {
					// Not in whitelist, skip
					continue
				}
			}

			// Use parameter binding for value ":param_name"
			// We use `colName` for the column identifier (from schema if available, ensuring correct case)
			// We use `key` for the parameter placeholder because that's what's in the `params` map
			dynamicWhere = append(dynamicWhere, fmt.Sprintf(`"%s" = :%s`, colName, key))
		}
	}

	// Get connection
	conn, err := s.registry.GetConnectionByID(*config.ConnectionID)
	if err != nil {
		return 0, fmt.Errorf("failed to get connection: %w", err)
	}
	if conn == nil {
		return 0, fmt.Errorf("connection not found")
	}

	// Decrypt password
	password, err := s.decryptPassword(conn.PasswordEncrypted)
	if err != nil {
		return 0, fmt.Errorf("failed to decrypt password: %w", err)
	}

	// Build connection string
	connString, err := s.buildConnectionString(conn, password)
	if err != nil {
		return 0, err
	}

	// Get connector
	connector, err := targets.GetConnector(conn.DBType)
	if err != nil {
		return 0, err
	}

	// Construct filter clause
	filterClause := ""
	if len(dynamicWhere) > 0 {
		filterClause = "WHERE " + strings.Join(dynamicWhere, " AND ")
	}

	// Build Count SQL
	var countSQL string
	if config.CountSQLText != "" {
		// Use helper to append dynamic filters to custom SQL
		countSQL = PrepareCustomCountSQL(config.CountSQLText, dynamicWhere)
	} else {
		// Strip LIMIT/OFFSET from inner query for accurate count
		innerSQL := RemoveLimitOffset(config.QuerySQL)
		
		// Wrap with count
		if err := ValidateSQL(innerSQL); err != nil {
			return 0, fmt.Errorf("invalid inner SQL: %w", err)
		}
		countSQL = fmt.Sprintf("SELECT COUNT(*) FROM (%s) AS t_count %s", innerSQL, filterClause)
	}

	// Validate
	if err := ValidateSQL(countSQL); err != nil {
		return 0, err
	}

	// Execute with INCREASED timeout for count
	ctx, cancel := context.WithTimeout(context.Background(), TotalCountTimeout)
	defer cancel()

	countRes, err := connector.Execute(ctx, connString, countSQL, params, 1)
	if err != nil {
		return 0, err
	}

	var total int64
	if len(countRes) > 0 {
		for _, col := range countRes[0] {
			switch n := col.Value.(type) {
			case int64:
				total = n
			case int:
				total = int64(n)
			case float64:
				total = int64(n)
			case string:
				if val, err := strconv.ParseInt(n, 10, 64); err == nil {
					total = val
				}
			}
			break
		}
	}

	return total, nil
}

// executeCachedEndpoint executes using cached endpoint config
func (s *Service) executeCachedEndpoint(config *cache.EndpointConfig, params map[string]interface{}) (interface{}, error) {
	if config.ConnectionID == nil {
		return nil, fmt.Errorf("query has no associated connection")
	}

	if params == nil {
		params = make(map[string]interface{})
	}

	// 1. Handle Pagination Params
	// Check for page/limit in params (from query string)
	var page = 1
	var limit = config.QueryDefaultLimit
	if limit <= 0 {
		limit = config.DefaultLimit
	}
	if limit <= 0 {
		limit = 100 // Fallback
	}

	// Parse page
	if p, ok := params["page"]; ok {
		// p might be string or int or float64 depending on JSON decoder?
		// From URL query params, it's string. From JSON body, might be number.
		switch v := p.(type) {
		case string:
			if val, err := strconv.Atoi(v); err == nil && val > 0 {
				page = val
			}
		case float64:
			if val := int(v); val > 0 {
				page = val
			}
		case int:
			if v > 0 {
				page = v
			}
		}
	}
	// Ensure params has int value for page
	params["page"] = page

	// Parse limit
	if l, ok := params["limit"]; ok {
		switch v := l.(type) {
		case string:
			if val, err := strconv.Atoi(v); err == nil && val > 0 {
				limit = val
			}
		case float64:
			if val := int(v); val > 0 {
				limit = val
			}
		case int:
			if v > 0 {
				limit = v
			}
		}
	}
	// Ensure params has int value for limit
	params["limit"] = limit

	// Clamp limit
	if config.MaxLimit > 0 && limit > config.MaxLimit {
		limit = config.MaxLimit
	}

	offset := (page - 1) * limit

	// Inject reserved params
	params["__limit"] = limit
	params["__offset"] = offset

	// 2. Handle Dynamic Filters (if enabled)
	var dynamicWhere []string
	if config.AllowDynamicFilters {
		// Build whitelist from ParamsSchema (stored as lowercase -> original name)
		var validParams map[string]string
		if config.ParamsSchema != "" {
			validParams = make(map[string]string)
			var schema []struct {
				Name string `json:"name"`
			}
			if err := json.Unmarshal([]byte(config.ParamsSchema), &schema); err == nil {
				for _, p := range schema {
					validParams[strings.ToLower(p.Name)] = p.Name
				}
			}
		}

		for key := range params {
			// Skip reserved keys
			if isReservedParam(key) {
				continue
			}

			// Validate column name (alphanumeric + underscore only) for SQL safety
			if !isValidColumnName(key) {
				continue
			}

			// Determine column name to use
			colName := key
			// Validate against ParamsSchema if available
			if validParams != nil {
				// Check lowercase key against whitelist
				if originalName, ok := validParams[strings.ToLower(key)]; ok {
					colName = originalName
				} else {
					// Not in whitelist, skip
					continue
				}
			}

			// Add to where clause
			// Important: Use parameter binding for value ":param_name"
			// Handle quoting based on DB type (later step or generalized)
			// For now, use standard double quotes for identifiers as it works in Postgres/SQLite/MySQL(ANSI)
			// TODO: Detect DB type for specific quoting if needed.
			dynamicWhere = append(dynamicWhere, fmt.Sprintf(`"%s" = :%s`, colName, key))
			
			// Ensure param value is available for binding (it already is in params map)
			// Just verify format? No, sqlx handles parsing.
		}
	}

	// End step 1 & 2

	// Get connection
	conn, err := s.registry.GetConnectionByID(*config.ConnectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}
	if conn == nil {
		return nil, fmt.Errorf("connection not found")
	}

	// Validate SQL
	if err := ValidateSQL(config.QuerySQL); err != nil {
		return nil, err
	}

	// Decrypt password
	password, err := s.decryptPassword(conn.PasswordEncrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt password: %w", err)
	}

	// Build connection string
	connString, err := s.buildConnectionString(conn, password)
	if err != nil {
		return nil, err
	}

	// Get connector
	connector, err := targets.GetConnector(conn.DBType)
	if err != nil {
		return nil, err
	}

	// Construct filter clause
	filterClause := ""
	if len(dynamicWhere) > 0 {
		filterClause = "WHERE " + strings.Join(dynamicWhere, " AND ")
	}

// Construct ORDER BY clause — validate before embedding into SQL.
	orderByClause := ""
	if config.QueryOrderBy != "" {
		if isValidOrderByClause(config.QueryOrderBy) {
			orderByClause = "ORDER BY " + config.QueryOrderBy
		} else {
			fmt.Printf("[WARN] Rejected unsafe ORDER BY clause from query config: %q\n", config.QueryOrderBy)
		}
	}

	// Use pre-computed HasLimit flag
	var wrappedSQL string
	
	// Clean the SQL logic for checking and appending
	cleanSQL := strings.TrimRight(strings.TrimSpace(config.QuerySQL), ";")
	
	if config.HasLimit {
		// If original query has limit, we can't easily pagination wrap usually, 
		// but dynamic filter also complicates this. 
		// For now, if dynamic filters exist, we attempt to wrap anyway but user beware.
		wrappedSQL = config.QuerySQL
	} else if len(dynamicWhere) == 0 && !HasOrderBy(cleanSQL) && IsSafeForAppending(cleanSQL) {
		// OPTIMIZATION: If no dynamic filters, no existing ORDER BY, and safe SQL:
		// Append clauses directly to avoid subquery wrapping.
		// This preserves index usage for ORDER BY clauses.
		wrappedSQL = fmt.Sprintf("%s %s LIMIT :__limit OFFSET :__offset", cleanSQL, orderByClause)
	} else {
		// Wrap with pagination, optional filters, and optional ORDER BY
		if orderByClause != "" && !isValidOrderByClause(strings.TrimPrefix(orderByClause, "ORDER BY ")) {
			return nil, fmt.Errorf("invalid ORDER BY clause")
		}
		wrappedSQL = fmt.Sprintf("SELECT * FROM (%s) AS t_paged %s %s LIMIT :__limit OFFSET :__offset", config.QuerySQL, filterClause, orderByClause)
	}

	// Execute with timeout
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()

	// DEBUG: Log the constructed SQL for troubleshooting
	fmt.Printf("[DEBUG] Executing SQL: %s\n", wrappedSQL)
	fmt.Printf("[DEBUG] With params: %+v\n", params)

	rows, err := connector.Execute(ctx, connString, wrappedSQL, params, APILimit*10)
	if err != nil {
		fmt.Printf("[ERROR] SQL execution failed: %v\n", err)
		return nil, err
	}

	// Execute Count Query if needed
	var total int64
	var totalPtr *int64
	if config.ReturnTotalCount {
		// Use explicit CountSQLText if available, otherwise fallback to wrapping
		var countSQL string
		if config.CountSQLText != "" {
			countSQL = config.CountSQLText
		} else {
			if err := ValidateSQL(config.QuerySQL); err != nil {
				return nil, fmt.Errorf("invalid query SQL: %w", err)
			}
			countSQL = fmt.Sprintf("SELECT COUNT(*) FROM (%s) AS t_count %s", config.QuerySQL, filterClause)
		}

		// Validate Count SQL
		if err := ValidateSQL(countSQL); err == nil {
			// Execute count
			countRes, err := connector.Execute(ctx, connString, countSQL, params, 1)
			if err == nil && len(countRes) > 0 {
				// Extract first value
				for _, col := range countRes[0] {
					switch n := col.Value.(type) {
					case int64:
						total = n
					case int:
						total = int64(n)
					case float64:
						total = int64(n)
					case string:
						if val, err := strconv.ParseInt(n, 10, 64); err == nil {
							total = val
						}
					}
					break
				}
				totalPtr = &total
			}
		}
	}

	// Ensure items is never nil - use empty array instead
	if rows == nil {
		rows = []targets.Row{}
	}

	return &PagedResponse{
		Total: totalPtr,
		Limit: limit,
		Page:  page,
		Items: rows,
	}, nil
}

// PagedResponse represents a paginated API response with fixed field order
type PagedResponse struct {
	Total *int64        `json:"total,omitempty"`
	Limit int           `json:"limit"`
	Page  int           `json:"page"`
	Items []targets.Row `json:"items"`
}

func (s *Service) buildConnectionString(conn *registry.Connection, password string) (string, error) {
	host := conn.Host
	port := conn.Port

	// Handle SSH tunnel if enabled
	if conn.SSHEnabled {
		sshPassword := ""
		sshPrivateKey := ""

		if conn.SSHPasswordEncrypted != nil {
			var err error
			sshPassword, err = s.decryptPassword(*conn.SSHPasswordEncrypted)
			if err != nil {
				return "", fmt.Errorf("failed to decrypt SSH password: %w", err)
			}
		}

		if conn.SSHPrivateKeyEncrypted != nil {
			var err error
			sshPrivateKey, err = s.decryptPassword(*conn.SSHPrivateKeyEncrypted)
			if err != nil {
				return "", fmt.Errorf("failed to decrypt SSH private key: %w", err)
			}
		}

		sshHost := ""
		if conn.SSHHost != nil {
			sshHost = *conn.SSHHost
		}
		sshUser := ""
		if conn.SSHUser != nil {
			sshUser = *conn.SSHUser
		}

		tunnelCfg := tunnel.Config{
			SSHHost:    sshHost,
			SSHPort:    conn.SSHPort,
			SSHUser:    sshUser,
			Password:   sshPassword,
			PrivateKey: sshPrivateKey,
			RemoteHost: conn.Host,
			RemotePort: conn.Port,
		}

		sshTunnel, err := tunnel.NewSSHTunnel(tunnelCfg)
		if err != nil {
			return "", fmt.Errorf("failed to create SSH tunnel: %w", err)
		}

		localPort, err := sshTunnel.Start()
		if err != nil {
			return "", fmt.Errorf("failed to start SSH tunnel: %w", err)
		}

		// Use localhost with the tunnel's local port
		host = "127.0.0.1"
		port = localPort
	}

	return targets.BuildConnectionString(conn.DBType, host, port, conn.DBName, conn.Username, password, conn.SSLMode)
}

// GetRegistry returns the registry instance
func (s *Service) GetRegistry() *registry.Registry {
	return s.registry
}

// HasLimit checks if SQL already contains a LIMIT clause
func HasLimit(sql string) bool {
	upperSQL := strings.ToUpper(sql)
	return strings.Contains(upperSQL, " LIMIT ")
}

// HasOrderBy checks if SQL already contains an ORDER BY clause
func HasOrderBy(sql string) bool {
	upperSQL := strings.ToUpper(sql)
	// Simple check for now. More robust parsing would require a SQL parser.
	return strings.Contains(upperSQL, " ORDER BY ")
}

// IsSafeForAppending checks if SQL string is safe for appending clauses
// Returns false if there are trailing comments or other structures that make appending unsafe.
func IsSafeForAppending(sql string) bool {
	trimmed := strings.TrimSpace(sql)
	// Check for single line comments (rough check)
	if strings.Contains(trimmed, "--") {
		return false
	}
	// Check for multi-line comments
	if strings.Contains(trimmed, "/*") {
		return false
	}
	// Check for UNION, INTERSECT, EXCEPT (appending ORDER BY/LIMIT applies to whole result usually, 
	// but might be tricky without parentheses in some dialects. Let's be conservative.)
	upper := strings.ToUpper(trimmed)
	if strings.Contains(upper, " UNION ") || strings.Contains(upper, " INTERSECT ") || strings.Contains(upper, " EXCEPT ") {
		return false
	}
	
	return true
}

func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// Reserved params that are not columns
func isReservedParam(key string) bool {
	// Check lower case
	lowerKey := strings.ToLower(key)
	reserved := map[string]bool{
		"page": true, "limit": true, "sort": true, "order": true,
		"token": true, "access_token": true, "__limit": true, "__offset": true,
	}
	return reserved[lowerKey]
}

// Validation for column names to prevent SQL injection on identifiers
// Allow alphanumeric and underscore only
func isValidColumnName(name string) bool {
	if len(name) == 0 || len(name) > 64 {
		return false
	}
	for _, r := range name {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' {
			return false
		}
	}
	return true
}

// isValidOrderByClause validates that an ORDER BY string only contains safe tokens:
// valid column identifiers (optionally table-prefixed with a dot), and the
// keywords ASC, DESC, NULLS, FIRST, LAST. Commas and spaces are allowed as
// separators. This prevents injection via admin-configured order_by fields.
func isValidOrderByClause(orderBy string) bool {
	if orderBy == "" {
		return true
	}
	allowedKeywords := map[string]bool{
		"ASC": true, "DESC": true, "NULLS": true, "FIRST": true, "LAST": true,
	}
	terms := strings.Split(orderBy, ",")
	for _, term := range terms {
		parts := strings.Fields(strings.TrimSpace(term))
		if len(parts) == 0 {
			continue
		}
		// First token: column name, optionally with table prefix (table.column)
		colParts := strings.SplitN(parts[0], ".", 2)
		for _, cp := range colParts {
			if !isValidColumnName(cp) {
				return false
			}
		}
		// Remaining tokens must be known ORDER BY keywords
		for _, kw := range parts[1:] {
			if !allowedKeywords[strings.ToUpper(kw)] {
				return false
			}
		}
	}
	return true
}

// GetQueryColumns returns the column names for a given endpoint result
func (s *Service) GetQueryColumns(endpointID string) ([]string, error) {
	// Get endpoint
	endpoint, err := s.registry.GetEndpointByID(endpointID)
	if err != nil {
		return nil, err
	}
	if endpoint == nil {
		return nil, fmt.Errorf("endpoint not found")
	}

	if endpoint.QueryID == nil {
		return nil, fmt.Errorf("endpoint has no associated query")
	}

	// Calculate a dummy config just for preview with LIMIT 0
    query, err := s.registry.GetQueryByID(*endpoint.QueryID)
    if err != nil {
        return nil, err
    }
    
    // We can use PreviewQuery logic but we need to resolve connection and params.
    // PreviewQuery takes connectionID and SQL.
    // We just want metadata. LIMIT 1 is robust enough.
    
    if query.ConnectionID == nil {
        return nil, fmt.Errorf("query has no connection")
    }

	// Just reuse PreviewQuery. 
    // Note: dynamic filters might change columns if the query uses "SELECT *".
    // But we just want a "sample" of columns.
	rows, err := s.PreviewQuery(*query.ConnectionID, query.SQLText, 1, 0)
	if err != nil {
		return nil, err // Maybe return empty list instead of error? No, error is better for now.
	}

	if len(rows) == 0 {
		return []string{}, nil
	}

	// Extract columns from the first row
	columns := make([]string, len(rows[0]))
	for i, col := range rows[0] {
		columns[i] = col.Name
	}

	return columns, nil
}
