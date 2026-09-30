package registry

import (
	"database/sql"
	"time"

	"github.com/apirout/apirout/internal/core/models"
	"github.com/jmoiron/sqlx"
)

// Endpoint aliases models.Endpoint
type Endpoint = models.Endpoint

// EndpointWithMetadata aliases models.EndpointWithMetadata
type EndpointWithMetadata = models.EndpointWithMetadata

// endpointColumns lists all columns mapped to Endpoint struct
const endpointColumns = `id, service_id, COALESCE(name, path) as name, group_id, method, path,
	COALESCE(protocols, 'http,https') as protocols,
	COALESCE(strip_path, FALSE) as strip_path,
	COALESCE(preserve_host, FALSE) as preserve_host,
	COALESCE(regex_priority, 0) as regex_priority,
	COALESCE(https_redirect_status_code, 426) as https_redirect_status_code,
	query_id, enabled, is_public, 
	rate_limit_enabled, rate_limit_rpm, rate_limit_burst,
	default_limit, max_limit, return_total_count, allow_dynamic_filters,
	allowed_domains, allowed_ips, cors_origins, description,
	created_by_user_id, created_at, updated_at`

// ListEndpoints returns all endpoints (basic)
func (r *Registry) ListEndpoints() ([]Endpoint, error) {
	var endpoints []Endpoint
	err := r.db.Select(&endpoints, "SELECT "+endpointColumns+" FROM endpoints ORDER BY path")
	return endpoints, err
}

// ListEndpointsWithMetadata returns all endpoints with group, service, and creator metadata
func (r *Registry) ListEndpointsWithMetadata() ([]EndpointWithMetadata, error) {
	var endpoints []EndpointWithMetadata
	err := r.db.Select(&endpoints, `
		SELECT e.id, e.service_id, s.name as service_name,
		       COALESCE(e.name, e.path) as name,
		       e.group_id, g.name as group_name, e.method, e.path,
		       COALESCE(e.protocols, 'http,https') as protocols,
		       COALESCE(e.strip_path, FALSE) as strip_path,
		       COALESCE(e.preserve_host, FALSE) as preserve_host,
		       COALESCE(e.regex_priority, 0) as regex_priority,
		       COALESCE(e.https_redirect_status_code, 426) as https_redirect_status_code,
		       e.query_id, e.enabled, e.is_public, 
		       e.rate_limit_enabled, e.rate_limit_rpm, e.rate_limit_burst,
		       e.default_limit, e.max_limit, e.return_total_count, e.allow_dynamic_filters,
		       e.allowed_domains, e.allowed_ips, e.cors_origins, e.description,
		       e.created_by_user_id, u.email as created_by_email,
		       e.created_at, e.updated_at
		FROM endpoints e
		LEFT JOIN services s ON s.id = e.service_id
		LEFT JOIN groups g ON g.id = e.group_id
		LEFT JOIN users u ON u.id = e.created_by_user_id
		ORDER BY e.path
	`)
	return endpoints, err
}

// ListEndpointsByServiceID returns endpoints assigned to a specific service
func (r *Registry) ListEndpointsByServiceID(serviceID string) ([]Endpoint, error) {
	var endpoints []Endpoint
	err := r.db.Select(&endpoints, r.db.Rebind("SELECT "+endpointColumns+" FROM endpoints WHERE service_id = ? ORDER BY path"), serviceID)
	return endpoints, err
}

// ListEndpointsByGroupIDs returns endpoints for given groups
func (r *Registry) ListEndpointsByGroupIDs(groupIDs []string) ([]Endpoint, error) {
	if len(groupIDs) == 0 {
		return []Endpoint{}, nil
	}

	query, args, err := sqlx.In("SELECT "+endpointColumns+" FROM endpoints WHERE group_id IN (?) ORDER BY path", groupIDs)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)

	var endpoints []Endpoint
	err = r.db.Select(&endpoints, query, args...)
	return endpoints, err
}

// ListEndpointsWithMetadataByGroupIDs returns endpoints with metadata filtered by groups
func (r *Registry) ListEndpointsWithMetadataByGroupIDs(groupIDs []string) ([]EndpointWithMetadata, error) {
	if len(groupIDs) == 0 {
		return []EndpointWithMetadata{}, nil
	}

	query, args, err := sqlx.In(`
		SELECT e.id, e.service_id, s.name as service_name,
		       COALESCE(e.name, e.path) as name,
		       e.group_id, g.name as group_name, e.method, e.path,
		       COALESCE(e.protocols, 'http,https') as protocols,
		       COALESCE(e.strip_path, FALSE) as strip_path,
		       COALESCE(e.preserve_host, FALSE) as preserve_host,
		       COALESCE(e.regex_priority, 0) as regex_priority,
		       COALESCE(e.https_redirect_status_code, 426) as https_redirect_status_code,
		       e.query_id, e.enabled, e.is_public, 
		       e.rate_limit_enabled, e.rate_limit_rpm, e.rate_limit_burst,
		       e.default_limit, e.max_limit, e.return_total_count, e.allow_dynamic_filters,
		       e.allowed_domains, e.allowed_ips, e.cors_origins, e.description,
		       e.created_by_user_id, u.email as created_by_email,
		       e.created_at, e.updated_at
		FROM endpoints e
		LEFT JOIN services s ON s.id = e.service_id
		LEFT JOIN groups g ON g.id = e.group_id
		LEFT JOIN users u ON u.id = e.created_by_user_id
		WHERE e.group_id IN (?)
		ORDER BY e.path
	`, groupIDs)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)

	var endpoints []EndpointWithMetadata
	err = r.db.Select(&endpoints, query, args...)
	return endpoints, err
}

// GetEndpointByID returns an endpoint by ID
func (r *Registry) GetEndpointByID(id string) (*Endpoint, error) {
	var endpoint Endpoint
	err := r.db.Get(&endpoint, r.db.Rebind("SELECT "+endpointColumns+" FROM endpoints WHERE id = ?"), id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &endpoint, err
}

// GetEndpointByPath returns an endpoint by path
func (r *Registry) GetEndpointByPath(path string) (*Endpoint, error) {
	var endpoint Endpoint
	err := r.db.Get(&endpoint, r.db.Rebind("SELECT "+endpointColumns+" FROM endpoints WHERE path = ?"), path)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &endpoint, err
}

// GetEndpointByMethodAndPath returns an endpoint by method and path (for duplicate check)
func (r *Registry) GetEndpointByMethodAndPath(method, path string) (*Endpoint, error) {
	var endpoint Endpoint
	err := r.db.Get(&endpoint, r.db.Rebind("SELECT "+endpointColumns+" FROM endpoints WHERE method = ? AND path = ?"), method, path)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &endpoint, err
}

// CreateEndpoint creates a new endpoint
func (r *Registry) CreateEndpoint(endpoint *Endpoint) error {
	endpoint.ID = generateID()
	now := time.Now()
	endpoint.CreatedAt = now
	endpoint.UpdatedAt = now

	if endpoint.Name == "" {
		endpoint.Name = endpoint.Path
	}
	if endpoint.Protocols == "" {
		endpoint.Protocols = "http,https"
	}
	if endpoint.AllowedDomains == "" {
		endpoint.AllowedDomains = "*"
	}
	if endpoint.AllowedIPs == "" {
		endpoint.AllowedIPs = "*"
	}
	if endpoint.CorsOrigins == "" {
		endpoint.CorsOrigins = "*"
	}
	if endpoint.Method == "" {
		endpoint.Method = "GET"
	}
	if endpoint.HTTPSRedirectStatusCode == 0 {
		endpoint.HTTPSRedirectStatusCode = 426
	}

	_, err := r.db.Exec(r.db.Rebind(`
		INSERT INTO endpoints (
			id, service_id, name, group_id, method, path, protocols, strip_path, preserve_host, regex_priority, https_redirect_status_code,
			query_id, enabled, is_public, 
			rate_limit_enabled, rate_limit_rpm, rate_limit_burst, 
			default_limit, max_limit, return_total_count, allow_dynamic_filters,
			allowed_domains, allowed_ips, cors_origins, description,
			created_by_user_id, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
		endpoint.ID, endpoint.ServiceID, endpoint.Name, endpoint.GroupID, endpoint.Method, endpoint.Path,
		endpoint.Protocols, endpoint.StripPath, endpoint.PreserveHost, endpoint.RegexPriority, endpoint.HTTPSRedirectStatusCode,
		endpoint.QueryID, endpoint.Enabled, endpoint.IsPublic,
		endpoint.RateLimitEnabled, endpoint.RateLimitRPM, endpoint.RateLimitBurst,
		endpoint.DefaultLimit, endpoint.MaxLimit, endpoint.ReturnTotalCount, endpoint.AllowDynamicFilters,
		endpoint.AllowedDomains, endpoint.AllowedIPs, endpoint.CorsOrigins, endpoint.Description,
		endpoint.CreatedByUserID, endpoint.CreatedAt, endpoint.UpdatedAt,
	)
	return err
}

// UpdateEndpoint updates an endpoint
func (r *Registry) UpdateEndpoint(endpoint *Endpoint) error {
	endpoint.UpdatedAt = time.Now()
	if endpoint.Name == "" {
		endpoint.Name = endpoint.Path
	}
	if endpoint.Protocols == "" {
		endpoint.Protocols = "http,https"
	}
	if endpoint.AllowedDomains == "" {
		endpoint.AllowedDomains = "*"
	}
	if endpoint.AllowedIPs == "" {
		endpoint.AllowedIPs = "*"
	}
	if endpoint.CorsOrigins == "" {
		endpoint.CorsOrigins = "*"
	}
	_, err := r.db.Exec(r.db.Rebind(`
		UPDATE endpoints SET
			service_id = ?, name = ?, group_id = ?, method = ?, path = ?,
			protocols = ?, strip_path = ?, preserve_host = ?, regex_priority = ?, https_redirect_status_code = ?,
			query_id = ?, enabled = ?, is_public = ?, 
			rate_limit_enabled = ?, rate_limit_rpm = ?, rate_limit_burst = ?, 
			default_limit = ?, max_limit = ?, return_total_count = ?, allow_dynamic_filters = ?,
			allowed_domains = ?, allowed_ips = ?, cors_origins = ?, description = ?,
			updated_at = ?
		WHERE id = ?`),
		endpoint.ServiceID, endpoint.Name, endpoint.GroupID, endpoint.Method, endpoint.Path,
		endpoint.Protocols, endpoint.StripPath, endpoint.PreserveHost, endpoint.RegexPriority, endpoint.HTTPSRedirectStatusCode,
		endpoint.QueryID, endpoint.Enabled, endpoint.IsPublic,
		endpoint.RateLimitEnabled, endpoint.RateLimitRPM, endpoint.RateLimitBurst,
		endpoint.DefaultLimit, endpoint.MaxLimit, endpoint.ReturnTotalCount, endpoint.AllowDynamicFilters,
		endpoint.AllowedDomains, endpoint.AllowedIPs, endpoint.CorsOrigins, endpoint.Description,
		endpoint.UpdatedAt, endpoint.ID,
	)
	return err
}

// DeleteEndpoint deletes an endpoint
func (r *Registry) DeleteEndpoint(id string) error {
	_, err := r.db.Exec(r.db.Rebind("DELETE FROM endpoints WHERE id = ?"), id)
	return err
}

// ListEnabledEndpoints returns all enabled endpoints
func (r *Registry) ListEnabledEndpoints() ([]Endpoint, error) {
	var endpoints []Endpoint
	err := r.db.Select(&endpoints, r.db.Rebind("SELECT "+endpointColumns+" FROM endpoints WHERE enabled = ? ORDER BY path"), true)
	return endpoints, err
}
