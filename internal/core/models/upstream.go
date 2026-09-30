package models

import "time"

// Upstream represents a load-balancing pool of Targets
type Upstream struct {
	ID                  string    `db:"id" json:"id"`
	Name                string    `db:"name" json:"name"`
	Algorithm           string    `db:"algorithm" json:"algorithm"` // "round-robin", "least-connections", "consistent-hashing", "latency"
	HashOn              string    `db:"hash_on" json:"hash_on"`     // "none", "ip", "header", "cookie"
	HashFallback        string    `db:"hash_fallback" json:"hash_fallback"`
	HashOnCookie        *string   `db:"hash_on_cookie" json:"hash_on_cookie"`
	Slots               int       `db:"slots" json:"slots"`
	Enabled             bool      `db:"enabled" json:"enabled"`
	HealthCheckEnabled  bool      `db:"health_check_enabled" json:"health_check_enabled"`
	HealthCheckPath     string    `db:"health_check_path" json:"health_check_path"`
	HealthCheckInterval int       `db:"health_check_interval" json:"health_check_interval"` // seconds
	HealthCheckTimeout  int       `db:"health_check_timeout" json:"health_check_timeout"`   // seconds
	HealthyThreshold    int       `db:"healthy_threshold" json:"healthy_threshold"`
	UnhealthyThreshold  int       `db:"unhealthy_threshold" json:"unhealthy_threshold"`
	CreatedAt           time.Time `db:"created_at" json:"created_at"`
	UpdatedAt           time.Time `db:"updated_at" json:"updated_at"`
}

// UpstreamWithDetails includes target counts and targets for UI listing
type UpstreamWithDetails struct {
	ID                  string    `db:"id" json:"id"`
	Name                string    `db:"name" json:"name"`
	Algorithm           string    `db:"algorithm" json:"algorithm"`
	HashOn              string    `db:"hash_on" json:"hash_on"`
	HashFallback        string    `db:"hash_fallback" json:"hash_fallback"`
	HashOnCookie        *string   `db:"hash_on_cookie" json:"hash_on_cookie"`
	Slots               int       `db:"slots" json:"slots"`
	Enabled             bool      `db:"enabled" json:"enabled"`
	HealthCheckEnabled  bool      `db:"health_check_enabled" json:"health_check_enabled"`
	HealthCheckPath     string    `db:"health_check_path" json:"health_check_path"`
	HealthCheckInterval int       `db:"health_check_interval" json:"health_check_interval"`
	HealthCheckTimeout  int       `db:"health_check_timeout" json:"health_check_timeout"`
	HealthyThreshold    int       `db:"healthy_threshold" json:"healthy_threshold"`
	UnhealthyThreshold  int       `db:"unhealthy_threshold" json:"unhealthy_threshold"`
	CreatedAt           time.Time `db:"created_at" json:"created_at"`
	UpdatedAt           time.Time `db:"updated_at" json:"updated_at"`
	TargetsCount        int       `db:"targets_count" json:"targets_count"`
	HealthyCount        int       `db:"healthy_count" json:"healthy_count"`
	Targets             []Target  `json:"targets,omitempty"`
}

// ToUpstream converts UpstreamWithDetails to base Upstream model
func (u *UpstreamWithDetails) ToUpstream() *Upstream {
	return &Upstream{
		ID:                  u.ID,
		Name:                u.Name,
		Algorithm:           u.Algorithm,
		HashOn:              u.HashOn,
		HashFallback:        u.HashFallback,
		HashOnCookie:        u.HashOnCookie,
		Slots:               u.Slots,
		Enabled:             u.Enabled,
		HealthCheckEnabled:  u.HealthCheckEnabled,
		HealthCheckPath:     u.HealthCheckPath,
		HealthCheckInterval: u.HealthCheckInterval,
		HealthCheckTimeout:  u.HealthCheckTimeout,
		HealthyThreshold:    u.HealthyThreshold,
		UnhealthyThreshold:  u.UnhealthyThreshold,
		CreatedAt:           u.CreatedAt,
		UpdatedAt:           u.UpdatedAt,
	}
}
