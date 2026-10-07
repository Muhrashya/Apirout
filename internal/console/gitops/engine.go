package gitops

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/apirout/apirout/internal/console/publisher"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

// ValidationError represents a structured dependency or validation failure during declarative ingestion
type ValidationError struct {
	Entity     string `json:"entity"`
	Identifier string `json:"identifier"`
	Reason     string `json:"reason"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for %s '%s': %s", e.Entity, e.Identifier, e.Reason)
}

// IngestionResult contains count statistics of successfully imported entities
type IngestionResult struct {
	GroupsCount      int `json:"groups"`
	ConnectionsCount int `json:"connections"`
	UpstreamsCount   int `json:"upstreams"`
	TargetsCount     int `json:"targets"`
	ServicesCount    int `json:"services"`
	ConsumersCount   int `json:"consumers"`
	QueriesCount     int `json:"queries"`
	EndpointsCount   int `json:"endpoints"`
}

// DeclarativeEngine applies declarative metadata snapshots with DAG dependency ordering and atomic rollback
type DeclarativeEngine struct {
	db        *sqlx.DB
	dbType    string
	publisher *publisher.RedisPublisher
}

// NewDeclarativeEngine creates a new transactional declarative state engine
func NewDeclarativeEngine(db *sqlx.DB, dbType string, pub *publisher.RedisPublisher) *DeclarativeEngine {
	return &DeclarativeEngine{
		db:        db,
		dbType:    dbType,
		publisher: pub,
	}
}

// Ingest executes the complete topological DAG ingestion inside a single atomic SQL transaction
func (e *DeclarativeEngine) Ingest(ctx context.Context, snapshot *MetadataSnapshot) (*IngestionResult, error) {
	if snapshot == nil {
		return nil, &ValidationError{Entity: "Snapshot", Identifier: "root", Reason: "snapshot cannot be nil"}
	}
	if e.db == nil {
		return nil, fmt.Errorf("engine database handle is not initialized")
	}

	tx, err := e.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		// Rollback is a no-op if transaction was already committed
		_ = tx.Rollback()
	}()

	now := time.Now().UTC()
	result := &IngestionResult{}

	// Caches to track resolved entities by their unique declarative names
	groupsByName := make(map[string]string)
	consumerGroupsByName := make(map[string]string)
	connectionsByName := make(map[string]string)
	upstreamsByName := make(map[string]string)
	servicesByName := make(map[string]string)
	queriesByName := make(map[string]string)

	// Pre-load existing groups, connections, upstreams, services, queries from DB within tx to support incremental snapshots
	if err := e.preloadExistingEntities(tx, groupsByName, consumerGroupsByName, connectionsByName, upstreamsByName, servicesByName, queriesByName); err != nil {
		return nil, fmt.Errorf("preload existing entities: %w", err)
	}

	// =========================================================================
	// TIER 1: GROUPS & CONSUMER GROUPS & DATABASE CONNECTIONS
	// =========================================================================

	// 1A. Resource Groups
	for _, g := range snapshot.Groups {
		if strings.TrimSpace(g.Name) == "" {
			return nil, &ValidationError{Entity: "Group", Identifier: "", Reason: "group name cannot be empty"}
		}

		var groupID string
		err := tx.Get(&groupID, tx.Rebind("SELECT id FROM groups WHERE name = ?"), g.Name)
		if err == sql.ErrNoRows {
			groupID = uuid.New().String()
			_, err = tx.Exec(tx.Rebind("INSERT INTO groups (id, name, description, created_at) VALUES (?, ?, ?, ?)"),
				groupID, g.Name, g.Description, now)
			if err != nil {
				return nil, &ValidationError{Entity: "Group", Identifier: g.Name, Reason: err.Error()}
			}
		} else if err != nil {
			return nil, &ValidationError{Entity: "Group", Identifier: g.Name, Reason: err.Error()}
		} else {
			_, err = tx.Exec(tx.Rebind("UPDATE groups SET description = ? WHERE id = ?"), g.Description, groupID)
			if err != nil {
				return nil, &ValidationError{Entity: "Group", Identifier: g.Name, Reason: err.Error()}
			}
		}
		groupsByName[g.Name] = groupID
		result.GroupsCount++

		// Also mirror into consumer_groups if not present
		var cgID string
		err = tx.Get(&cgID, tx.Rebind("SELECT id FROM consumer_groups WHERE name = ?"), g.Name)
		if err == sql.ErrNoRows {
			cgID = uuid.New().String()
			_, _ = tx.Exec(tx.Rebind("INSERT INTO consumer_groups (id, name, description, rate_limit_rpm, rate_limit_burst, created_at, updated_at) VALUES (?, ?, ?, 60, 10, ?, ?)"),
				cgID, g.Name, g.Description, now, now)
			consumerGroupsByName[g.Name] = cgID
		} else if err == nil {
			consumerGroupsByName[g.Name] = cgID
		}
	}

	// 1B. Database Connections
	for _, c := range snapshot.Connections {
		if strings.TrimSpace(c.Name) == "" {
			return nil, &ValidationError{Entity: "Connection", Identifier: "", Reason: "connection name cannot be empty"}
		}
		if strings.TrimSpace(c.Host) == "" {
			return nil, &ValidationError{Entity: "Connection", Identifier: c.Name, Reason: "host cannot be empty"}
		}
		if c.Port <= 0 {
			return nil, &ValidationError{Entity: "Connection", Identifier: c.Name, Reason: "port must be positive"}
		}

		var groupID *string
		if c.GroupName != "" {
			gid, ok := groupsByName[c.GroupName]
			if !ok {
				return nil, &ValidationError{Entity: "Connection", Identifier: c.Name, Reason: fmt.Sprintf("referenced group '%s' does not exist", c.GroupName)}
			}
			groupID = &gid
		}

		dbType := c.DBType
		if dbType == "" {
			dbType = "postgres"
		}
		sslMode := c.SSLMode
		if sslMode == "" {
			sslMode = "disable"
		}

		var connID string
		var existingEncPassword string
		err := tx.QueryRow(tx.Rebind("SELECT id, password_encrypted FROM connections WHERE name = ?"), c.Name).Scan(&connID, &existingEncPassword)
		if err == sql.ErrNoRows {
			connID = uuid.New().String()
			_, err = tx.Exec(tx.Rebind(`
				INSERT INTO connections 
				(id, group_id, name, db_type, host, port, db_name, username, password_encrypted, ssl_mode, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
				connID, groupID, c.Name, dbType, c.Host, c.Port, c.DBName, c.Username, "", sslMode, now, now)
			if err != nil {
				return nil, &ValidationError{Entity: "Connection", Identifier: c.Name, Reason: err.Error()}
			}
		} else if err != nil {
			return nil, &ValidationError{Entity: "Connection", Identifier: c.Name, Reason: err.Error()}
		} else {
			_, err = tx.Exec(tx.Rebind(`
				UPDATE connections SET 
				group_id = ?, db_type = ?, host = ?, port = ?, db_name = ?, username = ?, ssl_mode = ?, updated_at = ?
				WHERE id = ?`),
				groupID, dbType, c.Host, c.Port, c.DBName, c.Username, sslMode, now, connID)
			if err != nil {
				return nil, &ValidationError{Entity: "Connection", Identifier: c.Name, Reason: err.Error()}
			}
		}
		connectionsByName[c.Name] = connID
		result.ConnectionsCount++
	}

	// =========================================================================
	// TIER 2: SERVICES & UPSTREAMS
	// =========================================================================

	// 2A. Upstreams
	for _, u := range snapshot.Upstreams {
		if strings.TrimSpace(u.Name) == "" {
			return nil, &ValidationError{Entity: "Upstream", Identifier: "", Reason: "upstream name cannot be empty"}
		}

		algo := u.Algorithm
		if algo == "" {
			algo = "round-robin"
		}
		slots := u.Slots
		if slots <= 0 {
			slots = 10000
		}
		hcPath := u.HealthCheckPath
		if hcPath == "" {
			hcPath = "/health"
		}
		hcInterval := u.HealthCheckInterval
		if hcInterval <= 0 {
			hcInterval = 10
		}
		hcTimeout := u.HealthCheckTimeout
		if hcTimeout <= 0 {
			hcTimeout = 5
		}

		var upID string
		err := tx.Get(&upID, tx.Rebind("SELECT id FROM upstreams WHERE name = ?"), u.Name)
		if err == sql.ErrNoRows {
			upID = uuid.New().String()
			_, err = tx.Exec(tx.Rebind(`
				INSERT INTO upstreams 
				(id, name, algorithm, slots, enabled, health_check_enabled, health_check_path, health_check_interval, health_check_timeout, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
				upID, u.Name, algo, slots, u.Enabled, u.HealthCheckEnabled, hcPath, hcInterval, hcTimeout, now, now)
			if err != nil {
				return nil, &ValidationError{Entity: "Upstream", Identifier: u.Name, Reason: err.Error()}
			}
		} else if err != nil {
			return nil, &ValidationError{Entity: "Upstream", Identifier: u.Name, Reason: err.Error()}
		} else {
			_, err = tx.Exec(tx.Rebind(`
				UPDATE upstreams SET 
				algorithm = ?, slots = ?, enabled = ?, health_check_enabled = ?, health_check_path = ?, health_check_interval = ?, health_check_timeout = ?, updated_at = ?
				WHERE id = ?`),
				algo, slots, u.Enabled, u.HealthCheckEnabled, hcPath, hcInterval, hcTimeout, now, upID)
			if err != nil {
				return nil, &ValidationError{Entity: "Upstream", Identifier: u.Name, Reason: err.Error()}
			}
		}
		upstreamsByName[u.Name] = upID
		result.UpstreamsCount++
	}

	// 2B. Services
	for _, s := range snapshot.Services {
		if strings.TrimSpace(s.Name) == "" {
			return nil, &ValidationError{Entity: "Service", Identifier: "", Reason: "service name cannot be empty"}
		}

		var upID *string
		if s.UpstreamName != "" {
			uid, ok := upstreamsByName[s.UpstreamName]
			if !ok {
				return nil, &ValidationError{Entity: "Service", Identifier: s.Name, Reason: fmt.Sprintf("referenced upstream '%s' does not exist", s.UpstreamName)}
			}
			upID = &uid
		}

		var connID *string
		if s.ConnectionName != "" {
			cid, ok := connectionsByName[s.ConnectionName]
			if !ok {
				return nil, &ValidationError{Entity: "Service", Identifier: s.Name, Reason: fmt.Sprintf("referenced connection '%s' does not exist", s.ConnectionName)}
			}
			connID = &cid
		}

		proto := s.Protocol
		if proto == "" {
			proto = "http"
		}
		cTimeout := s.ConnectTimeout
		if cTimeout <= 0 {
			cTimeout = 60000
		}
		rTimeout := s.ReadTimeout
		if rTimeout <= 0 {
			rTimeout = 60000
		}
		wTimeout := s.WriteTimeout
		if wTimeout <= 0 {
			wTimeout = 60000
		}
		retries := s.Retries
		if retries < 0 {
			retries = 5
		}

		var svcID string
		err := tx.Get(&svcID, tx.Rebind("SELECT id FROM services WHERE name = ?"), s.Name)
		if err == sql.ErrNoRows {
			svcID = uuid.New().String()
			_, err = tx.Exec(tx.Rebind(`
				INSERT INTO services 
				(id, name, description, protocol, host, port, path, connect_timeout, read_timeout, write_timeout, retries, enabled, upstream_id, connection_id, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
				svcID, s.Name, s.Description, proto, s.Host, s.Port, s.Path, cTimeout, rTimeout, wTimeout, retries, s.Enabled, upID, connID, now, now)
			if err != nil {
				return nil, &ValidationError{Entity: "Service", Identifier: s.Name, Reason: err.Error()}
			}
		} else if err != nil {
			return nil, &ValidationError{Entity: "Service", Identifier: s.Name, Reason: err.Error()}
		} else {
			_, err = tx.Exec(tx.Rebind(`
				UPDATE services SET 
				description = ?, protocol = ?, host = ?, port = ?, path = ?, connect_timeout = ?, read_timeout = ?, write_timeout = ?, retries = ?, enabled = ?, upstream_id = ?, connection_id = ?, updated_at = ?
				WHERE id = ?`),
				s.Description, proto, s.Host, s.Port, s.Path, cTimeout, rTimeout, wTimeout, retries, s.Enabled, upID, connID, now, svcID)
			if err != nil {
				return nil, &ValidationError{Entity: "Service", Identifier: s.Name, Reason: err.Error()}
			}
		}
		servicesByName[s.Name] = svcID
		result.ServicesCount++
	}

	// =========================================================================
	// TIER 3: TARGETS & CONSUMERS
	// =========================================================================

	// 3A. Targets
	for _, u := range snapshot.Upstreams {
		upID := upstreamsByName[u.Name]
		for _, t := range u.Targets {
			host := t.Host
			port := t.Port
			targetStr := t.Target
			if targetStr == "" && host != "" {
				targetStr = fmt.Sprintf("%s:%d", host, port)
			}
			if targetStr == "" {
				return nil, &ValidationError{Entity: "Target", Identifier: u.Name, Reason: "target cannot be empty"}
			}
			if host == "" {
				parts := strings.Split(targetStr, ":")
				host = parts[0]
			}
			if port <= 0 {
				port = 80
			}
			weight := t.Weight
			if weight <= 0 {
				weight = 100
			}

			var tgtID string
			err := tx.Get(&tgtID, tx.Rebind("SELECT id FROM targets WHERE upstream_id = ? AND target = ?"), upID, targetStr)
			if err == sql.ErrNoRows {
				tgtID = uuid.New().String()
				_, err = tx.Exec(tx.Rebind(`
					INSERT INTO targets 
					(id, upstream_id, target, host, port, weight, enabled, health_status, failover, created_at, updated_at)
					VALUES (?, ?, ?, ?, ?, ?, ?, 'healthy', false, ?, ?)`),
					tgtID, upID, targetStr, host, port, weight, t.Enabled, now, now)
				if err != nil {
					return nil, &ValidationError{Entity: "Target", Identifier: targetStr, Reason: err.Error()}
				}
			} else if err != nil {
				return nil, &ValidationError{Entity: "Target", Identifier: targetStr, Reason: err.Error()}
			} else {
				_, err = tx.Exec(tx.Rebind(`
					UPDATE targets SET host = ?, port = ?, weight = ?, enabled = ?, updated_at = ?
					WHERE id = ?`),
					host, port, weight, t.Enabled, now, tgtID)
				if err != nil {
					return nil, &ValidationError{Entity: "Target", Identifier: targetStr, Reason: err.Error()}
				}
			}
			result.TargetsCount++
		}
	}

	// 3B. Consumers
	for _, c := range snapshot.Consumers {
		if strings.TrimSpace(c.Username) == "" {
			return nil, &ValidationError{Entity: "Consumer", Identifier: "", Reason: "username cannot be empty"}
		}

		var groupID *string
		if c.GroupName != "" {
			gid, ok := consumerGroupsByName[c.GroupName]
			if !ok {
				gid, ok = groupsByName[c.GroupName]
			}
			if !ok {
				return nil, &ValidationError{Entity: "Consumer", Identifier: c.Username, Reason: fmt.Sprintf("referenced consumer group '%s' does not exist", c.GroupName)}
			}
			groupID = &gid
		}

		status := c.Status
		if status == "" {
			status = "active"
		}

		var consID string
		err := tx.Get(&consID, tx.Rebind("SELECT id FROM api_clients WHERE username = ?"), c.Username)
		if err == sql.ErrNoRows {
			consID = uuid.New().String()
			defaultHash, _ := bcrypt.GenerateFromPassword([]byte("default-secret-token"), bcrypt.DefaultCost)
			_, err = tx.Exec(tx.Rebind(`
				INSERT INTO api_clients 
				(id, group_id, username, custom_id, info, status, password_hash, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`),
				consID, groupID, c.Username, c.CustomID, c.Description, status, string(defaultHash), now, now)
			if err != nil {
				return nil, &ValidationError{Entity: "Consumer", Identifier: c.Username, Reason: err.Error()}
			}
		} else if err != nil {
			return nil, &ValidationError{Entity: "Consumer", Identifier: c.Username, Reason: err.Error()}
		} else {
			_, err = tx.Exec(tx.Rebind(`
				UPDATE api_clients SET group_id = ?, custom_id = ?, info = ?, status = ?, updated_at = ?
				WHERE id = ?`),
				groupID, c.CustomID, c.Description, status, now, consID)
			if err != nil {
				return nil, &ValidationError{Entity: "Consumer", Identifier: c.Username, Reason: err.Error()}
			}
		}
		result.ConsumersCount++
	}

	// =========================================================================
	// TIER 4: STANDALONE QUERIES
	// =========================================================================

	for _, q := range snapshot.Queries {
		if strings.TrimSpace(q.Name) == "" {
			return nil, &ValidationError{Entity: "Query", Identifier: "", Reason: "query name cannot be empty"}
		}
		if strings.TrimSpace(q.SQLText) == "" {
			return nil, &ValidationError{Entity: "Query", Identifier: q.Name, Reason: "sql_text cannot be empty"}
		}

		var connID *string
		if q.Connection != "" {
			cid, ok := connectionsByName[q.Connection]
			if !ok {
				return nil, &ValidationError{Entity: "Query", Identifier: q.Name, Reason: fmt.Sprintf("referenced connection '%s' does not exist", q.Connection)}
			}
			connID = &cid
		}

		var groupID *string
		if q.GroupName != "" {
			gid, ok := groupsByName[q.GroupName]
			if !ok {
				return nil, &ValidationError{Entity: "Query", Identifier: q.Name, Reason: fmt.Sprintf("referenced group '%s' does not exist", q.GroupName)}
			}
			groupID = &gid
		}

		defLimit := q.DefaultLimit
		if defLimit <= 0 {
			defLimit = 100
		}

		var queryID string
		err := tx.Get(&queryID, tx.Rebind("SELECT id FROM queries WHERE name = ?"), q.Name)
		if err == sql.ErrNoRows {
			queryID = uuid.New().String()
			_, err = tx.Exec(tx.Rebind(`
				INSERT INTO queries 
				(id, group_id, connection_id, name, sql_text, order_by, count_sql_text, params_schema, created_at, updated_at, preview_limit, preview_offset, default_limit)
				VALUES (?, ?, ?, ?, ?, ?, ?, '{}', ?, ?, 200, 0, ?)`),
				queryID, groupID, connID, q.Name, q.SQLText, q.OrderBy, q.CountSQLText, now, now, defLimit)
			if err != nil {
				return nil, &ValidationError{Entity: "Query", Identifier: q.Name, Reason: err.Error()}
			}
		} else if err != nil {
			return nil, &ValidationError{Entity: "Query", Identifier: q.Name, Reason: err.Error()}
		} else {
			_, err = tx.Exec(tx.Rebind(`
				UPDATE queries SET 
				group_id = ?, connection_id = ?, sql_text = ?, order_by = ?, count_sql_text = ?, default_limit = ?, updated_at = ?
				WHERE id = ?`),
				groupID, connID, q.SQLText, q.OrderBy, q.CountSQLText, defLimit, now, queryID)
			if err != nil {
				return nil, &ValidationError{Entity: "Query", Identifier: q.Name, Reason: err.Error()}
			}
		}
		queriesByName[q.Name] = queryID
		result.QueriesCount++
	}

	// =========================================================================
	// TIER 5: ENDPOINTS & ROUTES
	// =========================================================================

	for _, ep := range snapshot.Endpoints {
		if strings.TrimSpace(ep.Path) == "" {
			return nil, &ValidationError{Entity: "Endpoint", Identifier: "", Reason: "endpoint path cannot be empty"}
		}

		method := strings.ToUpper(strings.TrimSpace(ep.Method))
		if method == "" {
			method = "GET"
		}

		var svcID *string
		if ep.ServiceName != "" {
			sid, ok := servicesByName[ep.ServiceName]
			if !ok {
				return nil, &ValidationError{Entity: "Endpoint", Identifier: ep.Path, Reason: fmt.Sprintf("referenced service '%s' does not exist", ep.ServiceName)}
			}
			svcID = &sid
		}

		var qID *string
		if ep.QueryName != "" {
			qid, ok := queriesByName[ep.QueryName]
			if !ok {
				return nil, &ValidationError{Entity: "Endpoint", Identifier: ep.Path, Reason: fmt.Sprintf("referenced query '%s' does not exist", ep.QueryName)}
			}
			qID = &qid
		}

		var groupID *string
		if ep.GroupName != "" {
			gid, ok := groupsByName[ep.GroupName]
			if !ok {
				return nil, &ValidationError{Entity: "Endpoint", Identifier: ep.Path, Reason: fmt.Sprintf("referenced group '%s' does not exist", ep.GroupName)}
			}
			groupID = &gid
		}

		name := ep.Name
		if name == "" {
			name = ep.Path
		}
		proto := ep.Protocols
		if proto == "" {
			proto = "http,https"
		}
		defLimit := ep.DefaultLimit
		if defLimit <= 0 {
			defLimit = 100
		}
		maxLimit := ep.MaxLimit
		if maxLimit <= 0 {
			maxLimit = 5000
		}
		rpm := ep.RateLimitRPM
		if rpm <= 0 {
			rpm = 60
		}
		burst := ep.RateLimitBurst
		if burst <= 0 {
			burst = 10
		}

		var endpointID string
		err := tx.Get(&endpointID, tx.Rebind("SELECT id FROM endpoints WHERE path = ? AND method = ?"), ep.Path, method)
		if err == sql.ErrNoRows {
			// Check if exists by path alone
			err = tx.Get(&endpointID, tx.Rebind("SELECT id FROM endpoints WHERE path = ?"), ep.Path)
		}

		if err == sql.ErrNoRows {
			endpointID = uuid.New().String()
			_, err = tx.Exec(tx.Rebind(`
				INSERT INTO endpoints 
				(id, service_id, name, group_id, method, path, protocols, strip_path, preserve_host, query_id, enabled, is_public, rate_limit_rpm, rate_limit_burst, default_limit, max_limit, return_total_count, allow_dynamic_filters, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
				endpointID, svcID, name, groupID, method, ep.Path, proto, ep.StripPath, ep.PreserveHost, qID, ep.Enabled, ep.IsPublic, rpm, burst, defLimit, maxLimit, ep.ReturnTotalCount, ep.AllowDynamicFilters, now, now)
			if err != nil {
				return nil, &ValidationError{Entity: "Endpoint", Identifier: ep.Path, Reason: err.Error()}
			}
		} else if err != nil {
			return nil, &ValidationError{Entity: "Endpoint", Identifier: ep.Path, Reason: err.Error()}
		} else {
			_, err = tx.Exec(tx.Rebind(`
				UPDATE endpoints SET 
				service_id = ?, name = ?, group_id = ?, method = ?, protocols = ?, strip_path = ?, preserve_host = ?, query_id = ?, enabled = ?, is_public = ?, rate_limit_rpm = ?, rate_limit_burst = ?, default_limit = ?, max_limit = ?, return_total_count = ?, allow_dynamic_filters = ?, updated_at = ?
				WHERE id = ?`),
				svcID, name, groupID, method, proto, ep.StripPath, ep.PreserveHost, qID, ep.Enabled, ep.IsPublic, rpm, burst, defLimit, maxLimit, ep.ReturnTotalCount, ep.AllowDynamicFilters, now, endpointID)
			if err != nil {
				return nil, &ValidationError{Entity: "Endpoint", Identifier: ep.Path, Reason: err.Error()}
			}
		}
		result.EndpointsCount++
	}

	// =========================================================================
	// COMMIT TRANSACTION
	// =========================================================================

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	// =========================================================================
	// EMIT BULK CONFIG INVALIDATION EVENT
	// =========================================================================

	if e.publisher != nil {
		_ = e.publisher.PublishBulkConfigUpdated()
	}

	return result, nil
}

// preloadExistingEntities loads existing system resources to resolve references in partial snapshots
func (e *DeclarativeEngine) preloadExistingEntities(tx *sqlx.Tx,
	groups map[string]string,
	consumerGroups map[string]string,
	connections map[string]string,
	upstreams map[string]string,
	services map[string]string,
	queries map[string]string) error {

	type row struct {
		ID   string `db:"id"`
		Name string `db:"name"`
	}

	var gRows []row
	if err := tx.Select(&gRows, "SELECT id, name FROM groups"); err == nil {
		for _, r := range gRows {
			groups[r.Name] = r.ID
		}
	}

	var cgRows []row
	if err := tx.Select(&cgRows, "SELECT id, name FROM consumer_groups"); err == nil {
		for _, r := range cgRows {
			consumerGroups[r.Name] = r.ID
		}
	}

	var cRows []row
	if err := tx.Select(&cRows, "SELECT id, name FROM connections"); err == nil {
		for _, r := range cRows {
			connections[r.Name] = r.ID
		}
	}

	var uRows []row
	if err := tx.Select(&uRows, "SELECT id, name FROM upstreams"); err == nil {
		for _, r := range uRows {
			upstreams[r.Name] = r.ID
		}
	}

	var sRows []row
	if err := tx.Select(&sRows, "SELECT id, name FROM services"); err == nil {
		for _, r := range sRows {
			services[r.Name] = r.ID
		}
	}

	var qRows []row
	if err := tx.Select(&qRows, "SELECT id, name FROM queries"); err == nil {
		for _, r := range qRows {
			queries[r.Name] = r.ID
		}
	}

	return nil
}
