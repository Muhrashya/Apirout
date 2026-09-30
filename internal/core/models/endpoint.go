package models

import "time"

// Endpoint (or Route) represents a published REST API endpoint / Gateway Route
type Endpoint struct {
	ID                       string    `db:"id" json:"id"`
	ServiceID                *string   `db:"service_id" json:"service_id"`
	Name                     string    `db:"name" json:"name"`
	GroupID                  *string   `db:"group_id" json:"group_id"`
	Method                   string    `db:"method" json:"method"`
	Path                     string    `db:"path" json:"path"`
	Protocols                string    `db:"protocols" json:"protocols"` // "http,https"
	StripPath                bool      `db:"strip_path" json:"strip_path"`
	PreserveHost             bool      `db:"preserve_host" json:"preserve_host"`
	RegexPriority            int       `db:"regex_priority" json:"regex_priority"`
	HTTPSRedirectStatusCode  int       `db:"https_redirect_status_code" json:"https_redirect_status_code"`
	QueryID                  *string   `db:"query_id" json:"query_id"`
	Enabled                  bool      `db:"enabled" json:"enabled"`
	IsPublic                 bool      `db:"is_public" json:"is_public"`
	RateLimitEnabled         bool      `db:"rate_limit_enabled" json:"rate_limit_enabled"`
	RateLimitRPM             int       `db:"rate_limit_rpm" json:"rate_limit_rpm"`
	RateLimitBurst           int       `db:"rate_limit_burst" json:"rate_limit_burst"`
	DefaultLimit             int       `db:"default_limit" json:"default_limit"`
	MaxLimit                 int       `db:"max_limit" json:"max_limit"`
	ReturnTotalCount         bool      `db:"return_total_count" json:"return_total_count"`
	AllowDynamicFilters      bool      `db:"allow_dynamic_filters" json:"allow_dynamic_filters"`
	AllowedDomains           string    `db:"allowed_domains" json:"allowed_domains"`
	AllowedIPs               string    `db:"allowed_ips" json:"allowed_ips"`
	CorsOrigins              string    `db:"cors_origins" json:"cors_origins"`
	Description              *string   `db:"description" json:"description"`
	CreatedByUserID          *string   `db:"created_by_user_id" json:"created_by_user_id"`
	CreatedAt                time.Time `db:"created_at" json:"created_at"`
	UpdatedAt                time.Time `db:"updated_at" json:"updated_at"`
}

// Route is an alias for Endpoint in Kong architectural terminology
type Route = Endpoint

// EndpointWithMetadata includes group, service and creator info for admin listing
type EndpointWithMetadata struct {
	ID                       string    `db:"id" json:"id"`
	ServiceID                *string   `db:"service_id" json:"service_id"`
	ServiceName              *string   `db:"service_name" json:"service_name"`
	Name                     string    `db:"name" json:"name"`
	GroupID                  *string   `db:"group_id" json:"group_id"`
	GroupName                *string   `db:"group_name" json:"group_name"`
	Method                   string    `db:"method" json:"method"`
	Path                     string    `db:"path" json:"path"`
	Protocols                string    `db:"protocols" json:"protocols"`
	StripPath                bool      `db:"strip_path" json:"strip_path"`
	PreserveHost             bool      `db:"preserve_host" json:"preserve_host"`
	RegexPriority            int       `db:"regex_priority" json:"regex_priority"`
	HTTPSRedirectStatusCode  int       `db:"https_redirect_status_code" json:"https_redirect_status_code"`
	QueryID                  *string   `db:"query_id" json:"query_id"`
	Enabled                  bool      `db:"enabled" json:"enabled"`
	IsPublic                 bool      `db:"is_public" json:"is_public"`
	RateLimitEnabled         bool      `db:"rate_limit_enabled" json:"rate_limit_enabled"`
	RateLimitRPM             int       `db:"rate_limit_rpm" json:"rate_limit_rpm"`
	RateLimitBurst           int       `db:"rate_limit_burst" json:"rate_limit_burst"`
	DefaultLimit             int       `db:"default_limit" json:"default_limit"`
	MaxLimit                 int       `db:"max_limit" json:"max_limit"`
	ReturnTotalCount         bool      `db:"return_total_count" json:"return_total_count"`
	AllowDynamicFilters      bool      `db:"allow_dynamic_filters" json:"allow_dynamic_filters"`
	AllowedDomains           string    `db:"allowed_domains" json:"allowed_domains"`
	AllowedIPs               string    `db:"allowed_ips" json:"allowed_ips"`
	CorsOrigins              string    `db:"cors_origins" json:"cors_origins"`
	Description              *string   `db:"description" json:"description"`
	CreatedByUserID          *string   `db:"created_by_user_id" json:"created_by_user_id"`
	CreatedByEmail           *string   `db:"created_by_email" json:"created_by_email"`
	CreatedAt                time.Time `db:"created_at" json:"created_at"`
	UpdatedAt                time.Time `db:"updated_at" json:"updated_at"`
}

// RouteWithMetadata is an alias for EndpointWithMetadata
type RouteWithMetadata = EndpointWithMetadata
