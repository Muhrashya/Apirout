package models

import "time"

// Service represents a logical backend service exposed through the API Gateway.
// It can either point directly to a backend host/port, to an Upstream pool for load balancing,
// or to a Database Connection for Instant API / SQL execution.
type Service struct {
	ID             string    `db:"id" json:"id"`
	Name           string    `db:"name" json:"name"`
	Description    *string   `db:"description" json:"description"`
	Protocol       string    `db:"protocol" json:"protocol"` // "http", "https", "grpc", "tcp", "database"
	Host           string    `db:"host" json:"host"`
	Port           int       `db:"port" json:"port"`
	Path           string    `db:"path" json:"path"`
	ConnectTimeout int       `db:"connect_timeout" json:"connect_timeout"` // milliseconds
	ReadTimeout    int       `db:"read_timeout" json:"read_timeout"`       // milliseconds
	WriteTimeout   int       `db:"write_timeout" json:"write_timeout"`     // milliseconds
	Retries        int       `db:"retries" json:"retries"`
	Enabled        bool      `db:"enabled" json:"enabled"`
	UpstreamID     *string   `db:"upstream_id" json:"upstream_id"`
	ConnectionID   *string   `db:"connection_id" json:"connection_id"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`
}

// ServiceWithDetails includes joined upstream and connection names and route count for UI listing
type ServiceWithDetails struct {
	ID             string    `db:"id" json:"id"`
	Name           string    `db:"name" json:"name"`
	Description    *string   `db:"description" json:"description"`
	Protocol       string    `db:"protocol" json:"protocol"`
	Host           string    `db:"host" json:"host"`
	Port           int       `db:"port" json:"port"`
	Path           string    `db:"path" json:"path"`
	ConnectTimeout int       `db:"connect_timeout" json:"connect_timeout"`
	ReadTimeout    int       `db:"read_timeout" json:"read_timeout"`
	WriteTimeout   int       `db:"write_timeout" json:"write_timeout"`
	Retries        int       `db:"retries" json:"retries"`
	Enabled        bool      `db:"enabled" json:"enabled"`
	UpstreamID     *string   `db:"upstream_id" json:"upstream_id"`
	ConnectionID   *string   `db:"connection_id" json:"connection_id"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`
	UpstreamName   *string   `db:"upstream_name" json:"upstream_name"`
	ConnectionName *string   `db:"connection_name" json:"connection_name"`
	RoutesCount    int       `db:"routes_count" json:"routes_count"`
}

// ToService converts ServiceWithDetails to base Service model
func (s *ServiceWithDetails) ToService() *Service {
	return &Service{
		ID:             s.ID,
		Name:           s.Name,
		Description:    s.Description,
		Protocol:       s.Protocol,
		Host:           s.Host,
		Port:           s.Port,
		Path:           s.Path,
		ConnectTimeout: s.ConnectTimeout,
		ReadTimeout:    s.ReadTimeout,
		WriteTimeout:   s.WriteTimeout,
		Retries:        s.Retries,
		Enabled:        s.Enabled,
		UpstreamID:     s.UpstreamID,
		ConnectionID:   s.ConnectionID,
		CreatedAt:      s.CreatedAt,
		UpdatedAt:      s.UpdatedAt,
	}
}
