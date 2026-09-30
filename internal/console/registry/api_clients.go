package registry

import (
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

// APIClient represents an API client for endpoint authentication
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

// apiClientColumns lists all columns mapped to APIClient struct
const apiClientColumns = "id, group_id, username, info, password_hash, created_at"

// ListAPIClients returns all API clients
func (r *Registry) ListAPIClients() ([]APIClient, error) {
	var clients []APIClient
	err := r.db.Select(&clients, "SELECT "+apiClientColumns+" FROM api_clients ORDER BY username")
	return clients, err
}

// ListAPIClientsWithCount returns all API clients with allowed endpoints count
func (r *Registry) ListAPIClientsWithCount() ([]APIClientWithCount, error) {
	var clients []APIClientWithCount
	err := r.db.Select(&clients, `
		SELECT ac.id, ac.group_id, ac.username, ac.info, ac.password_hash, ac.created_at,
			   COALESCE((SELECT COUNT(*) FROM api_client_endpoints WHERE api_client_id = ac.id), 0) as allowed_endpoints_count
		FROM api_clients ac
		ORDER BY ac.username
	`)
	return clients, err
}

// ListAPIClientsWithCountByGroupIDs returns API clients for given groups with count
func (r *Registry) ListAPIClientsWithCountByGroupIDs(groupIDs []string) ([]APIClientWithCount, error) {
	if len(groupIDs) == 0 {
		return []APIClientWithCount{}, nil
	}

	query, args, err := sqlx.In(`
		SELECT ac.id, ac.group_id, ac.username, ac.info, ac.password_hash, ac.created_at,
			   COALESCE((SELECT COUNT(*) FROM api_client_endpoints WHERE api_client_id = ac.id), 0) as allowed_endpoints_count
		FROM api_clients ac
		WHERE ac.group_id IN (?)
		ORDER BY ac.username
	`, groupIDs)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)

	var clients []APIClientWithCount
	err = r.db.Select(&clients, query, args...)
	return clients, err
}

// ListAPIClientsByGroupIDs returns API clients for given groups
func (r *Registry) ListAPIClientsByGroupIDs(groupIDs []string) ([]APIClient, error) {
	if len(groupIDs) == 0 {
		return []APIClient{}, nil
	}

	query, args, err := sqlx.In("SELECT "+apiClientColumns+" FROM api_clients WHERE group_id IN (?) ORDER BY username", groupIDs)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)

	var clients []APIClient
	err = r.db.Select(&clients, query, args...)
	return clients, err
}

// GetAPIClientByUsername returns an API client by username
func (r *Registry) GetAPIClientByUsername(username string) (*APIClient, error) {
	var client APIClient
	err := r.db.Get(&client, r.db.Rebind("SELECT "+apiClientColumns+" FROM api_clients WHERE username = ?"), username)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &client, err
}

// GetAPIClientByID returns an API client by ID
func (r *Registry) GetAPIClientByID(id string) (*APIClient, error) {
	var client APIClient
	err := r.db.Get(&client, r.db.Rebind("SELECT "+apiClientColumns+" FROM api_clients WHERE id = ?"), id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &client, err
}

// CreateAPIClient creates a new API client (legacy, without endpoints)
func (r *Registry) CreateAPIClient(groupID *string, username, info, password string) (*APIClient, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	var infoPtr *string
	if info != "" {
		infoPtr = &info
	}

	id := generateID()
	now := time.Now()
	_, err = r.db.Exec(
		r.db.Rebind("INSERT INTO api_clients (id, group_id, username, info, password_hash, created_at) VALUES (?, ?, ?, ?, ?, ?)"),
		id, groupID, username, infoPtr, string(hash), now,
	)
	if err != nil {
		return nil, err
	}

	return &APIClient{ID: id, GroupID: groupID, Username: username, Info: infoPtr, CreatedAt: now}, nil
}

// CreateAPIClientWithEndpoints creates a new API client with allowed endpoint IDs
func (r *Registry) CreateAPIClientWithEndpoints(groupID *string, username, info, password string, endpointIDs []string) (*APIClient, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	id := generateID()
	now := time.Now()

	var infoPtr *string
	if info != "" {
		infoPtr = &info
	}

	// Start transaction
	tx, err := r.db.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Insert API client
	_, err = tx.Exec(
		r.db.Rebind("INSERT INTO api_clients (id, group_id, username, info, password_hash, created_at) VALUES (?, ?, ?, ?, ?, ?)"),
		id, groupID, username, infoPtr, string(hash), now,
	)
	if err != nil {
		return nil, err
	}

	// Insert endpoint allowlist
	for _, epID := range endpointIDs {
		_, err = tx.Exec(
			r.db.Rebind("INSERT INTO api_client_endpoints (api_client_id, endpoint_id, created_at) VALUES (?, ?, ?)"),
			id, epID, now,
		)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &APIClient{ID: id, Username: username, Info: infoPtr, CreatedAt: now}, nil
}

// UpdateAPIClient updates an API client's info and allowed endpoints
func (r *Registry) UpdateAPIClient(clientID string, info string, endpointIDs []string) error {
	return r.UpdateAPIClientWithPassword(clientID, info, "", endpointIDs)
}

// UpdateAPIClientWithPassword updates an API client's info, optional password, and allowed endpoints
func (r *Registry) UpdateAPIClientWithPassword(clientID string, info string, password string, endpointIDs []string) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var infoPtr *string
	if info != "" {
		infoPtr = &info
	}

	if password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		_, err = tx.Exec(r.db.Rebind("UPDATE api_clients SET info = ?, password_hash = ? WHERE id = ?"), infoPtr, string(hash), clientID)
		if err != nil {
			return err
		}
	} else {
		_, err = tx.Exec(r.db.Rebind("UPDATE api_clients SET info = ? WHERE id = ?"), infoPtr, clientID)
		if err != nil {
			return err
		}
	}

	// Delete existing mappings
	_, err = tx.Exec(r.db.Rebind("DELETE FROM api_client_endpoints WHERE api_client_id = ?"), clientID)
	if err != nil {
		return err
	}

	// Insert new mappings
	now := time.Now()
	for _, epID := range endpointIDs {
		_, err = tx.Exec(
			r.db.Rebind("INSERT INTO api_client_endpoints (api_client_id, endpoint_id, created_at) VALUES (?, ?, ?)"),
			clientID, epID, now,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetAPIClientAllowedEndpointIDs returns the allowed endpoint IDs for a client
func (r *Registry) GetAPIClientAllowedEndpointIDs(clientID string) ([]string, error) {
	var ids []string
	err := r.db.Select(&ids, r.db.Rebind("SELECT endpoint_id FROM api_client_endpoints WHERE api_client_id = ?"), clientID)
	return ids, err
}

// IsAPIClientAllowedForEndpoint checks if a client can access an endpoint
func (r *Registry) IsAPIClientAllowedForEndpoint(clientID, endpointID string) (bool, error) {
	var count int
	err := r.db.Get(&count, r.db.Rebind("SELECT COUNT(*) FROM api_client_endpoints WHERE api_client_id = ? AND endpoint_id = ?"), clientID, endpointID)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// DeleteAPIClient deletes an API client
func (r *Registry) DeleteAPIClient(id string) error {
	_, err := r.db.Exec(r.db.Rebind("DELETE FROM api_clients WHERE id = ?"), id)
	return err
}

// VerifyAPIClientPassword verifies an API client's password
func (r *Registry) VerifyAPIClientPassword(client *APIClient, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(client.PasswordHash), []byte(password))
	return err == nil
}
