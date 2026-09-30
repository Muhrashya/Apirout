package registry

import (
	"fmt"
)

// isValidIdentifier validates that a name is a safe SQL identifier
// (alphanumeric + underscore only, max 64 chars). Used to guard DDL statements
// that cannot use parameterized queries (e.g. PRAGMA, ALTER TABLE).
func isValidIdentifier(name string) bool {
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

// MigrateSchema checks for missing columns and adds them
func (r *Registry) MigrateSchema() error {
	// Define types based on DB
	// Define types based on DB
	boolType := "INTEGER"
	trueDefault := "1"
	falseDefault := "0"
	if r.dbType == "postgres" || r.dbType == "postgresql" {
		boolType = "BOOLEAN"
		trueDefault = "TRUE"
		falseDefault = "FALSE"
		
		// Fix incorrect column types from previous incomplete migrations
		if err := r.fixPostgresBooleanColumns(); err != nil {
			return err
		}
	}

	// 0. Check endpoints table for description
	if err := r.ensureColumnExists("endpoints", "description", "TEXT"); err != nil {
		return err
	}

	// 1. Check endpoints table for rate_limit columns
	if err := r.ensureColumnExists("endpoints", "rate_limit_enabled", fmt.Sprintf("%s DEFAULT %s", boolType, trueDefault)); err != nil {
		return err
	}
	if err := r.ensureColumnExists("endpoints", "rate_limit_rpm", "INTEGER DEFAULT 60"); err != nil {
		return err
	}
	if err := r.ensureColumnExists("endpoints", "rate_limit_burst", "INTEGER DEFAULT 10"); err != nil {
		return err
	}

	// 2. Check queries table for preview pagination default
	if err := r.ensureColumnExists("queries", "preview_limit", "INTEGER DEFAULT 200"); err != nil {
		return err
	}
	if err := r.ensureColumnExists("queries", "preview_offset", "INTEGER DEFAULT 0"); err != nil {
		return err
	}

	// 3. Check endpoints table for pagination config
	if err := r.ensureColumnExists("endpoints", "default_limit", "INTEGER DEFAULT 100"); err != nil {
		return err
	}
	if err := r.ensureColumnExists("endpoints", "max_limit", "INTEGER DEFAULT 5000"); err != nil {
		return err
	}
	if err := r.ensureColumnExists("endpoints", "return_total_count", fmt.Sprintf("%s DEFAULT %s", boolType, falseDefault)); err != nil {
		return err
	}
	if err := r.ensureColumnExists("endpoints", "allow_dynamic_filters", fmt.Sprintf("%s DEFAULT %s", boolType, falseDefault)); err != nil {
		return err
	}

	// 4. Check queries table for count_sql_text
	if err := r.ensureColumnExists("queries", "count_sql_text", "TEXT"); err != nil {
		return err
	}

	// 5. Check queries table for default_limit
	if err := r.ensureColumnExists("queries", "default_limit", "INTEGER DEFAULT 100"); err != nil {
		return err
	}

	// 6. Check queries table for order_by
	if err := r.ensureColumnExists("queries", "order_by", "TEXT"); err != nil {
		return err
	}

	// 7. Check api_clients table for info
	if err := r.ensureColumnExists("api_clients", "info", "VARCHAR(255)"); err != nil {
		return err
	}

	// 8. Multi-domain routing columns
	if err := r.ensureColumnExists("endpoints", "allowed_domains", "TEXT DEFAULT '*'"); err != nil {
		return err
	}
	if err := r.ensureColumnExists("endpoints", "cors_origins", "TEXT DEFAULT '*'"); err != nil {
		return err
	}

	// 9. Gateway Domains table for multi-domain management
	gwDomainsTable := `CREATE TABLE IF NOT EXISTS gateway_domains (
		id VARCHAR(255) PRIMARY KEY,
		domain_name VARCHAR(255) UNIQUE NOT NULL,
		display_name VARCHAR(255) NOT NULL,
		is_default BOOLEAN DEFAULT FALSE,
		environment VARCHAR(50) DEFAULT 'production',
		default_allowed_ips TEXT DEFAULT '*',
		description TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`
	if r.dbType == "sqlite" {
		gwDomainsTable = `CREATE TABLE IF NOT EXISTS gateway_domains (
			id TEXT PRIMARY KEY,
			domain_name TEXT UNIQUE NOT NULL,
			display_name TEXT NOT NULL,
			is_default INTEGER DEFAULT 0,
			environment TEXT DEFAULT 'production',
			default_allowed_ips TEXT DEFAULT '*',
			description TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`
	}
	_, _ = r.db.Exec(gwDomainsTable)

	// 10. IP restriction / allowlist columns (Kong parity)
	if err := r.ensureColumnExists("endpoints", "allowed_ips", "TEXT DEFAULT '*'"); err != nil {
		return err
	}
	if err := r.ensureColumnExists("gateway_domains", "default_allowed_ips", "TEXT DEFAULT '*'"); err != nil {
		return err
	}

	// 11. Kong Architecture: Upstreams table
	upstreamsTable := `CREATE TABLE IF NOT EXISTS upstreams (
		id VARCHAR(255) PRIMARY KEY,
		name VARCHAR(255) UNIQUE NOT NULL,
		algorithm VARCHAR(50) DEFAULT 'round-robin',
		hash_on VARCHAR(50) DEFAULT 'none',
		hash_fallback VARCHAR(50) DEFAULT 'none',
		hash_on_cookie VARCHAR(255),
		slots INTEGER DEFAULT 10000,
		enabled BOOLEAN DEFAULT TRUE,
		health_check_enabled BOOLEAN DEFAULT FALSE,
		health_check_path VARCHAR(255) DEFAULT '/health',
		health_check_interval INTEGER DEFAULT 10,
		health_check_timeout INTEGER DEFAULT 5,
		healthy_threshold INTEGER DEFAULT 2,
		unhealthy_threshold INTEGER DEFAULT 3,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`
	if r.dbType == "sqlite" {
		upstreamsTable = `CREATE TABLE IF NOT EXISTS upstreams (
			id TEXT PRIMARY KEY,
			name TEXT UNIQUE NOT NULL,
			algorithm TEXT DEFAULT 'round-robin',
			hash_on TEXT DEFAULT 'none',
			hash_fallback TEXT DEFAULT 'none',
			hash_on_cookie TEXT,
			slots INTEGER DEFAULT 10000,
			enabled INTEGER DEFAULT 1,
			health_check_enabled INTEGER DEFAULT 0,
			health_check_path TEXT DEFAULT '/health',
			health_check_interval INTEGER DEFAULT 10,
			health_check_timeout INTEGER DEFAULT 5,
			healthy_threshold INTEGER DEFAULT 2,
			unhealthy_threshold INTEGER DEFAULT 3,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`
	} else if r.dbType == "mysql" {
		upstreamsTable = `CREATE TABLE IF NOT EXISTS upstreams (
			id VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) UNIQUE NOT NULL,
			algorithm VARCHAR(50) DEFAULT 'round-robin',
			hash_on VARCHAR(50) DEFAULT 'none',
			hash_fallback VARCHAR(50) DEFAULT 'none',
			hash_on_cookie VARCHAR(255),
			slots INT DEFAULT 10000,
			enabled TINYINT(1) DEFAULT 1,
			health_check_enabled TINYINT(1) DEFAULT 0,
			health_check_path VARCHAR(255) DEFAULT '/health',
			health_check_interval INT DEFAULT 10,
			health_check_timeout INT DEFAULT 5,
			healthy_threshold INT DEFAULT 2,
			unhealthy_threshold INT DEFAULT 3,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)`
	}
	if _, err := r.db.Exec(upstreamsTable); err != nil {
		fmt.Printf("Warning creating upstreams table: %v\n", err)
	}

	// 12. Kong Architecture: Targets table
	targetsTable := `CREATE TABLE IF NOT EXISTS targets (
		id VARCHAR(255) PRIMARY KEY,
		upstream_id VARCHAR(255) NOT NULL REFERENCES upstreams(id) ON DELETE CASCADE,
		target VARCHAR(255) NOT NULL,
		host VARCHAR(255) NOT NULL,
		port INTEGER NOT NULL,
		weight INTEGER DEFAULT 100,
		enabled BOOLEAN DEFAULT TRUE,
		health_status VARCHAR(50) DEFAULT 'healthy',
		failover BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`
	if r.dbType == "sqlite" {
		targetsTable = `CREATE TABLE IF NOT EXISTS targets (
			id TEXT PRIMARY KEY,
			upstream_id TEXT NOT NULL REFERENCES upstreams(id) ON DELETE CASCADE,
			target TEXT NOT NULL,
			host TEXT NOT NULL,
			port INTEGER NOT NULL,
			weight INTEGER DEFAULT 100,
			enabled INTEGER DEFAULT 1,
			health_status TEXT DEFAULT 'healthy',
			failover INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`
	} else if r.dbType == "mysql" {
		targetsTable = `CREATE TABLE IF NOT EXISTS targets (
			id VARCHAR(255) PRIMARY KEY,
			upstream_id VARCHAR(255) NOT NULL,
			target VARCHAR(255) NOT NULL,
			host VARCHAR(255) NOT NULL,
			port INT NOT NULL,
			weight INT DEFAULT 100,
			enabled TINYINT(1) DEFAULT 1,
			health_status VARCHAR(50) DEFAULT 'healthy',
			failover TINYINT(1) DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			FOREIGN KEY (upstream_id) REFERENCES upstreams(id) ON DELETE CASCADE
		)`
	}
	if _, err := r.db.Exec(targetsTable); err != nil {
		fmt.Printf("Warning creating targets table: %v\n", err)
	}

	// 13. Kong Architecture: Services table
	servicesTable := `CREATE TABLE IF NOT EXISTS services (
		id VARCHAR(255) PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		description TEXT,
		protocol VARCHAR(50) DEFAULT 'http',
		host VARCHAR(255) DEFAULT '',
		port INTEGER DEFAULT 80,
		path VARCHAR(500) DEFAULT '',
		connect_timeout INTEGER DEFAULT 60000,
		read_timeout INTEGER DEFAULT 60000,
		write_timeout INTEGER DEFAULT 60000,
		retries INTEGER DEFAULT 5,
		enabled BOOLEAN DEFAULT TRUE,
		upstream_id VARCHAR(255) REFERENCES upstreams(id) ON DELETE SET NULL,
		connection_id VARCHAR(255) REFERENCES connections(id) ON DELETE SET NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`
	if r.dbType == "sqlite" {
		servicesTable = `CREATE TABLE IF NOT EXISTS services (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			protocol TEXT DEFAULT 'http',
			host TEXT DEFAULT '',
			port INTEGER DEFAULT 80,
			path TEXT DEFAULT '',
			connect_timeout INTEGER DEFAULT 60000,
			read_timeout INTEGER DEFAULT 60000,
			write_timeout INTEGER DEFAULT 60000,
			retries INTEGER DEFAULT 5,
			enabled INTEGER DEFAULT 1,
			upstream_id TEXT REFERENCES upstreams(id) ON DELETE SET NULL,
			connection_id TEXT REFERENCES connections(id) ON DELETE SET NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`
	} else if r.dbType == "mysql" {
		servicesTable = `CREATE TABLE IF NOT EXISTS services (
			id VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			protocol VARCHAR(50) DEFAULT 'http',
			host VARCHAR(255) DEFAULT '',
			port INT DEFAULT 80,
			path VARCHAR(500) DEFAULT '',
			connect_timeout INT DEFAULT 60000,
			read_timeout INT DEFAULT 60000,
			write_timeout INT DEFAULT 60000,
			retries INT DEFAULT 5,
			enabled TINYINT(1) DEFAULT 1,
			upstream_id VARCHAR(255),
			connection_id VARCHAR(255),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			FOREIGN KEY (upstream_id) REFERENCES upstreams(id) ON DELETE SET NULL,
			FOREIGN KEY (connection_id) REFERENCES connections(id) ON DELETE SET NULL
		)`
	}
	if _, err := r.db.Exec(servicesTable); err != nil {
		fmt.Printf("Warning creating services table: %v\n", err)
	}

	// 14. Kong Architecture: Consumer Groups table
	consumerGroupsTable := `CREATE TABLE IF NOT EXISTS consumer_groups (
		id VARCHAR(255) PRIMARY KEY,
		name VARCHAR(255) UNIQUE NOT NULL,
		description TEXT,
		rate_limit_rpm INTEGER DEFAULT 60,
		rate_limit_burst INTEGER DEFAULT 10,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`
	if r.dbType == "sqlite" {
		consumerGroupsTable = `CREATE TABLE IF NOT EXISTS consumer_groups (
			id TEXT PRIMARY KEY,
			name TEXT UNIQUE NOT NULL,
			description TEXT,
			rate_limit_rpm INTEGER DEFAULT 60,
			rate_limit_burst INTEGER DEFAULT 10,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`
	} else if r.dbType == "mysql" {
		consumerGroupsTable = `CREATE TABLE IF NOT EXISTS consumer_groups (
			id VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) UNIQUE NOT NULL,
			description TEXT,
			rate_limit_rpm INT DEFAULT 60,
			rate_limit_burst INT DEFAULT 10,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)`
	}
	if _, err := r.db.Exec(consumerGroupsTable); err != nil {
		fmt.Printf("Warning creating consumer_groups table: %v\n", err)
	}

	// 15. Kong Architecture: Consumer Credentials table
	consumerCredsTable := `CREATE TABLE IF NOT EXISTS consumer_credentials (
		id VARCHAR(255) PRIMARY KEY,
		consumer_id VARCHAR(255) NOT NULL REFERENCES api_clients(id) ON DELETE CASCADE,
		name VARCHAR(255) NOT NULL,
		key_type VARCHAR(50) DEFAULT 'api-key',
		key_id VARCHAR(255) UNIQUE NOT NULL,
		secret_hash TEXT NOT NULL,
		masked_key VARCHAR(255) NOT NULL,
		enabled BOOLEAN DEFAULT TRUE,
		expires_at TIMESTAMP NULL,
		last_used_at TIMESTAMP NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`
	if r.dbType == "sqlite" {
		consumerCredsTable = `CREATE TABLE IF NOT EXISTS consumer_credentials (
			id TEXT PRIMARY KEY,
			consumer_id TEXT NOT NULL REFERENCES api_clients(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			key_type TEXT DEFAULT 'api-key',
			key_id TEXT UNIQUE NOT NULL,
			secret_hash TEXT NOT NULL,
			masked_key TEXT NOT NULL,
			enabled INTEGER DEFAULT 1,
			expires_at DATETIME NULL,
			last_used_at DATETIME NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`
	} else if r.dbType == "mysql" {
		consumerCredsTable = `CREATE TABLE IF NOT EXISTS consumer_credentials (
			id VARCHAR(255) PRIMARY KEY,
			consumer_id VARCHAR(255) NOT NULL,
			name VARCHAR(255) NOT NULL,
			key_type VARCHAR(50) DEFAULT 'api-key',
			key_id VARCHAR(255) UNIQUE NOT NULL,
			secret_hash TEXT NOT NULL,
			masked_key VARCHAR(255) NOT NULL,
			enabled TINYINT(1) DEFAULT 1,
			expires_at TIMESTAMP NULL,
			last_used_at TIMESTAMP NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (consumer_id) REFERENCES api_clients(id) ON DELETE CASCADE
		)`
	}
	if _, err := r.db.Exec(consumerCredsTable); err != nil {
		fmt.Printf("Warning creating consumer_credentials table: %v\n", err)
	}

	// 16. Kong Architecture: Extend endpoints table with Route concepts
	if err := r.ensureColumnExists("endpoints", "service_id", "VARCHAR(255)"); err != nil {
		return err
	}
	if err := r.ensureColumnExists("endpoints", "name", "VARCHAR(255)"); err != nil {
		return err
	}
	if err := r.ensureColumnExists("endpoints", "protocols", "VARCHAR(100) DEFAULT 'http,https'"); err != nil {
		return err
	}
	if err := r.ensureColumnExists("endpoints", "strip_path", fmt.Sprintf("%s DEFAULT %s", boolType, falseDefault)); err != nil {
		return err
	}
	if err := r.ensureColumnExists("endpoints", "preserve_host", fmt.Sprintf("%s DEFAULT %s", boolType, falseDefault)); err != nil {
		return err
	}
	if err := r.ensureColumnExists("endpoints", "regex_priority", "INTEGER DEFAULT 0"); err != nil {
		return err
	}
	if err := r.ensureColumnExists("endpoints", "https_redirect_status_code", "INTEGER DEFAULT 426"); err != nil {
		return err
	}

	// 17. Kong Architecture: Extend api_clients table with Consumer concepts
	if err := r.ensureColumnExists("api_clients", "custom_id", "VARCHAR(255)"); err != nil {
		return err
	}
	if err := r.ensureColumnExists("api_clients", "status", "VARCHAR(50) DEFAULT 'active'"); err != nil {
		return err
	}
	if err := r.ensureColumnExists("api_clients", "updated_at", "TIMESTAMP DEFAULT CURRENT_TIMESTAMP"); err != nil {
		return err
	}

	// 18. Auto-migration logic: Create default Services for existing database connections
	if r.dbType == "postgres" || r.dbType == "postgresql" {
		_, _ = r.db.Exec(`
			INSERT INTO services (id, name, description, protocol, host, port, path, enabled, connection_id, created_at, updated_at)
			SELECT c.id, c.name, 'Database Data Source for ' || c.name, 'database', c.host, c.port, '', TRUE, c.id, c.created_at, CURRENT_TIMESTAMP
			FROM connections c
			WHERE NOT EXISTS (SELECT 1 FROM services s WHERE s.connection_id = c.id)
			ON CONFLICT DO NOTHING;

			UPDATE endpoints e
			SET service_id = s.id
			FROM queries q
			JOIN services s ON s.connection_id = q.connection_id
			WHERE e.query_id = q.id AND (e.service_id IS NULL OR e.service_id = '');

			UPDATE endpoints SET name = path WHERE name IS NULL OR name = '';
			UPDATE endpoints SET protocols = 'http,https' WHERE protocols IS NULL OR protocols = '';
			UPDATE api_clients SET status = 'active' WHERE status IS NULL OR status = '';
		`)
	} else if r.dbType == "sqlite" {
		_, _ = r.db.Exec(`
			INSERT OR IGNORE INTO services (id, name, description, protocol, host, port, path, enabled, connection_id, created_at, updated_at)
			SELECT c.id, c.name, 'Database Data Source for ' || c.name, 'database', c.host, c.port, '', 1, c.id, c.created_at, CURRENT_TIMESTAMP
			FROM connections c
			WHERE NOT EXISTS (SELECT 1 FROM services s WHERE s.connection_id = c.id);

			UPDATE endpoints SET name = path WHERE name IS NULL OR name = '';
			UPDATE endpoints SET protocols = 'http,https' WHERE protocols IS NULL OR protocols = '';
			UPDATE api_clients SET status = 'active' WHERE status IS NULL OR status = '';
		`)
	}

	// Update existing endpoints max_limit from 1000 to 5000
	_, _ = r.db.Exec("UPDATE endpoints SET max_limit = 5000 WHERE max_limit = 1000")

	// 19. System Settings table for white-labeling & platform configuration
	_, _ = r.db.Exec(`CREATE TABLE IF NOT EXISTS system_settings (
		key VARCHAR(100) PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)

	// 20. Auth Plugin Configs table (Kong-style inbound auth plugins)
	authPluginTable := `CREATE TABLE IF NOT EXISTS auth_plugin_configs (
		id VARCHAR(255) PRIMARY KEY,
		plugin_name VARCHAR(50) NOT NULL,
		scope VARCHAR(20) NOT NULL DEFAULT 'route',
		service_id VARCHAR(255) REFERENCES services(id) ON DELETE CASCADE,
		route_id VARCHAR(255) REFERENCES endpoints(id) ON DELETE CASCADE,
		consumer_id VARCHAR(255) REFERENCES api_clients(id) ON DELETE CASCADE,
		config JSONB NOT NULL DEFAULT '{}',
		enabled BOOLEAN DEFAULT TRUE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		CONSTRAINT unique_plugin_scope UNIQUE (plugin_name, scope, service_id, route_id, consumer_id)
	)`
	if r.dbType == "sqlite" {
		authPluginTable = `CREATE TABLE IF NOT EXISTS auth_plugin_configs (
			id TEXT PRIMARY KEY,
			plugin_name TEXT NOT NULL,
			scope TEXT NOT NULL DEFAULT 'route',
			service_id TEXT REFERENCES services(id) ON DELETE CASCADE,
			route_id TEXT REFERENCES endpoints(id) ON DELETE CASCADE,
			consumer_id TEXT REFERENCES api_clients(id) ON DELETE CASCADE,
			config TEXT NOT NULL DEFAULT '{}',
			enabled INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`
	}
	if _, err := r.db.Exec(authPluginTable); err != nil {
		fmt.Printf("Warning creating auth_plugin_configs table: %v\n", err)
	}

	// 21. Service Outbound Auth table (credential injection to upstream backends)
	outboundAuthTable := `CREATE TABLE IF NOT EXISTS service_outbound_auth (
		id VARCHAR(255) PRIMARY KEY,
		service_id VARCHAR(255) NOT NULL REFERENCES services(id) ON DELETE CASCADE,
		auth_type VARCHAR(50) NOT NULL,
		config JSONB NOT NULL DEFAULT '{}',
		enabled BOOLEAN DEFAULT TRUE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		CONSTRAINT unique_service_outbound UNIQUE (service_id)
	)`
	if r.dbType == "sqlite" {
		outboundAuthTable = `CREATE TABLE IF NOT EXISTS service_outbound_auth (
			id TEXT PRIMARY KEY,
			service_id TEXT NOT NULL REFERENCES services(id) ON DELETE CASCADE,
			auth_type TEXT NOT NULL,
			config TEXT NOT NULL DEFAULT '{}',
			enabled INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`
	}
	if _, err := r.db.Exec(outboundAuthTable); err != nil {
		fmt.Printf("Warning creating service_outbound_auth table: %v\n", err)
	}

	return nil
}

// ensureColumnExists checks if a column exists and adds it if not
func (r *Registry) ensureColumnExists(table, column, definition string) error {
	// Guard: table and column must be safe identifiers before any SQL construction.
	// All callers in this file use hardcoded literals, but this prevents any future
	// misuse from propagating into SQL statements.
	if !isValidIdentifier(table) || !isValidIdentifier(column) {
		return fmt.Errorf("ensureColumnExists: invalid identifier table=%q column=%q", table, column)
	}

	// Different query depending on DB type
	var exists bool
	var checkQuery string

	if r.dbType == "postgres" || r.dbType == "postgresql" {
		// Use parameterized query — table_name and column_name are bound args, not interpolated.
		checkQuery = "SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name=$1 AND column_name=$2)"
	} else if r.dbType == "mysql" {
		// Use parameterized query with positional placeholders for MySQL.
		checkQuery = "SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name=? AND column_name=? AND table_schema=DATABASE())"
	} else { // sqlite
		// SQLite PRAGMA does not support parameterized queries; identifier is
		// validated by isValidIdentifier() above before reaching this point.
		rows, err := r.db.Query("PRAGMA table_info(" + table + ")")
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var cid int
			var name, ctype string
			var notnull, pk int
			var dfltValue interface{}
			if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
				return err
			}
			if name == column {
				exists = true
				break
			}
		}

		if !exists {
			_, err := r.db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition))
			return err
		}
		return nil
	}

	// For Postgres/MySQL — pass table and column as bound parameters.
	if err := r.db.Get(&exists, checkQuery, table, column); err != nil {
		return fmt.Errorf("failed to check column %s in %s: %w", column, table, err)
	}

	if !exists {
		// Postgres/MySQL syntax
		alterQuery := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition)
		if _, err := r.db.Exec(alterQuery); err != nil {
			return fmt.Errorf("failed to add column %s to %s: %w", column, table, err)
		}
		fmt.Printf("Migrated: Added column %s to table %s\n", column, table)
	}

	return nil
}
