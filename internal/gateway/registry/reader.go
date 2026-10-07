package registry

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/apirout/apirout/internal/core/models"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// Reader provides read-mostly data access for the Data Plane Gateway
type Reader struct {
	db     *sqlx.DB
	dbType string
}

// NewReader connects to the shared config store (PostgreSQL)
func NewReader(dbType, host, port, dbname, user, password, sslmode string) (*Reader, error) {
	var dsn string
	switch dbType {
	case "postgres", "postgresql":
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			host, port, user, password, dbname, sslmode)
	default:
		return nil, fmt.Errorf("unsupported registry db type for data plane: %s", dbType)
	}

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("connect to config database: %w", err)
	}

	maxOpen := getEnvInt("DB_MAX_OPEN_CONNS", 25)
	maxIdle := getEnvInt("DB_MAX_IDLE_CONNS", 5)
	maxLifetime := getEnvDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute)
	maxIdleTime := getEnvDuration("DB_CONN_MAX_IDLE_TIME", 10*time.Minute)

	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(maxLifetime)
	db.SetConnMaxIdleTime(maxIdleTime)

	return &Reader{db: db.Unsafe(), dbType: dbType}, nil
}

// Close closes the database connection
func (r *Reader) Close() error {
	return r.db.Close()
}

// GetEndpointByPath retrieves active endpoint metadata by URL path (exact or longest prefix)
func (r *Reader) GetEndpointByPath(path string) (*models.Endpoint, error) {
	var endpoint models.Endpoint
	// 1. Try exact match first
	err := r.db.Get(&endpoint, "SELECT * FROM endpoints WHERE path = $1", path)
	if err == nil {
		return &endpoint, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	// 2. Fall back to longest prefix match (e.g. /api/v1/users/123 matches /api/v1/users)
	prefixQuery := `
		SELECT * FROM endpoints 
		WHERE ($1 LIKE path || '/%' OR $1 LIKE path || '?%')
		ORDER BY LENGTH(path) DESC, regex_priority DESC 
		LIMIT 1
	`
	err = r.db.Get(&endpoint, prefixQuery, path)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &endpoint, err
}

// GetAllActiveEndpoints retrieves all enabled endpoints for compiling the in-memory Radix router
func (r *Reader) GetAllActiveEndpoints() ([]models.Endpoint, error) {
	var endpoints []models.Endpoint
	err := r.db.Select(&endpoints, "SELECT * FROM endpoints WHERE enabled = TRUE ORDER BY LENGTH(path) DESC")
	if err != nil {
		return nil, err
	}
	return endpoints, nil
}

// GetQueryByID retrieves query definition by ID
func (r *Reader) GetQueryByID(id string) (*models.Query, error) {
	var query models.Query
	err := r.db.Get(&query, "SELECT * FROM queries WHERE id = $1", id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &query, err
}

// GetConnectionByID retrieves connection profile by ID
func (r *Reader) GetConnectionByID(id string) (*models.Connection, error) {
	var conn models.Connection
	err := r.db.Get(&conn, "SELECT * FROM connections WHERE id = $1", id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &conn, err
}

// GetAPIClientByUsername finds an external API client by username
func (r *Reader) GetAPIClientByUsername(username string) (*models.APIClient, error) {
	var client models.APIClient
	err := r.db.Get(&client, "SELECT * FROM api_clients WHERE username = $1", username)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &client, err
}

// IsAPIClientAllowedForEndpoint checks client allowlist permissions
func (r *Reader) IsAPIClientAllowedForEndpoint(apiClientID, endpointID string) (bool, error) {
	var count int
	// If no restrictions defined in api_client_endpoints, client has access to group endpoints
	query := `
		SELECT COUNT(*) FROM api_client_endpoints ace
		WHERE ace.api_client_id = $1 AND ace.endpoint_id = $2
	`
	err := r.db.Get(&count, query, apiClientID, endpointID)
	if err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}

	// Check if client has no specific endpoint restrictions, meaning group-level access
	var totalRestrictions int
	err = r.db.Get(&totalRestrictions, "SELECT COUNT(*) FROM api_client_endpoints WHERE api_client_id = $1", apiClientID)
	if err != nil {
		return false, err
	}

	// If no specific restrictions, check if in same group
	if totalRestrictions == 0 {
		var sameGroup bool
		groupQuery := `
			SELECT EXISTS(
				SELECT 1 FROM api_clients c
				JOIN endpoints e ON c.group_id = e.group_id
				WHERE c.id = $1 AND e.id = $2
			)
		`
		err = r.db.Get(&sameGroup, groupQuery, apiClientID, endpointID)
		return sameGroup, err
	}

	return false, nil
}

// GetServiceByID retrieves a logical Gateway service by ID
func (r *Reader) GetServiceByID(id string) (*models.Service, error) {
	var service models.Service
	err := r.db.Get(&service, "SELECT * FROM services WHERE id = $1", id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &service, err
}

// GetUpstreamByID retrieves an upstream pool definition by ID
func (r *Reader) GetUpstreamByID(id string) (*models.Upstream, error) {
	var upstream models.Upstream
	err := r.db.Get(&upstream, "SELECT * FROM upstreams WHERE id = $1", id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &upstream, err
}

// GetHealthyTargetsByUpstreamID retrieves active and healthy targets for an upstream
func (r *Reader) GetHealthyTargetsByUpstreamID(upstreamID string) ([]models.Target, error) {
	var targets []models.Target
	query := `
		SELECT * FROM targets
		WHERE upstream_id = $1 AND enabled = TRUE AND health_status = 'healthy'
		ORDER BY weight DESC, target ASC
	`
	err := r.db.Select(&targets, query, upstreamID)
	if err != nil {
		return nil, err
	}

	// If no strictly healthy targets found, check if any enabled targets exist (failover / checking)
	if len(targets) == 0 {
		fallbackQuery := `SELECT * FROM targets WHERE upstream_id = $1 AND enabled = TRUE ORDER BY weight DESC`
		_ = r.db.Select(&targets, fallbackQuery, upstreamID)
	}

	return targets, nil
}

// ValidateAPIKey checks raw API key against consumer_credentials and returns Consumer
func (r *Reader) ValidateAPIKey(rawKey string) (*models.Consumer, *models.ConsumerCredential, error) {
	if rawKey == "" {
		return nil, nil, fmt.Errorf("empty API key")
	}

	parts := strings.Split(rawKey, "_")
	var keyID string
	if len(parts) >= 3 && parts[0] == "ak" {
		keyID = fmt.Sprintf("%s_%s_%s", parts[0], parts[1], parts[2])
	}

	if keyID == "" {
		return nil, nil, fmt.Errorf("invalid API key format")
	}

	var creds []models.ConsumerCredential
	query := `SELECT * FROM consumer_credentials WHERE key_id = $1 AND enabled = TRUE`
	if err := r.db.Select(&creds, query, keyID); err != nil || len(creds) == 0 {
		return nil, nil, fmt.Errorf("invalid API key")
	}

	for _, cred := range creds {
		if cred.ExpiresAt != nil && cred.ExpiresAt.Before(time.Now()) {
			continue
		}

		// bcrypt compare
		if err := bcrypt.CompareHashAndPassword([]byte(cred.SecretHash), []byte(rawKey)); err == nil {
			var consumer models.Consumer
			err := r.db.Get(&consumer, "SELECT id, group_id, username, custom_id, info as description, COALESCE(status, 'active') as status, created_at, updated_at FROM api_clients WHERE id = $1", cred.ConsumerID)
			if err != nil {
				return nil, nil, err
			}

			if consumer.Status == "suspended" || consumer.Status == "revoked" {
				return nil, nil, fmt.Errorf("consumer account is %s", consumer.Status)
			}

			return &consumer, &cred, nil
		}
	}

	return nil, nil, fmt.Errorf("invalid API key")
}

// CreateAuditLog records an API request log asynchronously
func (r *Reader) CreateAuditLog(log *models.AuditLog) error {
	query := `
		INSERT INTO audit_logs (id, event_type, user_id, api_client_id, endpoint_id, ip_address, details, success, duration_ms, error_message, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.db.Exec(query,
		log.ID, log.EventType, log.UserID, log.APIClientID, log.EndpointID, log.IPAddress, log.Details,
		log.Success, log.DurationMs, log.ErrorMessage, log.Timestamp,
	)
	return err
}

// GetConsumerGroupByID retrieves consumer group policy metadata
func (r *Reader) GetConsumerGroupByID(id string) (*models.ConsumerGroup, error) {
	var g models.ConsumerGroup
	err := r.db.Get(&g, "SELECT * FROM consumer_groups WHERE id = $1", id)
	if err != nil {
		return nil, err
	}
	return &g, nil
}

// GetAuthPluginsForRoute returns enabled auth plugins attached to a specific route
func (r *Reader) GetAuthPluginsForRoute(routeID string) ([]models.AuthPluginConfig, error) {
	var plugins []models.AuthPluginConfig
	query := `SELECT * FROM auth_plugin_configs WHERE route_id = $1 AND enabled = TRUE ORDER BY plugin_name`
	err := r.db.Select(&plugins, query, routeID)
	if err != nil {
		return nil, err
	}
	return plugins, nil
}

// GetAuthPluginsForService returns enabled auth plugins attached to a specific service
func (r *Reader) GetAuthPluginsForService(serviceID string) ([]models.AuthPluginConfig, error) {
	var plugins []models.AuthPluginConfig
	query := `SELECT * FROM auth_plugin_configs WHERE service_id = $1 AND enabled = TRUE ORDER BY plugin_name`
	err := r.db.Select(&plugins, query, serviceID)
	if err != nil {
		return nil, err
	}
	return plugins, nil
}

// GetGlobalAuthPlugins returns enabled global auth plugins (not attached to any specific entity)
func (r *Reader) GetGlobalAuthPlugins() ([]models.AuthPluginConfig, error) {
	var plugins []models.AuthPluginConfig
	query := `SELECT * FROM auth_plugin_configs WHERE scope = 'global' AND enabled = TRUE ORDER BY plugin_name`
	err := r.db.Select(&plugins, query)
	if err != nil {
		return nil, err
	}
	return plugins, nil
}

// GetOutboundAuth returns the outbound auth config for a service
func (r *Reader) GetOutboundAuth(serviceID string) (*models.ServiceOutboundAuth, error) {
	var auth models.ServiceOutboundAuth
	err := r.db.Get(&auth, `SELECT * FROM service_outbound_auth WHERE service_id = $1 AND enabled = TRUE`, serviceID)
	if err != nil {
		return nil, nil // No outbound auth configured
	}
	return &auth, nil
}

// GetGatewayDomains returns all configured gateway domains
func (r *Reader) GetGatewayDomains() ([]models.GatewayDomain, error) {
	var domains []models.GatewayDomain
	query := `SELECT id, domain_name, display_name, is_default, environment, COALESCE(default_allowed_ips, '*') as default_allowed_ips, description, created_at, updated_at FROM gateway_domains ORDER BY is_default DESC, domain_name ASC`
	err := r.db.Select(&domains, query)
	if err != nil {
		return nil, err
	}
	return domains, nil
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
