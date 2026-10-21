package registry

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/jmoiron/sqlx"
)

// Connection represents a database connection profile
type Connection struct {
	ID                     string    `db:"id" json:"id"`
	GroupID                *string   `db:"group_id" json:"group_id"`
	Name                   string    `db:"name" json:"name"`
	DBType                 string    `db:"db_type" json:"db_type"`
	Host                   string    `db:"host" json:"host"`
	Port                   int       `db:"port" json:"port"`
	DBName                 string    `db:"db_name" json:"dbname"`
	Username               string    `db:"username" json:"username"`
	PasswordEncrypted      string    `db:"password_encrypted" json:"-"`
	SSLMode                string    `db:"ssl_mode" json:"sslmode"`
	IsReadOnly             bool      `db:"is_read_only" json:"is_read_only"`
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

// MarshalJSON ensures both snake_case (db_name, ssl_mode) and standard (dbname, sslmode)
// are serialized to prevent field mismatches between frontend API clients and backend models.
func (c Connection) MarshalJSON() ([]byte, error) {
	type Alias Connection
	return json.Marshal(&struct {
		Alias
		DBNameAlias  string `json:"db_name"`
		SSLModeAlias string `json:"ssl_mode"`
	}{
		Alias:        Alias(c),
		DBNameAlias:  c.DBName,
		SSLModeAlias: c.SSLMode,
	})
}


// ConnectionHealth represents connection health status
type ConnectionHealth struct {
	ConnectionID     string     `db:"connection_id" json:"connection_id"`
	LastSuccess      bool       `db:"last_success" json:"last_success"`
	LastLatencyMs    *int       `db:"last_latency_ms" json:"last_latency_ms"`
	LastErrorMessage *string    `db:"last_error_message" json:"last_error_message"`
	LastCheckedAt    *time.Time `db:"last_checked_at" json:"last_checked_at"`
}

// connectionColumns lists all columns mapped to Connection struct
const connectionColumns = `id, group_id, name, db_type, host, port, db_name, username, password_encrypted, ssl_mode,
	is_read_only, ssh_enabled, ssh_host, ssh_port, ssh_user, ssh_auth_method, ssh_password_encrypted, ssh_private_key_encrypted,
	created_at, updated_at`

// ListConnections returns all connections
func (r *Registry) ListConnections() ([]Connection, error) {
	var connections []Connection
	err := r.db.Select(&connections, "SELECT "+connectionColumns+" FROM connections ORDER BY name")
	return connections, err
}

// ListConnectionsByGroupIDs returns connections for given groups
// ListConnectionsByGroupIDs returns connections for given groups AND global connections (group_id IS NULL)
func (r *Registry) ListConnectionsByGroupIDs(groupIDs []string) ([]Connection, error) {
	var query string
	var args []interface{}
	var err error

	if len(groupIDs) == 0 {
		query = "SELECT " + connectionColumns + " FROM connections WHERE group_id IS NULL ORDER BY name"
	} else {
		query, args, err = sqlx.In("SELECT "+connectionColumns+" FROM connections WHERE (group_id IN (?) OR group_id IS NULL) ORDER BY name", groupIDs)
		if err != nil {
			return nil, err
		}
		query = r.db.Rebind(query)
	}

	var connections []Connection
	err = r.db.Select(&connections, query, args...)
	return connections, err
}

// GetConnectionByID returns a connection by ID
func (r *Registry) GetConnectionByID(id string) (*Connection, error) {
	var conn Connection
	err := r.db.Get(&conn, r.db.Rebind("SELECT "+connectionColumns+" FROM connections WHERE id = ?"), id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &conn, err
}

// CreateConnection creates a new connection
func (r *Registry) CreateConnection(conn *Connection) error {
	conn.ID = generateID()
	now := time.Now()
	conn.CreatedAt = now
	conn.UpdatedAt = now

	_, err := r.db.Exec(r.db.Rebind(`
		INSERT INTO connections 
		(id, group_id, name, db_type, host, port, db_name, username, password_encrypted, ssl_mode,
		 is_read_only, ssh_enabled, ssh_host, ssh_port, ssh_user, ssh_auth_method, ssh_password_encrypted, ssh_private_key_encrypted,
		 created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
		conn.ID, conn.GroupID, conn.Name, conn.DBType, conn.Host, conn.Port, conn.DBName,
		conn.Username, conn.PasswordEncrypted, conn.SSLMode,
		conn.IsReadOnly, conn.SSHEnabled, conn.SSHHost, conn.SSHPort, conn.SSHUser, conn.SSHAuthMethod,
		conn.SSHPasswordEncrypted, conn.SSHPrivateKeyEncrypted,
		conn.CreatedAt, conn.UpdatedAt,
	)
	return err
}

// UpdateConnection updates a connection
func (r *Registry) UpdateConnection(conn *Connection) error {
	conn.UpdatedAt = time.Now()
	_, err := r.db.Exec(r.db.Rebind(`
		UPDATE connections SET
		group_id = ?, name = ?, db_type = ?, host = ?, port = ?, db_name = ?, username = ?, 
		password_encrypted = ?, ssl_mode = ?, is_read_only = ?, ssh_enabled = ?, ssh_host = ?, ssh_port = ?, 
		ssh_user = ?, ssh_auth_method = ?, ssh_password_encrypted = ?, ssh_private_key_encrypted = ?,
		updated_at = ?
		WHERE id = ?`),
		conn.GroupID, conn.Name, conn.DBType, conn.Host, conn.Port, conn.DBName,
		conn.Username, conn.PasswordEncrypted, conn.SSLMode, conn.IsReadOnly,
		conn.SSHEnabled, conn.SSHHost, conn.SSHPort, conn.SSHUser, conn.SSHAuthMethod,
		conn.SSHPasswordEncrypted, conn.SSHPrivateKeyEncrypted,
		conn.UpdatedAt, conn.ID,
	)
	if err != nil {
		return err
	}
	if r.publisher != nil {
		_ = r.publisher.PublishConnectionChanged("UPDATE", conn.ID)
	}
	return nil
}

// DeleteConnection deletes a connection
func (r *Registry) DeleteConnection(id string) error {
	_, err := r.db.Exec(r.db.Rebind("DELETE FROM connections WHERE id = ?"), id)
	if err != nil {
		return err
	}
	if r.publisher != nil {
		_ = r.publisher.PublishConnectionChanged("DELETE", id)
	}
	return nil
}

// GetConnectionHealth returns health status for a connection
func (r *Registry) GetConnectionHealth(connectionID string) (*ConnectionHealth, error) {
	var health ConnectionHealth
	err := r.db.Get(&health, r.db.Rebind("SELECT * FROM connection_health WHERE connection_id = ?"), connectionID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &health, err
}

// UpsertConnectionHealth updates or inserts connection health
func (r *Registry) UpsertConnectionHealth(health *ConnectionHealth) error {
	var query string
	if r.dbType == "postgres" {
		query = `
			INSERT INTO connection_health 
			(connection_id, last_success, last_latency_ms, last_error_message, last_checked_at)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT (connection_id) DO UPDATE SET
			last_success = EXCLUDED.last_success,
			last_latency_ms = EXCLUDED.last_latency_ms,
			last_error_message = EXCLUDED.last_error_message,
			last_checked_at = EXCLUDED.last_checked_at`
	} else if r.dbType == "mysql" {
		query = `
			INSERT INTO connection_health 
			(connection_id, last_success, last_latency_ms, last_error_message, last_checked_at)
			VALUES (?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
			last_success = VALUES(last_success),
			last_latency_ms = VALUES(last_latency_ms),
			last_error_message = VALUES(last_error_message),
			last_checked_at = VALUES(last_checked_at)`
	} else {
		query = `
			INSERT OR REPLACE INTO connection_health 
			(connection_id, last_success, last_latency_ms, last_error_message, last_checked_at)
			VALUES (?, ?, ?, ?, ?)`
	}

	_, err := r.db.Exec(r.db.Rebind(query),
		health.ConnectionID, health.LastSuccess, health.LastLatencyMs,
		health.LastErrorMessage, health.LastCheckedAt,
	)
	return err
}

// ListConnectionsWithHealth returns all connections with their health status
// Note: We use MapScan here to handle potential type mismatches (e.g. SQLite int vs bool)
// and checking for legacy column name variations.
func (r *Registry) ListConnectionsWithHealth() ([]map[string]interface{}, error) {
	rows, err := r.db.Queryx(`
		SELECT 
			c.id, c.group_id, c.name, c.db_type, c.host, c.port, c.db_name, c.username, c.password_encrypted, c.ssl_mode,
			c.ssh_enabled, c.ssh_host, c.ssh_port, c.ssh_user, c.ssh_auth_method, c.ssh_password_encrypted, c.ssh_private_key_encrypted,
			c.created_at, c.updated_at,
			ch.last_success, ch.last_latency_ms, ch.last_error_message, ch.last_checked_at
		FROM connections c
		LEFT JOIN connection_health ch ON c.id = ch.connection_id
		ORDER BY c.name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		result := make(map[string]interface{})
		if err := rows.MapScan(result); err != nil {
			return nil, err
		}

		// Fix db_name -> dbname, ssl_mode -> sslmode (keep both for compatibility)
		if val, ok := result["db_name"]; ok {
			result["dbname"] = val
		}
		if val, ok := result["ssl_mode"]; ok {
			result["sslmode"] = val
		}

		// Fix boolean fields (SQLite returns int64)
		fixBool := func(key string) {
			if val, ok := result[key]; ok {
				if i, ok := val.(int64); ok {
					result[key] = i != 0
				}
			}
		}

		fixBool("ssh_enabled")
		fixBool("last_success")

		// SECURITY: Never expose encrypted passwords or keys in list views
		delete(result, "password_encrypted")
		delete(result, "ssh_password_encrypted")
		delete(result, "ssh_private_key_encrypted")

		results = append(results, result)
	}
	return results, nil
}
