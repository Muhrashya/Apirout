package models

import "time"

// APIClient represents an external machine/application consumer
type APIClient struct {
	ID           string    `db:"id" json:"id"`
	GroupID      *string   `db:"group_id" json:"group_id"`
	Username     string    `db:"username" json:"username"`
	Info         *string   `db:"info" json:"info"`
	PasswordHash string    `db:"password_hash" json:"-"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

// APIClientWithCount includes allowed endpoints count for listing
type APIClientWithCount struct {
	ID                    string    `db:"id" json:"id"`
	GroupID               *string   `db:"group_id" json:"group_id"`
	Username              string    `db:"username" json:"username"`
	Info                  *string   `db:"info" json:"info"`
	PasswordHash          string    `db:"password_hash" json:"-"`
	CreatedAt             time.Time `db:"created_at" json:"created_at"`
	AllowedEndpointsCount int       `db:"allowed_endpoints_count" json:"allowed_endpoints_count"`
}
