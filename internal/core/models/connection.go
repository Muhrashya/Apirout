package models

import "time"

// Connection represents a database connection profile
type Connection struct {
	ID                     string    `db:"id" json:"id"`
	GroupID                *string   `db:"group_id" json:"group_id"`
	Name                   string    `db:"name" json:"name"`
	DBType                 string    `db:"db_type" json:"db_type"` // "postgres", "mysql", "mssql"
	Host                   string    `db:"host" json:"host"`
	Port                   int       `db:"port" json:"port"`
	DBName                 string    `db:"db_name" json:"dbname"`
	Username               string    `db:"username" json:"username"`
	PasswordEncrypted      string    `db:"password_encrypted" json:"-"`
	SSLMode                string    `db:"ssl_mode" json:"sslmode"`
	SSHEnabled             bool      `db:"ssh_enabled" json:"ssh_enabled"`
	SSHHost                *string   `db:"ssh_host" json:"ssh_host"`
	SSHPort                int       `db:"ssh_port" json:"ssh_port"`
	SSHUser                *string   `db:"ssh_user" json:"ssh_user"`
	SSHAuthMethod          *string   `db:"ssh_auth_method" json:"ssh_auth_method"`
	SSHPasswordEncrypted   *string   `db:"ssh_password_encrypted" json:"-"`
	SSHPrivateKeyEncrypted *string   `db:"ssh_private_key_encrypted" json:"-"`
	CreatedAt              time.Time `db:"created_at" json:"created_at"`
	UpdatedAt              time.Time `db:"updated_at" json:"updated_at"`
}

// ConnectionHealth represents connection health status and latency
type ConnectionHealth struct {
	ConnectionID     string     `db:"connection_id" json:"connection_id"`
	LastSuccess      bool       `db:"last_success" json:"last_success"`
	LastLatencyMs    *int       `db:"last_latency_ms" json:"last_latency_ms"`
	LastErrorMessage *string    `db:"last_error_message" json:"last_error_message"`
	LastCheckedAt    *time.Time `db:"last_checked_at" json:"last_checked_at"`
}
