package gitops

import (
	"time"
)

// MetadataSnapshot represents the complete declarative state of Apirout
type MetadataSnapshot struct {
	Version     string           `json:"version" yaml:"version"`
	ExportedAt  time.Time        `json:"exported_at" yaml:"exported_at"`
	Groups      []GroupSpec      `json:"groups" yaml:"groups"`
	Services    []ServiceSpec    `json:"services,omitempty" yaml:"services,omitempty"`
	Upstreams   []UpstreamSpec   `json:"upstreams,omitempty" yaml:"upstreams,omitempty"`
	Consumers   []ConsumerSpec   `json:"consumers,omitempty" yaml:"consumers,omitempty"`
	Connections []ConnectionSpec `json:"connections,omitempty" yaml:"connections,omitempty"`
	Queries     []QuerySpec      `json:"queries,omitempty" yaml:"queries,omitempty"`
	Endpoints   []EndpointSpec   `json:"endpoints" yaml:"endpoints"`
}

type GroupSpec struct {
	Name        string  `json:"name" yaml:"name"`
	Description *string `json:"description,omitempty" yaml:"description,omitempty"`
}

type ServiceSpec struct {
	Name           string  `json:"name" yaml:"name"`
	Description    *string `json:"description,omitempty" yaml:"description,omitempty"`
	Protocol       string  `json:"protocol" yaml:"protocol"`
	Host           string  `json:"host" yaml:"host"`
	Port           int     `json:"port" yaml:"port"`
	Path           string  `json:"path" yaml:"path"`
	ConnectTimeout int     `json:"connect_timeout,omitempty" yaml:"connect_timeout,omitempty"`
	ReadTimeout    int     `json:"read_timeout,omitempty" yaml:"read_timeout,omitempty"`
	WriteTimeout   int     `json:"write_timeout,omitempty" yaml:"write_timeout,omitempty"`
	Retries        int     `json:"retries,omitempty" yaml:"retries,omitempty"`
	Enabled        bool    `json:"enabled" yaml:"enabled"`
	UpstreamName   string  `json:"upstream_name,omitempty" yaml:"upstream_name,omitempty"`
	ConnectionName string  `json:"connection_name,omitempty" yaml:"connection_name,omitempty"`
}

type UpstreamSpec struct {
	Name                string       `json:"name" yaml:"name"`
	Algorithm           string       `json:"algorithm" yaml:"algorithm"`
	Slots               int          `json:"slots,omitempty" yaml:"slots,omitempty"`
	Enabled             bool         `json:"enabled" yaml:"enabled"`
	HealthCheckEnabled  bool         `json:"health_check_enabled,omitempty" yaml:"health_check_enabled,omitempty"`
	HealthCheckPath     string       `json:"health_check_path,omitempty" yaml:"health_check_path,omitempty"`
	HealthCheckInterval int          `json:"health_check_interval,omitempty" yaml:"health_check_interval,omitempty"`
	HealthCheckTimeout  int          `json:"health_check_timeout,omitempty" yaml:"health_check_timeout,omitempty"`
	Targets             []TargetSpec `json:"targets,omitempty" yaml:"targets,omitempty"`
}

type TargetSpec struct {
	Target  string `json:"target" yaml:"target"`
	Host    string `json:"host,omitempty" yaml:"host,omitempty"`
	Port    int    `json:"port,omitempty" yaml:"port,omitempty"`
	Weight  int    `json:"weight" yaml:"weight"`
	Enabled bool   `json:"enabled" yaml:"enabled"`
}

type ConsumerSpec struct {
	Username    string  `json:"username" yaml:"username"`
	CustomID    *string `json:"custom_id,omitempty" yaml:"custom_id,omitempty"`
	Description *string `json:"description,omitempty" yaml:"description,omitempty"`
	GroupName   string  `json:"group_name,omitempty" yaml:"group_name,omitempty"`
	Status      string  `json:"status,omitempty" yaml:"status,omitempty"`
}

type ConnectionSpec struct {
	Name      string `json:"name" yaml:"name"`
	DBType    string `json:"db_type" yaml:"db_type"`
	Host      string `json:"host" yaml:"host"`
	Port      int    `json:"port" yaml:"port"`
	DBName    string `json:"dbname" yaml:"dbname"`
	Username  string `json:"username" yaml:"username"`
	SSLMode   string `json:"sslmode" yaml:"sslmode"`
	GroupName string `json:"group_name,omitempty" yaml:"group_name,omitempty"`
}

type QuerySpec struct {
	Name         string  `json:"name" yaml:"name"`
	GroupName    string  `json:"group_name,omitempty" yaml:"group_name,omitempty"`
	Connection   string  `json:"connection" yaml:"connection"`
	SQLText      string  `json:"sql_text" yaml:"sql_text"`
	OrderBy      *string `json:"order_by,omitempty" yaml:"order_by,omitempty"`
	CountSQLText *string `json:"count_sql_text,omitempty" yaml:"count_sql_text,omitempty"`
	DefaultLimit int     `json:"default_limit" yaml:"default_limit"`
}

type EndpointSpec struct {
	Path                string `json:"path" yaml:"path"`
	Name                string `json:"name,omitempty" yaml:"name,omitempty"`
	Method              string `json:"method" yaml:"method"`
	ServiceName         string `json:"service_name,omitempty" yaml:"service_name,omitempty"`
	QueryName           string `json:"query_name,omitempty" yaml:"query_name,omitempty"`
	GroupName           string `json:"group_name,omitempty" yaml:"group_name,omitempty"`
	Enabled             bool   `json:"enabled" yaml:"enabled"`
	IsPublic            bool   `json:"is_public" yaml:"is_public"`
	Protocols           string `json:"protocols,omitempty" yaml:"protocols,omitempty"`
	StripPath           bool   `json:"strip_path,omitempty" yaml:"strip_path,omitempty"`
	PreserveHost        bool   `json:"preserve_host,omitempty" yaml:"preserve_host,omitempty"`
	DefaultLimit        int    `json:"default_limit" yaml:"default_limit"`
	MaxLimit            int    `json:"max_limit" yaml:"max_limit"`
	RateLimitRPM        int    `json:"rate_limit_rpm" yaml:"rate_limit_rpm"`
	RateLimitBurst      int    `json:"rate_limit_burst" yaml:"rate_limit_burst"`
	AllowDynamicFilters bool   `json:"allow_dynamic_filters" yaml:"allow_dynamic_filters"`
	ReturnTotalCount    bool   `json:"return_total_count" yaml:"return_total_count"`
}
