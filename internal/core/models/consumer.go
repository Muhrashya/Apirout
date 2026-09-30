package models

import "time"

// Consumer represents an external API client (web, mobile, partner, M2M)
type Consumer struct {
	ID          string    `db:"id" json:"id"`
	GroupID     *string   `db:"group_id" json:"group_id"` // Consumer Group ID
	Username    string    `db:"username" json:"username"`
	CustomID    *string   `db:"custom_id" json:"custom_id"`
	Description *string   `db:"description" json:"description"`
	Status      string    `db:"status" json:"status"` // "active", "suspended", "revoked"
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

// ConsumerWithDetails provides consumer info with credential count and group name
type ConsumerWithDetails struct {
	ID                    string    `db:"id" json:"id"`
	GroupID               *string   `db:"group_id" json:"group_id"`
	GroupName             *string   `db:"group_name" json:"group_name"`
	Username              string    `db:"username" json:"username"`
	CustomID              *string   `db:"custom_id" json:"custom_id"`
	Description           *string   `db:"description" json:"description"`
	Status                string    `db:"status" json:"status"`
	CreatedAt             time.Time `db:"created_at" json:"created_at"`
	UpdatedAt             time.Time `db:"updated_at" json:"updated_at"`
	AllowedEndpointsCount int       `db:"allowed_endpoints_count" json:"allowed_endpoints_count"`
	CredentialsCount      int       `db:"credentials_count" json:"credentials_count"`
}

// ToConsumer converts ConsumerWithDetails to base Consumer model
func (c *ConsumerWithDetails) ToConsumer() *Consumer {
	return &Consumer{
		ID:          c.ID,
		GroupID:     c.GroupID,
		Username:    c.Username,
		CustomID:    c.CustomID,
		Description: c.Description,
		Status:      c.Status,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}

// ConsumerGroup represents a policy group for external API consumers
type ConsumerGroup struct {
	ID             string    `db:"id" json:"id"`
	Name           string    `db:"name" json:"name"`
	Description    *string   `db:"description" json:"description"`
	RateLimitRPM   int       `db:"rate_limit_rpm" json:"rate_limit_rpm"`
	RateLimitBurst int       `db:"rate_limit_burst" json:"rate_limit_burst"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`
}

// ConsumerGroupWithCount includes consumer count
type ConsumerGroupWithCount struct {
	ID             string    `db:"id" json:"id"`
	Name           string    `db:"name" json:"name"`
	Description    *string   `db:"description" json:"description"`
	RateLimitRPM   int       `db:"rate_limit_rpm" json:"rate_limit_rpm"`
	RateLimitBurst int       `db:"rate_limit_burst" json:"rate_limit_burst"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`
	ConsumersCount int       `db:"consumers_count" json:"consumers_count"`
}

// ConsumerCredential stores separate API keys / secrets for a Consumer.
// Never exposes plaintext secret after creation.
type ConsumerCredential struct {
	ID         string     `db:"id" json:"id"`
	ConsumerID string     `db:"consumer_id" json:"consumer_id"`
	Name       string     `db:"name" json:"name"`
	KeyType    string     `db:"key_type" json:"key_type"` // "api-key", "hmac", "jwt", "oauth2"
	KeyID      string     `db:"key_id" json:"key_id"`     // public identifier / prefix
	SecretHash string     `db:"secret_hash" json:"-"`     // bcrypt hash, never serialized
	MaskedKey  string     `db:"masked_key" json:"masked_key"` // e.g. "ak_live_...9f4a"
	Enabled    bool       `db:"enabled" json:"enabled"`
	ExpiresAt  *time.Time `db:"expires_at" json:"expires_at"`
	LastUsedAt *time.Time `db:"last_used_at" json:"last_used_at"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
}

// ConsumerCredentialCreated is returned ONLY upon creation to reveal the secret once
type ConsumerCredentialCreated struct {
	ConsumerCredential
	KeySecret string `json:"key_secret"`
}

// ConsumerRoutePermission represents the authorization link between consumer and route
type ConsumerRoutePermission struct {
	ConsumerID     string    `db:"consumer_id" json:"consumer_id"`
	RouteID        string    `db:"route_id" json:"route_id"`
	AllowedMethods string    `db:"allowed_methods" json:"allowed_methods"` // "*" or "GET,POST"
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
}
