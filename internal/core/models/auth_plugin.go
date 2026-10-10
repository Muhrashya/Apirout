package models

import "time"

// AuthPluginConfig represents an authentication plugin attached to a Service, Route, or Consumer.
// Follows Kong Gateway plugin architecture where auth plugins can be scoped to different entities.
type AuthPluginConfig struct {
	ID         string    `db:"id" json:"id"`
	PluginName string    `db:"plugin_name" json:"plugin_name"` // "key-auth", "basic-auth", "jwt-auth", "oauth2", "hmac-auth"
	Scope      string    `db:"scope" json:"scope"`             // "service", "route", "consumer", "global"
	ServiceID  *string   `db:"service_id" json:"service_id"`
	RouteID    *string   `db:"route_id" json:"route_id"`
	ConsumerID *string   `db:"consumer_id" json:"consumer_id"`
	Config     string    `db:"config" json:"config"` // JSONB plugin-specific configuration
	Enabled    bool      `db:"enabled" json:"enabled"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`
}

// AuthPluginConfigWithNames includes entity names for admin listing
type AuthPluginConfigWithNames struct {
	AuthPluginConfig
	ServiceName *string `db:"service_name" json:"service_name"`
	RouteName   *string `db:"route_name" json:"route_name"`
	ConsumerName *string `db:"consumer_name" json:"consumer_name"`
}

// ServiceOutboundAuth stores outbound authentication config for a Service.
// When the gateway proxies to an upstream backend, it can inject auth credentials.
type ServiceOutboundAuth struct {
	ID        string    `db:"id" json:"id"`
	ServiceID string    `db:"service_id" json:"service_id"`
	AuthType  string    `db:"auth_type" json:"auth_type"` // "basic", "bearer", "api-key", "aws-sigv4", "oauth2-cc"
	Config    string    `db:"config" json:"config"`       // JSONB encrypted credential config
	Enabled   bool      `db:"enabled" json:"enabled"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// Supported inbound auth plugin names
const (
	PluginKeyAuth   = "key-auth"
	PluginBasicAuth = "basic-auth"
	PluginJWTAuth   = "jwt-auth"
	PluginHMACAuth  = "hmac-auth"
)

// Supported outbound auth types
const (
	OutboundBasic     = "basic"
	OutboundBearer    = "bearer"
	OutboundAPIKey    = "api-key"
	OutboundAWSSigV4  = "aws-sigv4"
	OutboundOAuth2CC  = "oauth2-cc"
)

// ValidInboundPlugins returns all valid inbound auth plugin names
func ValidInboundPlugins() []string {
	return []string{PluginKeyAuth, PluginBasicAuth, PluginJWTAuth, PluginHMACAuth}
}

// ValidOutboundTypes returns all valid outbound auth type names
func ValidOutboundTypes() []string {
	return []string{OutboundBasic, OutboundBearer, OutboundAPIKey, OutboundAWSSigV4, OutboundOAuth2CC}
}
