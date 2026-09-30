package models

import "time"

// Target represents a concrete backend instance belonging to an Upstream pool
type Target struct {
	ID           string    `db:"id" json:"id"`
	UpstreamID   string    `db:"upstream_id" json:"upstream_id"`
	Target       string    `db:"target" json:"target"` // host:port e.g. "10.10.10.11:8080"
	Host         string    `db:"host" json:"host"`
	Port         int       `db:"port" json:"port"`
	Weight       int       `db:"weight" json:"weight"` // 1-1000, default 100
	Enabled      bool      `db:"enabled" json:"enabled"`
	HealthStatus string    `db:"health_status" json:"health_status"` // "healthy", "unhealthy", "checking", "disabled", "unknown"
	Failover     bool      `db:"failover" json:"failover"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}
