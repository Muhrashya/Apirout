package models

import "time"

// GatewayDomain represents a registered domain for host-based routing
type GatewayDomain struct {
	ID          string    `db:"id" json:"id"`
	DomainName  string    `db:"domain_name" json:"domain_name"`
	DisplayName string    `db:"display_name" json:"display_name"`
	IsDefault         bool      `db:"is_default" json:"is_default"`
	Environment       string    `db:"environment" json:"environment"`
	DefaultAllowedIPs string    `db:"default_allowed_ips" json:"default_allowed_ips"`
	Description       *string   `db:"description" json:"description"`
	CreatedAt         time.Time `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time `db:"updated_at" json:"updated_at"`
}
