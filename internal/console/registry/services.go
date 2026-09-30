package registry

import (
	"database/sql"
	"time"

	"github.com/apirout/apirout/internal/core/models"
)

// ListServices returns all services with their associated upstream name, connection name, and routes count
func (r *Registry) ListServices() ([]models.ServiceWithDetails, error) {
	var services []models.ServiceWithDetails
	query := `
		SELECT s.id, s.name, s.description, s.protocol, s.host, s.port, s.path,
		       s.connect_timeout, s.read_timeout, s.write_timeout, s.retries, s.enabled,
		       s.upstream_id, s.connection_id, s.created_at, s.updated_at,
		       u.name as upstream_name,
		       c.name as connection_name,
		       (SELECT COUNT(*) FROM endpoints e WHERE e.service_id = s.id) as routes_count
		FROM services s
		LEFT JOIN upstreams u ON u.id = s.upstream_id
		LEFT JOIN connections c ON c.id = s.connection_id
		ORDER BY s.name ASC
	`
	err := r.db.Select(&services, query)
	if err != nil {
		return nil, err
	}
	return services, nil
}

// GetServiceByID returns a single service by ID
func (r *Registry) GetServiceByID(id string) (*models.ServiceWithDetails, error) {
	var service models.ServiceWithDetails
	query := `
		SELECT s.id, s.name, s.description, s.protocol, s.host, s.port, s.path,
		       s.connect_timeout, s.read_timeout, s.write_timeout, s.retries, s.enabled,
		       s.upstream_id, s.connection_id, s.created_at, s.updated_at,
		       u.name as upstream_name,
		       c.name as connection_name,
		       (SELECT COUNT(*) FROM endpoints e WHERE e.service_id = s.id) as routes_count
		FROM services s
		LEFT JOIN upstreams u ON u.id = s.upstream_id
		LEFT JOIN connections c ON c.id = s.connection_id
		WHERE s.id = ?
	`
	err := r.db.Get(&service, r.db.Rebind(query), id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &service, nil
}

// CreateService creates a new gateway service
func (r *Registry) CreateService(service *models.Service) error {
	service.ID = generateID()
	now := time.Now()
	service.CreatedAt = now
	service.UpdatedAt = now

	if service.Protocol == "" {
		service.Protocol = "http"
	}
	if service.ConnectTimeout <= 0 {
		service.ConnectTimeout = 60000
	}
	if service.ReadTimeout <= 0 {
		service.ReadTimeout = 60000
	}
	if service.WriteTimeout <= 0 {
		service.WriteTimeout = 60000
	}
	if service.Retries < 0 {
		service.Retries = 5
	}

	query := `
		INSERT INTO services (
			id, name, description, protocol, host, port, path,
			connect_timeout, read_timeout, write_timeout, retries,
			enabled, upstream_id, connection_id, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.Exec(r.db.Rebind(query),
		service.ID, service.Name, service.Description, service.Protocol,
		service.Host, service.Port, service.Path,
		service.ConnectTimeout, service.ReadTimeout, service.WriteTimeout, service.Retries,
		service.Enabled, service.UpstreamID, service.ConnectionID,
		service.CreatedAt, service.UpdatedAt,
	)
	return err
}

// UpdateService updates an existing service
func (r *Registry) UpdateService(service *models.Service) error {
	service.UpdatedAt = time.Now()

	if service.Protocol == "" {
		service.Protocol = "http"
	}
	if service.ConnectTimeout <= 0 {
		service.ConnectTimeout = 60000
	}
	if service.ReadTimeout <= 0 {
		service.ReadTimeout = 60000
	}
	if service.WriteTimeout <= 0 {
		service.WriteTimeout = 60000
	}

	query := `
		UPDATE services SET
			name = ?, description = ?, protocol = ?, host = ?, port = ?, path = ?,
			connect_timeout = ?, read_timeout = ?, write_timeout = ?, retries = ?,
			enabled = ?, upstream_id = ?, connection_id = ?, updated_at = ?
		WHERE id = ?
	`
	_, err := r.db.Exec(r.db.Rebind(query),
		service.Name, service.Description, service.Protocol, service.Host, service.Port, service.Path,
		service.ConnectTimeout, service.ReadTimeout, service.WriteTimeout, service.Retries,
		service.Enabled, service.UpstreamID, service.ConnectionID, service.UpdatedAt,
		service.ID,
	)
	return err
}

// DeleteService deletes a service by ID
func (r *Registry) DeleteService(id string) error {
	// First unlink any endpoints referencing this service
	_, _ = r.db.Exec(r.db.Rebind("UPDATE endpoints SET service_id = NULL WHERE service_id = ?"), id)
	_, err := r.db.Exec(r.db.Rebind("DELETE FROM services WHERE id = ?"), id)
	return err
}
