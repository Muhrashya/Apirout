package models

import "time"

// AuditLog represents an audit log entry for both admin and client actions
type AuditLog struct {
	ID           string    `db:"id" json:"id"`
	EventType    string    `db:"event_type" json:"event_type"`
	UserID       *string   `db:"user_id" json:"user_id"`
	APIClientID  *string   `db:"api_client_id" json:"api_client_id"`
	EndpointID   *string   `db:"endpoint_id" json:"endpoint_id"`
	IPAddress    *string   `db:"ip_address" json:"ip_address"`
	Details      string    `db:"details" json:"details"`
	Success      bool      `db:"success" json:"success"`
	DurationMs   *int      `db:"duration_ms" json:"duration_ms"`
	ErrorMessage *string   `db:"error_message" json:"error_message"`
	Timestamp    time.Time `db:"timestamp" json:"timestamp"`
}

// Metric represents latency and status code metric entry
type Metric struct {
	ID         string    `db:"id" json:"id"`
	EndpointID *string   `db:"endpoint_id" json:"endpoint_id"`
	Timestamp  time.Time `db:"timestamp" json:"timestamp"`
	DurationMs int       `db:"duration_ms" json:"duration_ms"`
	StatusCode int       `db:"status_code" json:"status_code"`
	Error      *string   `db:"error" json:"error"`
}
