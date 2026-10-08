package registry

import (
	"database/sql"
	"time"

	"github.com/apirout/apirout/internal/core/models"
)

// ListAuthPlugins returns all auth plugin configs, optionally filtered by scope and entity ID
func (r *Registry) ListAuthPlugins(scope, entityID string) ([]models.AuthPluginConfigWithNames, error) {
	var plugins []models.AuthPluginConfigWithNames
	query := `
		SELECT ap.id, ap.plugin_name, ap.scope, ap.service_id, ap.route_id, ap.consumer_id,
		       ap.config, ap.enabled, ap.created_at, ap.updated_at,
		       s.name as service_name,
		       e.name as route_name,
		       ac.username as consumer_name
		FROM auth_plugin_configs ap
		LEFT JOIN services s ON s.id = ap.service_id
		LEFT JOIN endpoints e ON e.id = ap.route_id
		LEFT JOIN api_clients ac ON ac.id = ap.consumer_id
		WHERE 1=1
	`

	args := []interface{}{}
	argIdx := 1

	if scope != "" {
		query += " AND ap.scope = ?"
		args = append(args, scope)
		argIdx++
	}

	if entityID != "" {
		switch scope {
		case "service":
			query += " AND ap.service_id = ?"
		case "route":
			query += " AND ap.route_id = ?"
		case "consumer":
			query += " AND ap.consumer_id = ?"
		default:
			query += " AND (ap.service_id = ? OR ap.route_id = ? OR ap.consumer_id = ?)"
			args = append(args, entityID, entityID)
			argIdx += 2
		}
		args = append(args, entityID)
	}

	query += " ORDER BY ap.created_at DESC"

	err := r.db.Select(&plugins, r.db.Rebind(query), args...)
	if err != nil {
		return nil, err
	}
	return plugins, nil
}

// GetAuthPluginByID returns a single auth plugin config by ID
func (r *Registry) GetAuthPluginByID(id string) (*models.AuthPluginConfig, error) {
	var plugin models.AuthPluginConfig
	err := r.db.Get(&plugin, r.db.Rebind("SELECT * FROM auth_plugin_configs WHERE id = ?"), id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &plugin, nil
}

// CreateAuthPlugin creates a new auth plugin configuration
func (r *Registry) CreateAuthPlugin(plugin *models.AuthPluginConfig) error {
	plugin.ID = generateID()
	now := time.Now()
	plugin.CreatedAt = now
	plugin.UpdatedAt = now

	if plugin.Config == "" {
		plugin.Config = "{}"
	}

	query := `
		INSERT INTO auth_plugin_configs (
			id, plugin_name, scope, service_id, route_id, consumer_id,
			config, enabled, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.Exec(r.db.Rebind(query),
		plugin.ID, plugin.PluginName, plugin.Scope,
		plugin.ServiceID, plugin.RouteID, plugin.ConsumerID,
		plugin.Config, plugin.Enabled,
		plugin.CreatedAt, plugin.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if r.publisher != nil {
		_ = r.publisher.PublishBulkConfigUpdated()
	}
	return nil
}

// UpdateAuthPlugin updates an existing auth plugin configuration
func (r *Registry) UpdateAuthPlugin(plugin *models.AuthPluginConfig) error {
	plugin.UpdatedAt = time.Now()

	query := `
		UPDATE auth_plugin_configs SET
			plugin_name = ?, scope = ?, service_id = ?, route_id = ?, consumer_id = ?,
			config = ?, enabled = ?, updated_at = ?
		WHERE id = ?
	`
	_, err := r.db.Exec(r.db.Rebind(query),
		plugin.PluginName, plugin.Scope,
		plugin.ServiceID, plugin.RouteID, plugin.ConsumerID,
		plugin.Config, plugin.Enabled, plugin.UpdatedAt,
		plugin.ID,
	)
	if err != nil {
		return err
	}
	if r.publisher != nil {
		_ = r.publisher.PublishBulkConfigUpdated()
	}
	return nil
}

// DeleteAuthPlugin deletes an auth plugin by ID
func (r *Registry) DeleteAuthPlugin(id string) error {
	_, err := r.db.Exec(r.db.Rebind("DELETE FROM auth_plugin_configs WHERE id = ?"), id)
	if err != nil {
		return err
	}
	if r.publisher != nil {
		_ = r.publisher.PublishBulkConfigUpdated()
	}
	return nil
}

// GetAuthPluginsByRouteID returns all auth plugins for a specific route
func (r *Registry) GetAuthPluginsByRouteID(routeID string) ([]models.AuthPluginConfig, error) {
	var plugins []models.AuthPluginConfig
	query := `SELECT * FROM auth_plugin_configs WHERE route_id = ? AND enabled = true ORDER BY plugin_name`
	err := r.db.Select(&plugins, r.db.Rebind(query), routeID)
	return plugins, err
}

// GetAuthPluginsByServiceID returns all auth plugins for a specific service
func (r *Registry) GetAuthPluginsByServiceID(serviceID string) ([]models.AuthPluginConfig, error) {
	var plugins []models.AuthPluginConfig
	query := `SELECT * FROM auth_plugin_configs WHERE service_id = ? AND enabled = true ORDER BY plugin_name`
	err := r.db.Select(&plugins, r.db.Rebind(query), serviceID)
	return plugins, err
}

// --- Outbound Auth CRUD ---

// GetOutboundAuth returns the outbound auth config for a service
func (r *Registry) GetOutboundAuth(serviceID string) (*models.ServiceOutboundAuth, error) {
	var auth models.ServiceOutboundAuth
	err := r.db.Get(&auth, r.db.Rebind("SELECT * FROM service_outbound_auth WHERE service_id = ?"), serviceID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &auth, nil
}

// SetOutboundAuth creates or replaces the outbound auth config for a service
func (r *Registry) SetOutboundAuth(auth *models.ServiceOutboundAuth) error {
	now := time.Now()
	auth.UpdatedAt = now

	if auth.Config == "" {
		auth.Config = "{}"
	}

	// Try update first
	updateQuery := `
		UPDATE service_outbound_auth SET
			auth_type = ?, config = ?, enabled = ?, updated_at = ?
		WHERE service_id = ?
	`
	result, err := r.db.Exec(r.db.Rebind(updateQuery),
		auth.AuthType, auth.Config, auth.Enabled, auth.UpdatedAt, auth.ServiceID,
	)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		return nil
	}

	// Insert if no rows were updated
	auth.ID = generateID()
	auth.CreatedAt = now

	insertQuery := `
		INSERT INTO service_outbound_auth (
			id, service_id, auth_type, config, enabled, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err = r.db.Exec(r.db.Rebind(insertQuery),
		auth.ID, auth.ServiceID, auth.AuthType, auth.Config, auth.Enabled,
		auth.CreatedAt, auth.UpdatedAt,
	)
	return err
}

// DeleteOutboundAuth removes outbound auth for a service
func (r *Registry) DeleteOutboundAuth(serviceID string) error {
	_, err := r.db.Exec(r.db.Rebind("DELETE FROM service_outbound_auth WHERE service_id = ?"), serviceID)
	return err
}
