package registry

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/apirout/apirout/internal/core/models"
)

// ListUpstreams returns all upstreams with target counts
func (r *Registry) ListUpstreams() ([]models.UpstreamWithDetails, error) {
	var upstreams []models.UpstreamWithDetails
	query := `
		SELECT u.id, u.name, u.algorithm, u.hash_on, u.hash_fallback, u.hash_on_cookie,
		       u.slots, u.enabled, u.health_check_enabled, u.health_check_path,
		       u.health_check_interval, u.health_check_timeout, u.healthy_threshold, u.unhealthy_threshold,
		       u.created_at, u.updated_at,
		       (SELECT COUNT(*) FROM targets t WHERE t.upstream_id = u.id) as targets_count,
		       (SELECT COUNT(*) FROM targets t WHERE t.upstream_id = u.id AND t.health_status = 'healthy' AND t.enabled = TRUE) as healthy_count
		FROM upstreams u
		ORDER BY u.name ASC
	`
	err := r.db.Select(&upstreams, query)
	if err != nil {
		return nil, err
	}

	// Fetch targets for each upstream
	for i := range upstreams {
		targets, err := r.ListTargetsByUpstreamID(upstreams[i].ID)
		if err == nil {
			upstreams[i].Targets = targets
		}
	}

	return upstreams, nil
}

// GetUpstreamByID returns an upstream by ID with all its targets
func (r *Registry) GetUpstreamByID(id string) (*models.UpstreamWithDetails, error) {
	var upstream models.UpstreamWithDetails
	query := `
		SELECT u.id, u.name, u.algorithm, u.hash_on, u.hash_fallback, u.hash_on_cookie,
		       u.slots, u.enabled, u.health_check_enabled, u.health_check_path,
		       u.health_check_interval, u.health_check_timeout, u.healthy_threshold, u.unhealthy_threshold,
		       u.created_at, u.updated_at,
		       (SELECT COUNT(*) FROM targets t WHERE t.upstream_id = u.id) as targets_count,
		       (SELECT COUNT(*) FROM targets t WHERE t.upstream_id = u.id AND t.health_status = 'healthy' AND t.enabled = TRUE) as healthy_count
		FROM upstreams u
		WHERE u.id = ?
	`
	err := r.db.Get(&upstream, r.db.Rebind(query), id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	targets, err := r.ListTargetsByUpstreamID(id)
	if err == nil {
		upstream.Targets = targets
	}

	return &upstream, nil
}

// CreateUpstream creates a new upstream pool
func (r *Registry) CreateUpstream(upstream *models.Upstream) error {
	upstream.ID = generateID()
	now := time.Now()
	upstream.CreatedAt = now
	upstream.UpdatedAt = now

	if upstream.Algorithm == "" {
		upstream.Algorithm = "round-robin"
	}
	if upstream.Slots <= 0 {
		upstream.Slots = 10000
	}
	if upstream.HashOn == "" {
		upstream.HashOn = "none"
	}
	if upstream.HashFallback == "" {
		upstream.HashFallback = "none"
	}
	if upstream.HealthCheckPath == "" {
		upstream.HealthCheckPath = "/health"
	}
	if upstream.HealthCheckInterval <= 0 {
		upstream.HealthCheckInterval = 10
	}
	if upstream.HealthCheckTimeout <= 0 {
		upstream.HealthCheckTimeout = 5
	}
	if upstream.HealthyThreshold <= 0 {
		upstream.HealthyThreshold = 2
	}
	if upstream.UnhealthyThreshold <= 0 {
		upstream.UnhealthyThreshold = 3
	}

	query := `
		INSERT INTO upstreams (
			id, name, algorithm, hash_on, hash_fallback, hash_on_cookie,
			slots, enabled, health_check_enabled, health_check_path,
			health_check_interval, health_check_timeout, healthy_threshold, unhealthy_threshold,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.Exec(r.db.Rebind(query),
		upstream.ID, upstream.Name, upstream.Algorithm, upstream.HashOn, upstream.HashFallback, upstream.HashOnCookie,
		upstream.Slots, upstream.Enabled, upstream.HealthCheckEnabled, upstream.HealthCheckPath,
		upstream.HealthCheckInterval, upstream.HealthCheckTimeout, upstream.HealthyThreshold, upstream.UnhealthyThreshold,
		upstream.CreatedAt, upstream.UpdatedAt,
	)
	return err
}

// UpdateUpstream updates an upstream pool
func (r *Registry) UpdateUpstream(upstream *models.Upstream) error {
	upstream.UpdatedAt = time.Now()

	if upstream.Algorithm == "" {
		upstream.Algorithm = "round-robin"
	}
	if upstream.Slots <= 0 {
		upstream.Slots = 10000
	}
	if upstream.HealthCheckPath == "" {
		upstream.HealthCheckPath = "/health"
	}

	query := `
		UPDATE upstreams SET
			name = ?, algorithm = ?, hash_on = ?, hash_fallback = ?, hash_on_cookie = ?,
			slots = ?, enabled = ?, health_check_enabled = ?, health_check_path = ?,
			health_check_interval = ?, health_check_timeout = ?, healthy_threshold = ?, unhealthy_threshold = ?,
			updated_at = ?
		WHERE id = ?
	`
	_, err := r.db.Exec(r.db.Rebind(query),
		upstream.Name, upstream.Algorithm, upstream.HashOn, upstream.HashFallback, upstream.HashOnCookie,
		upstream.Slots, upstream.Enabled, upstream.HealthCheckEnabled, upstream.HealthCheckPath,
		upstream.HealthCheckInterval, upstream.HealthCheckTimeout, upstream.HealthyThreshold, upstream.UnhealthyThreshold,
		upstream.UpdatedAt, upstream.ID,
	)
	return err
}

// DeleteUpstream deletes an upstream pool and associated targets
func (r *Registry) DeleteUpstream(id string) error {
	// First unlink any services using this upstream
	_, _ = r.db.Exec(r.db.Rebind("UPDATE services SET upstream_id = NULL WHERE upstream_id = ?"), id)
	_, _ = r.db.Exec(r.db.Rebind("DELETE FROM targets WHERE upstream_id = ?"), id)
	_, err := r.db.Exec(r.db.Rebind("DELETE FROM upstreams WHERE id = ?"), id)
	return err
}

// ==========================================
// TARGETS MANAGEMENT
// ==========================================

// ListTargetsByUpstreamID returns all targets for an upstream
func (r *Registry) ListTargetsByUpstreamID(upstreamID string) ([]models.Target, error) {
	var targets []models.Target
	err := r.db.Select(&targets, r.db.Rebind("SELECT * FROM targets WHERE upstream_id = ? ORDER BY target ASC"), upstreamID)
	if err != nil {
		return nil, err
	}
	return targets, nil
}

// GetTargetByID returns a single target by ID
func (r *Registry) GetTargetByID(id string) (*models.Target, error) {
	var target models.Target
	err := r.db.Get(&target, r.db.Rebind("SELECT * FROM targets WHERE id = ?"), id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &target, nil
}

// CreateTarget creates a target instance under an upstream
func (r *Registry) CreateTarget(target *models.Target) error {
	target.ID = generateID()
	now := time.Now()
	target.CreatedAt = now
	target.UpdatedAt = now

	// Parse host:port if needed
	if target.Target != "" && (target.Host == "" || target.Port == 0) {
		parts := strings.Split(target.Target, ":")
		target.Host = parts[0]
		if len(parts) > 1 {
			if p, err := strconv.Atoi(parts[1]); err == nil {
				target.Port = p
			}
		}
	}
	if target.Port == 0 {
		target.Port = 80
	}
	if target.Target == "" {
		target.Target = fmt.Sprintf("%s:%d", target.Host, target.Port)
	}
	if target.Weight <= 0 {
		target.Weight = 100
	}
	if target.HealthStatus == "" {
		target.HealthStatus = "healthy"
	}

	query := `
		INSERT INTO targets (
			id, upstream_id, target, host, port, weight,
			enabled, health_status, failover, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.Exec(r.db.Rebind(query),
		target.ID, target.UpstreamID, target.Target, target.Host, target.Port, target.Weight,
		target.Enabled, target.HealthStatus, target.Failover, target.CreatedAt, target.UpdatedAt,
	)
	return err
}

// UpdateTarget updates an existing target
func (r *Registry) UpdateTarget(target *models.Target) error {
	target.UpdatedAt = time.Now()

	if target.Target != "" && (target.Host == "" || target.Port == 0) {
		parts := strings.Split(target.Target, ":")
		target.Host = parts[0]
		if len(parts) > 1 {
			if p, err := strconv.Atoi(parts[1]); err == nil {
				target.Port = p
			}
		}
	}
	if target.Target == "" && target.Host != "" {
		target.Target = fmt.Sprintf("%s:%d", target.Host, target.Port)
	}
	if target.Weight <= 0 {
		target.Weight = 100
	}

	query := `
		UPDATE targets SET
			target = ?, host = ?, port = ?, weight = ?,
			enabled = ?, health_status = ?, failover = ?, updated_at = ?
		WHERE id = ?
	`
	_, err := r.db.Exec(r.db.Rebind(query),
		target.Target, target.Host, target.Port, target.Weight,
		target.Enabled, target.HealthStatus, target.Failover, target.UpdatedAt,
		target.ID,
	)
	return err
}

// UpdateTargetHealthStatus updates the health status of a target
func (r *Registry) UpdateTargetHealthStatus(id string, status string) error {
	query := `UPDATE targets SET health_status = ?, updated_at = ? WHERE id = ?`
	_, err := r.db.Exec(r.db.Rebind(query), status, time.Now(), id)
	return err
}

// UpdateTargetEnabled toggles the enabled state of a target
func (r *Registry) UpdateTargetEnabled(id string, enabled bool) error {
	query := `UPDATE targets SET enabled = ?, updated_at = ? WHERE id = ?`
	_, err := r.db.Exec(r.db.Rebind(query), enabled, time.Now(), id)
	return err
}

// DeleteTarget deletes a target
func (r *Registry) DeleteTarget(id string) error {
	_, err := r.db.Exec(r.db.Rebind("DELETE FROM targets WHERE id = ?"), id)
	return err
}
