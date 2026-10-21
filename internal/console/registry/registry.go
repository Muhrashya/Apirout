package registry

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

// DBConfig holds database configuration
type DBConfig struct {
	Type     string // "sqlite", "postgres", "mysql"
	Host     string
	Port     int
	Database string
	Username string
	Password string
	SSLMode  string // for postgres: disable, require, verify-ca, verify-full
	DataDir  string // for sqlite only
}

// Publisher defines the invalidation event emitter interface for control-plane mutations
type Publisher interface {
	PublishEndpointChanged(action, endpointID, path string) error
	PublishConnectionChanged(action, connectionID string) error
	PublishTargetHealthChanged(targetID, healthStatus string) error
	PublishBulkConfigUpdated() error
}

// Registry manages the database for app state
type Registry struct {
	db        *sqlx.DB
	dbType    string
	publisher Publisher
}

// SetPublisher assigns the event publisher for automated cache invalidation on persistence
func (r *Registry) SetPublisher(pub Publisher) {
	r.publisher = pub
}

// GetPublisher returns the configured event publisher, if any
func (r *Registry) GetPublisher() Publisher {
	return r.publisher
}

// New creates a new Registry instance with configurable database backend
func New(dataDir string) (*Registry, error) {
	// Check for DATABASE_URL or individual env vars
	dbType := getEnvDefault("DB_TYPE", "sqlite")

	switch dbType {
	case "postgres", "postgresql":
		return NewWithPostgres()
	case "mysql":
		return NewWithMySQL()
	default:
		// Use DB_PATH if set, otherwise use dataDir/registry.db
		dbPath := os.Getenv("DB_PATH")
		if dbPath == "" {
			dbPath = filepath.Join(dataDir, "registry.db")
		}
		return NewWithSQLite(dbPath)
	}
}

// NewWithConfig creates a Registry with explicit configuration
func NewWithConfig(cfg DBConfig) (*Registry, error) {
	switch cfg.Type {
	case "postgres", "postgresql":
		dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.Database, cfg.SSLMode)
		return connectDB("postgres", dsn)
	case "mysql":
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
			cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
		return connectDB("mysql", dsn)
	default:
		return NewWithSQLite(cfg.DataDir)
	}
}

// NewWithSQLite creates a Registry with SQLite backend
// dbPath can be full path to .db file or directory
func NewWithSQLite(dbPath string) (*Registry, error) {
	// Check if dbPath is a directory or file
	if !strings.HasSuffix(dbPath, ".db") {
		// It's a directory, append registry.db
		if err := os.MkdirAll(dbPath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create data directory: %w", err)
		}
		dbPath = filepath.Join(dbPath, "registry.db")
	} else {
		// It's a file path, ensure parent directory exists
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create data directory: %w", err)
		}
	}

	db, err := sqlx.Connect("sqlite", dbPath+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SQLite: %w", err)
	}

	return &Registry{db: db.Unsafe(), dbType: "sqlite"}, nil
}

// NewWithPostgres creates a Registry with PostgreSQL backend
func NewWithPostgres() (*Registry, error) {
	host := getEnvDefault("DB_HOST", "localhost")
	port := getEnvDefault("DB_PORT", "5432")
	user := getEnvDefault("DB_USER", "postgres")
	pass := os.Getenv("DB_PASSWORD")
	dbname := getEnvDefault("DB_NAME", "apirout")
	sslmode := getEnvDefault("DB_SSLMODE", "disable")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, pass, dbname, sslmode)

	return connectDB("postgres", dsn)
}

// NewWithMySQL creates a Registry with MySQL backend
func NewWithMySQL() (*Registry, error) {
	host := getEnvDefault("DB_HOST", "localhost")
	port := getEnvDefault("DB_PORT", "3306")
	user := getEnvDefault("DB_USER", "root")
	pass := os.Getenv("DB_PASSWORD")
	dbname := getEnvDefault("DB_NAME", "apirout")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4",
		user, pass, host, port, dbname)

	return connectDB("mysql", dsn)
}

func connectDB(driverName, dsn string) (*Registry, error) {
	db, err := sqlx.Connect(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", driverName, err)
	}

	// Configure pool from granular environment variables
	maxOpen := getEnvInt("DB_MAX_OPEN_CONNS", 25)
	maxIdle := getEnvInt("DB_MAX_IDLE_CONNS", 5)
	maxLifetime := getEnvDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute)
	maxIdleTime := getEnvDuration("DB_CONN_MAX_IDLE_TIME", 10*time.Minute)

	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(maxLifetime)
	db.SetConnMaxIdleTime(maxIdleTime)

	return &Registry{db: db.Unsafe(), dbType: driverName}, nil
}

func getEnvDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		var i int
		if _, err := fmt.Sscanf(val, "%d", &i); err == nil && i > 0 {
			return i
		}
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
		var sec int
		if _, err := fmt.Sscanf(val, "%d", &sec); err == nil && sec > 0 {
			return time.Duration(sec) * time.Second
		}
	}
	return defaultVal
}

// Close closes the database connection
func (r *Registry) Close() error {
	return r.db.Close()
}

// DB returns the underlying sqlx.DB instance
func (r *Registry) DB() *sqlx.DB {
	return r.db
}

// DBType returns the database type
func (r *Registry) DBType() string {
	return r.dbType
}

// placeholder returns the correct placeholder for the database type
func (r *Registry) placeholder(index int) string {
	switch r.dbType {
	case "postgres":
		return fmt.Sprintf("$%d", index)
	default:
		return "?"
	}
}

// Migrate runs database migrations
func (r *Registry) Migrate() error {
	// Use appropriate SQL syntax based on database type
	var migrations []string

	switch r.dbType {
	case "postgres":
		migrations = r.getPostgresMigrations()
	case "mysql":
		migrations = r.getMySQLMigrations()
	default:
		migrations = r.getSQLiteMigrations()
	}

	for _, m := range migrations {
		if _, err := r.db.Exec(m); err != nil {
			// Ignore "already exists" and "duplicate column" errors
			errStr := strings.ToLower(err.Error())
			if strings.Contains(errStr, "already exists") ||
				strings.Contains(errStr, "duplicate column") ||
				strings.Contains(errStr, "duplicate key") ||
				strings.Contains(errStr, "unique constraint") {
				continue
			}
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}

func (r *Registry) getSQLiteMigrations() []string {
	return []string{
		// Users table
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'MEMBER',
			password_changed_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`ALTER TABLE users ADD COLUMN password_changed_at DATETIME`,

		// Groups table
		`CREATE TABLE IF NOT EXISTS groups (
			id TEXT PRIMARY KEY,
			name TEXT UNIQUE NOT NULL,
			description TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// User-Group assignments with permissions
		`CREATE TABLE IF NOT EXISTS user_groups (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			group_id TEXT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
			can_manage_connections INTEGER DEFAULT 0,
			can_manage_queries INTEGER DEFAULT 0,
			can_manage_endpoints INTEGER DEFAULT 0,
			can_execute_endpoints INTEGER DEFAULT 1,
			can_view_docs INTEGER DEFAULT 1,
			can_view_dashboard INTEGER DEFAULT 1,
			can_view_audit_logs INTEGER DEFAULT 0,
			UNIQUE(user_id, group_id)
		)`,

		// Connections table
		`CREATE TABLE IF NOT EXISTS connections (
			id TEXT PRIMARY KEY,
			group_id TEXT REFERENCES groups(id) ON DELETE SET NULL,
			name TEXT NOT NULL,
			db_type TEXT NOT NULL,
			host TEXT NOT NULL,
			port INTEGER NOT NULL,
			db_name TEXT NOT NULL,
			username TEXT NOT NULL,
			password_encrypted TEXT NOT NULL,
			ssl_mode TEXT DEFAULT 'disable',
			is_read_only INTEGER DEFAULT 0,
			ssh_enabled INTEGER DEFAULT 0,
			ssh_host TEXT,
			ssh_port INTEGER DEFAULT 22,
			ssh_user TEXT,
			ssh_password_encrypted TEXT,
			ssh_private_key_encrypted TEXT,
			connection_health TEXT,
			last_tested_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Queries table
		`CREATE TABLE IF NOT EXISTS queries (
			id TEXT PRIMARY KEY,
			group_id TEXT REFERENCES groups(id) ON DELETE SET NULL,
			connection_id TEXT REFERENCES connections(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			description TEXT,
			sql_text TEXT NOT NULL,
			count_sql_text TEXT,
			params_schema TEXT DEFAULT '{}',
			preview_limit INTEGER DEFAULT 200,
			preview_offset INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Endpoints table
		`CREATE TABLE IF NOT EXISTS endpoints (
			id TEXT PRIMARY KEY,
			group_id TEXT REFERENCES groups(id) ON DELETE SET NULL,
			method TEXT NOT NULL DEFAULT 'GET',
			path TEXT UNIQUE NOT NULL,
			query_id TEXT REFERENCES queries(id) ON DELETE CASCADE,
			enabled INTEGER DEFAULT 1,
			is_public INTEGER DEFAULT 0,
			rate_limit_enabled INTEGER DEFAULT 1,
			rate_limit_rpm INTEGER DEFAULT 60,
			rate_limit_burst INTEGER DEFAULT 10,
			default_limit INTEGER DEFAULT 100,
			max_limit INTEGER DEFAULT 5000,
			return_total_count INTEGER DEFAULT 0,
			allow_dynamic_filters INTEGER DEFAULT 0,
			created_by_user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// API Clients table
		`CREATE TABLE IF NOT EXISTS api_clients (
			id TEXT PRIMARY KEY,
			group_id TEXT REFERENCES groups(id) ON DELETE CASCADE,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Audit logs table
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id TEXT PRIMARY KEY,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			event_type TEXT NOT NULL,
			user_id TEXT,
			api_client_id TEXT,
			ip_address TEXT,
			details TEXT,
			success INTEGER DEFAULT 1
		)`,

		// Metrics table
		`CREATE TABLE IF NOT EXISTS metrics (
			id TEXT PRIMARY KEY,
			endpoint_id TEXT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			response_time_ms INTEGER,
			status_code INTEGER,
			error TEXT
		)`,

		// Add missing columns to audit_logs (for older databases)
		`ALTER TABLE audit_logs ADD COLUMN endpoint_id TEXT`,
		`ALTER TABLE audit_logs ADD COLUMN duration_ms INTEGER`,
		`ALTER TABLE audit_logs ADD COLUMN error_message TEXT`,
		`ALTER TABLE audit_logs ADD COLUMN created_at DATETIME`,
		`ALTER TABLE audit_logs ADD COLUMN ip_address TEXT`,

		// Add missing columns to groups (for older databases)
		`ALTER TABLE groups ADD COLUMN created_at DATETIME`,

		// Add missing columns to metrics (for older databases)
		`ALTER TABLE metrics ADD COLUMN duration_ms INTEGER`,

		// Add missing columns to connections (for older databases)
		`ALTER TABLE connections ADD COLUMN dbname TEXT`,
		`ALTER TABLE connections ADD COLUMN sslmode TEXT`,
		`ALTER TABLE connections ADD COLUMN is_read_only INTEGER DEFAULT 0`,
		`ALTER TABLE connections ADD COLUMN ssh_auth_method TEXT`,
		`ALTER TABLE connections ADD COLUMN updated_at DATETIME`,

		// Add missing column to endpoints (for older databases)
		`ALTER TABLE endpoints ADD COLUMN allow_dynamic_filters INTEGER DEFAULT 0`,

		// Connection health table (separate from connections.connection_health column)
		`CREATE TABLE IF NOT EXISTS connection_health (
			connection_id TEXT PRIMARY KEY REFERENCES connections(id) ON DELETE CASCADE,
			last_success INTEGER DEFAULT 0,
			last_latency_ms INTEGER,
			last_error_message TEXT,
			last_checked_at DATETIME
		)`,

		// Indexes
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp ON audit_logs(timestamp DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_metrics_timestamp ON metrics(timestamp DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_metrics_endpoint_id ON metrics(endpoint_id)`,
		`CREATE INDEX IF NOT EXISTS idx_endpoints_path ON endpoints(path)`,
		`CREATE INDEX IF NOT EXISTS idx_endpoints_created_by ON endpoints(created_by_user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_endpoints_group_id ON endpoints(group_id)`,

		// API Client Endpoints allowlist table
		`CREATE TABLE IF NOT EXISTS api_client_endpoints (
			api_client_id TEXT NOT NULL REFERENCES api_clients(id) ON DELETE CASCADE,
			endpoint_id TEXT NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (api_client_id, endpoint_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_ace_endpoint ON api_client_endpoints(endpoint_id)`,

		// Migration: auto-allow endpoints for existing API clients with group_id
		`INSERT OR IGNORE INTO api_client_endpoints (api_client_id, endpoint_id, created_at)
		 SELECT ac.id, e.id, CURRENT_TIMESTAMP
		 FROM api_clients ac
		 INNER JOIN endpoints e ON e.group_id = ac.group_id
		 WHERE ac.group_id IS NOT NULL`,
	}
}

func (r *Registry) getPostgresMigrations() []string {
	return []string{
		// Users table
		`CREATE TABLE IF NOT EXISTS users (
			id VARCHAR(255) PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			role VARCHAR(50) NOT NULL DEFAULT 'MEMBER',
			password_changed_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS password_changed_at TIMESTAMP`,

		// Groups table
		`CREATE TABLE IF NOT EXISTS groups (
			id VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) UNIQUE NOT NULL,
			description TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// User-Group assignments with permissions
		`CREATE TABLE IF NOT EXISTS user_groups (
			id VARCHAR(255) PRIMARY KEY,
			user_id VARCHAR(255) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			group_id VARCHAR(255) NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
			can_manage_connections BOOLEAN DEFAULT FALSE,
			can_manage_queries BOOLEAN DEFAULT FALSE,
			can_manage_endpoints BOOLEAN DEFAULT FALSE,
			can_execute_endpoints BOOLEAN DEFAULT TRUE,
			can_view_docs BOOLEAN DEFAULT TRUE,
			can_view_dashboard BOOLEAN DEFAULT TRUE,
			can_view_audit_logs BOOLEAN DEFAULT FALSE,
			UNIQUE(user_id, group_id)
		)`,

		// Connections table
		`CREATE TABLE IF NOT EXISTS connections (
			id VARCHAR(255) PRIMARY KEY,
			group_id VARCHAR(255) REFERENCES groups(id) ON DELETE SET NULL,
			name VARCHAR(255) NOT NULL,
			db_type VARCHAR(50) NOT NULL,
			host VARCHAR(255) NOT NULL,
			port INTEGER NOT NULL,
			db_name VARCHAR(255) NOT NULL,
			username VARCHAR(255) NOT NULL,
			password_encrypted TEXT NOT NULL,
			ssl_mode VARCHAR(50) DEFAULT 'disable',
			is_read_only BOOLEAN DEFAULT FALSE,
			ssh_enabled BOOLEAN DEFAULT FALSE,
			ssh_host VARCHAR(255),
			ssh_port INTEGER DEFAULT 22,
			ssh_user VARCHAR(255),
			ssh_auth_method VARCHAR(50) DEFAULT 'password',
			ssh_password_encrypted TEXT,
			ssh_private_key_encrypted TEXT,
			connection_health TEXT,
			last_tested_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Add missing columns to connections (for older databases)
		`ALTER TABLE connections ADD COLUMN IF NOT EXISTS is_read_only BOOLEAN DEFAULT FALSE`,
		`ALTER TABLE connections ADD COLUMN IF NOT EXISTS ssh_auth_method VARCHAR(50) DEFAULT 'password'`,
		`ALTER TABLE connections ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP`,

		// Queries table
		`CREATE TABLE IF NOT EXISTS queries (
			id VARCHAR(255) PRIMARY KEY,
			group_id VARCHAR(255) REFERENCES groups(id) ON DELETE SET NULL,
			connection_id VARCHAR(255) REFERENCES connections(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			sql_text TEXT NOT NULL,
			params_schema TEXT DEFAULT '{}',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Endpoints table
		`CREATE TABLE IF NOT EXISTS endpoints (
			id VARCHAR(255) PRIMARY KEY,
			group_id VARCHAR(255) REFERENCES groups(id) ON DELETE SET NULL,
			method VARCHAR(10) NOT NULL DEFAULT 'GET',
			path VARCHAR(500) UNIQUE NOT NULL,
			query_id VARCHAR(255) REFERENCES queries(id) ON DELETE CASCADE,
			enabled BOOLEAN DEFAULT TRUE,
			is_public BOOLEAN DEFAULT FALSE,
			rate_limit_enabled BOOLEAN DEFAULT TRUE,
			rate_limit_rpm INTEGER DEFAULT 60,
			rate_limit_burst INTEGER DEFAULT 10,
			default_limit INTEGER DEFAULT 100,
			max_limit INTEGER DEFAULT 5000,
			return_total_count BOOLEAN DEFAULT FALSE,
			allow_dynamic_filters BOOLEAN DEFAULT FALSE,
			created_by_user_id VARCHAR(255) REFERENCES users(id) ON DELETE SET NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Add missing column to endpoints (for older databases)
		`ALTER TABLE endpoints ADD COLUMN IF NOT EXISTS allow_dynamic_filters BOOLEAN DEFAULT FALSE`,

		// API Clients table
		`CREATE TABLE IF NOT EXISTS api_clients (
			id VARCHAR(255) PRIMARY KEY,
			group_id VARCHAR(255) REFERENCES groups(id) ON DELETE CASCADE,
			username VARCHAR(255) UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Audit logs table
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id VARCHAR(255) PRIMARY KEY,
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			event_type VARCHAR(100) NOT NULL,
			user_id VARCHAR(255),
			api_client_id VARCHAR(255),
			ip_address VARCHAR(255),
			details TEXT,
			success BOOLEAN DEFAULT TRUE
		)`,

		// Metrics table
		`CREATE TABLE IF NOT EXISTS metrics (
			id VARCHAR(255) PRIMARY KEY,
			endpoint_id VARCHAR(255),
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			duration_ms INTEGER,
			status_code INTEGER,
			error TEXT
		)`,

		// Add missing columns to audit_logs (for older databases)
		`ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS endpoint_id VARCHAR(255)`,
		`ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS duration_ms INTEGER`,
		`ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS error_message TEXT`,
		`ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS ip_address VARCHAR(255)`,

		// Add missing columns to metrics (for older databases)
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS duration_ms INTEGER`,

		// Connection health table
		`CREATE TABLE IF NOT EXISTS connection_health (
			connection_id VARCHAR(255) PRIMARY KEY REFERENCES connections(id) ON DELETE CASCADE,
			last_success BOOLEAN DEFAULT FALSE,
			last_latency_ms INTEGER,
			last_error_message TEXT,
			last_checked_at TIMESTAMP
		)`,

		// Indexes
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp ON audit_logs(timestamp DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_metrics_timestamp ON metrics(timestamp DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_metrics_endpoint_id ON metrics(endpoint_id)`,
		`CREATE INDEX IF NOT EXISTS idx_endpoints_path ON endpoints(path)`,
		`CREATE INDEX IF NOT EXISTS idx_endpoints_created_by ON endpoints(created_by_user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_endpoints_group_id ON endpoints(group_id)`,

		// API Client Endpoints allowlist table
		`CREATE TABLE IF NOT EXISTS api_client_endpoints (
			api_client_id VARCHAR(255) NOT NULL REFERENCES api_clients(id) ON DELETE CASCADE,
			endpoint_id VARCHAR(255) NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (api_client_id, endpoint_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_ace_endpoint ON api_client_endpoints(endpoint_id)`,

		// Migration: auto-allow endpoints for existing API clients with group_id
		`INSERT INTO api_client_endpoints (api_client_id, endpoint_id, created_at)
		 SELECT ac.id, e.id, CURRENT_TIMESTAMP
		 FROM api_clients ac
		 INNER JOIN endpoints e ON e.group_id = ac.group_id
		 WHERE ac.group_id IS NOT NULL
		 ON CONFLICT DO NOTHING`,
	}
}

func (r *Registry) getMySQLMigrations() []string {
	return []string{
		// Users table
		`CREATE TABLE IF NOT EXISTS users (
			id VARCHAR(255) PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			role VARCHAR(50) NOT NULL DEFAULT 'MEMBER',
			password_changed_at DATETIME,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`ALTER TABLE users ADD COLUMN password_changed_at DATETIME`,

		// Groups table
		`CREATE TABLE IF NOT EXISTS ` + "`groups`" + ` (
			id VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) UNIQUE NOT NULL,
			description TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// User-Group assignments with permissions
		`CREATE TABLE IF NOT EXISTS user_groups (
			id VARCHAR(255) PRIMARY KEY,
			user_id VARCHAR(255) NOT NULL,
			group_id VARCHAR(255) NOT NULL,
			can_manage_connections TINYINT(1) DEFAULT 0,
			can_manage_queries TINYINT(1) DEFAULT 0,
			can_manage_endpoints TINYINT(1) DEFAULT 0,
			can_execute_endpoints TINYINT(1) DEFAULT 1,
			can_view_docs TINYINT(1) DEFAULT 1,
			can_view_dashboard TINYINT(1) DEFAULT 1,
			can_view_audit_logs TINYINT(1) DEFAULT 0,
			UNIQUE KEY unique_user_group (user_id, group_id),
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (group_id) REFERENCES ` + "`groups`" + `(id) ON DELETE CASCADE
		)`,

		// Connections table
		`CREATE TABLE IF NOT EXISTS connections (
			id VARCHAR(255) PRIMARY KEY,
			group_id VARCHAR(255),
			name VARCHAR(255) NOT NULL,
			db_type VARCHAR(50) NOT NULL,
			host VARCHAR(255) NOT NULL,
			port INT NOT NULL,
			db_name VARCHAR(255) NOT NULL,
			username VARCHAR(255) NOT NULL,
			password_encrypted TEXT NOT NULL,
			ssl_mode VARCHAR(50) DEFAULT 'disable',
			is_read_only TINYINT(1) DEFAULT 0,
			ssh_enabled TINYINT(1) DEFAULT 0,
			ssh_host VARCHAR(255),
			ssh_port INT DEFAULT 22,
			ssh_user VARCHAR(255),
			ssh_password_encrypted TEXT,
			ssh_private_key_encrypted TEXT,
			connection_health TEXT,
			last_tested_at TIMESTAMP NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (group_id) REFERENCES ` + "`groups`" + `(id) ON DELETE SET NULL
		)`,

		// Queries table
		`CREATE TABLE IF NOT EXISTS queries (
			id VARCHAR(255) PRIMARY KEY,
			group_id VARCHAR(255),
			connection_id VARCHAR(255),
			name VARCHAR(255) NOT NULL,
			description TEXT,
			sql_text TEXT NOT NULL,
			params_schema TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			FOREIGN KEY (group_id) REFERENCES ` + "`groups`" + `(id) ON DELETE SET NULL,
			FOREIGN KEY (connection_id) REFERENCES connections(id) ON DELETE CASCADE
		)`,

		// Endpoints table
		`CREATE TABLE IF NOT EXISTS endpoints (
			id VARCHAR(255) PRIMARY KEY,
			group_id VARCHAR(255),
			method VARCHAR(10) NOT NULL DEFAULT 'GET',
			path VARCHAR(500) UNIQUE NOT NULL,
			query_id VARCHAR(255),
			enabled TINYINT(1) DEFAULT 1,
			is_public TINYINT(1) DEFAULT 0,
			rate_limit_enabled TINYINT(1) DEFAULT 1,
			rate_limit_rpm INT DEFAULT 60,
			rate_limit_burst INT DEFAULT 10,
			default_limit INT DEFAULT 100,
			max_limit INT DEFAULT 5000,
			return_total_count TINYINT(1) DEFAULT 0,
			allow_dynamic_filters TINYINT(1) DEFAULT 0,
			created_by_user_id VARCHAR(255),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			FOREIGN KEY (group_id) REFERENCES ` + "`groups`" + `(id) ON DELETE SET NULL,
			FOREIGN KEY (query_id) REFERENCES queries(id) ON DELETE CASCADE,
			FOREIGN KEY (created_by_user_id) REFERENCES users(id) ON DELETE SET NULL
		)`,

		// API Clients table
		`CREATE TABLE IF NOT EXISTS api_clients (
			id VARCHAR(255) PRIMARY KEY,
			group_id VARCHAR(255),
			username VARCHAR(255) UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (group_id) REFERENCES ` + "`groups`" + `(id) ON DELETE CASCADE
		)`,

		// Audit logs table
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id VARCHAR(255) PRIMARY KEY,
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			event_type VARCHAR(100) NOT NULL,
			user_id VARCHAR(255),
			api_client_id VARCHAR(255),
			ip_address VARCHAR(255),
			details TEXT,
			success TINYINT(1) DEFAULT 1
		)`,

		// Metrics table
		`CREATE TABLE IF NOT EXISTS metrics (
			id VARCHAR(255) PRIMARY KEY,
			endpoint_id VARCHAR(255),
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			response_time_ms INT,
			status_code INT,
			error TEXT
		)`,

		// Add missing columns to audit_logs (for older databases)
		// MySQL doesn't support IF NOT EXISTS for ALTER, so we ignore errors
		`ALTER TABLE audit_logs ADD COLUMN endpoint_id VARCHAR(255)`,
		`ALTER TABLE audit_logs ADD COLUMN duration_ms INT`,
		`ALTER TABLE audit_logs ADD COLUMN error_message TEXT`,
		`ALTER TABLE audit_logs ADD COLUMN ip_address VARCHAR(255)`,
		`ALTER TABLE connections ADD COLUMN is_read_only TINYINT(1) DEFAULT 0`,

		// Indexes
		`CREATE INDEX idx_audit_logs_timestamp ON audit_logs(timestamp DESC)`,
		`CREATE INDEX idx_metrics_timestamp ON metrics(timestamp DESC)`,
		`CREATE INDEX idx_metrics_endpoint_id ON metrics(endpoint_id)`,
		`CREATE INDEX idx_endpoints_path ON endpoints(path)`,
		`CREATE INDEX idx_endpoints_created_by ON endpoints(created_by_user_id)`,
		`CREATE INDEX idx_endpoints_group_id ON endpoints(group_id)`,

		// API Client Endpoints allowlist table
		`CREATE TABLE IF NOT EXISTS api_client_endpoints (
			api_client_id VARCHAR(255) NOT NULL,
			endpoint_id VARCHAR(255) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (api_client_id, endpoint_id),
			FOREIGN KEY (api_client_id) REFERENCES api_clients(id) ON DELETE CASCADE,
			FOREIGN KEY (endpoint_id) REFERENCES endpoints(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX idx_ace_endpoint ON api_client_endpoints(endpoint_id)`,

		// Migration: auto-allow endpoints for existing API clients with group_id
		`INSERT IGNORE INTO api_client_endpoints (api_client_id, endpoint_id, created_at)
		 SELECT ac.id, e.id, CURRENT_TIMESTAMP
		 FROM api_clients ac
		 INNER JOIN endpoints e ON e.group_id = ac.group_id
		 WHERE ac.group_id IS NOT NULL`,
	}
}

// EnsureSuperAdmin creates default super admin if not exists
func (r *Registry) EnsureSuperAdmin(email, password string) error {
	var count int
	err := r.db.Get(&count, "SELECT COUNT(*) FROM users WHERE role = 'SUPER_ADMIN'")
	if err != nil {
		return err
	}

	if count > 0 {
		return nil // Already exists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	id := generateID()
	_, err = r.db.Exec(
		r.db.Rebind("INSERT INTO users (id, email, password_hash, role) VALUES (?, ?, ?, ?)"),
		id, email, string(hash), "SUPER_ADMIN",
	)
	return err
}

// HasSuperAdmin checks if at least one SUPER_ADMIN user exists in the registry
func (r *Registry) HasSuperAdmin() (bool, error) {
	var count int
	err := r.db.Get(&count, "SELECT COUNT(*) FROM users WHERE role = 'SUPER_ADMIN'")
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// CreateSuperAdmin creates the initial SUPER_ADMIN user
func (r *Registry) CreateSuperAdmin(email, password string) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	id := generateID()
	now := time.Now()
	_, err = r.db.Exec(
		r.db.Rebind("INSERT INTO users (id, email, password_hash, role, created_at) VALUES (?, ?, ?, ?, ?)"),
		id, email, string(hash), "SUPER_ADMIN", now,
	)
	if err != nil {
		return nil, err
	}

	return &User{
		ID:           id,
		Email:        email,
		PasswordHash: string(hash),
		Role:         "SUPER_ADMIN",
		CreatedAt:    now,
	}, nil
}

// User represents a user in the system
type User struct {
	ID                string     `db:"id" json:"id"`
	Email             string     `db:"email" json:"email"`
	PasswordHash      string     `db:"password_hash" json:"-"`
	Role              string     `db:"role" json:"role"`
	PasswordChangedAt *time.Time `db:"password_changed_at" json:"password_changed_at,omitempty"`
	CreatedAt         time.Time  `db:"created_at" json:"created_at"`
}

// GetUserByEmail finds a user by email
func (r *Registry) GetUserByEmail(email string) (*User, error) {
	var user User
	err := r.db.Get(&user, r.db.Rebind("SELECT * FROM users WHERE email = ?"), email)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &user, err
}

// GetUserByID finds a user by ID
func (r *Registry) GetUserByID(id string) (*User, error) {
	var user User
	err := r.db.Get(&user, r.db.Rebind("SELECT * FROM users WHERE id = ?"), id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &user, err
}

// ListUsers returns all users
func (r *Registry) ListUsers() ([]User, error) {
	var users []User
	err := r.db.Select(&users, "SELECT * FROM users ORDER BY created_at DESC")
	return users, err
}

// CreateUser creates a new user
func (r *Registry) CreateUser(email, password, role string) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	user := &User{
		ID:                generateID(),
		Email:             email,
		PasswordHash:      string(hash),
		Role:              role,
		PasswordChangedAt: &now,
		CreatedAt:         now,
	}

	_, err = r.db.Exec(
		r.db.Rebind("INSERT INTO users (id, email, password_hash, role, password_changed_at, created_at) VALUES (?, ?, ?, ?, ?, ?)"),
		user.ID, user.Email, user.PasswordHash, user.Role, user.PasswordChangedAt, user.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// UpdateUser updates a user's email, role, and optionally password
func (r *Registry) UpdateUser(id, email, role, password string) error {
	if password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		now := time.Now()
		_, err = r.db.Exec(r.db.Rebind("UPDATE users SET email = ?, role = ?, password_hash = ?, password_changed_at = ? WHERE id = ?"), email, role, string(hash), now, id)
		return err
	}
	_, err := r.db.Exec(r.db.Rebind("UPDATE users SET email = ?, role = ? WHERE id = ?"), email, role, id)
	return err
}

// UpdateUserPassword updates a user's password
func (r *Registry) UpdateUserPassword(userID, newPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	now := time.Now()
	_, err = r.db.Exec(r.db.Rebind("UPDATE users SET password_hash = ?, password_changed_at = ? WHERE id = ?"), string(hash), now, userID)
	return err
}

// DeleteUser deletes a user by ID
func (r *Registry) DeleteUser(id string) error {
	_, err := r.db.Exec(r.db.Rebind("DELETE FROM users WHERE id = ?"), id)
	return err
}
