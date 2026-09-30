package registry

import (
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
)

// Query represents a saved query
type Query struct {
	ID            string    `db:"id" json:"id"`
	GroupID       *string   `db:"group_id" json:"group_id"`
	ConnectionID  *string   `db:"connection_id" json:"connection_id"`
	Name          string    `db:"name" json:"name"`
	Description   *string   `db:"description" json:"description"`
	SQLText       string    `db:"sql_text" json:"sql_text"`
	OrderBy       *string   `db:"order_by" json:"order_by"`
	CountSQLText  *string   `db:"count_sql_text" json:"count_sql_text"`
	ParamsSchema  string    `db:"params_schema" json:"params_schema"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
	PreviewLimit  int       `db:"preview_limit" json:"preview_limit"`
	PreviewOffset int       `db:"preview_offset" json:"preview_offset"`
	DefaultLimit  int       `db:"default_limit" json:"default_limit"`
}

// ListQueries returns all queries
func (r *Registry) ListQueries() ([]Query, error) {
	var queries []Query
	err := r.db.Select(&queries, "SELECT * FROM queries ORDER BY name")
	// Handle nulls if any (though we set default)
	return queries, err
}

// ListQueriesByGroupIDs returns queries for given groups
func (r *Registry) ListQueriesByGroupIDs(groupIDs []string) ([]Query, error) {
	if len(groupIDs) == 0 {
		return []Query{}, nil
	}

	query, args, err := sqlx.In("SELECT * FROM queries WHERE group_id IN (?) ORDER BY name", groupIDs)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)

	var queries []Query
	err = r.db.Select(&queries, query, args...)
	return queries, err
}

// GetQueryByID returns a query by ID
func (r *Registry) GetQueryByID(id string) (*Query, error) {
	var query Query
	err := r.db.Get(&query, r.db.Rebind("SELECT * FROM queries WHERE id = ?"), id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &query, err
}

// CreateQuery creates a new query
func (r *Registry) CreateQuery(query *Query) error {
	query.ID = generateID()
	now := time.Now()
	query.CreatedAt = now
	query.UpdatedAt = now

	if query.DefaultLimit <= 0 {
		query.DefaultLimit = 100
	}

	_, err := r.db.Exec(r.db.Rebind(`
		INSERT INTO queries (id, group_id, connection_id, name, sql_text, order_by, count_sql_text, params_schema, created_at, updated_at, preview_limit, preview_offset, default_limit)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
		query.ID, query.GroupID, query.ConnectionID, query.Name, query.SQLText, query.OrderBy, query.CountSQLText, query.ParamsSchema,
		query.CreatedAt, query.UpdatedAt, query.PreviewLimit, query.PreviewOffset, query.DefaultLimit,
	)
	return err
}

// UpdateQuery updates a query
func (r *Registry) UpdateQuery(query *Query) error {
	query.UpdatedAt = time.Now()
	_, err := r.db.Exec(r.db.Rebind(`
		UPDATE queries SET group_id = ?, connection_id = ?, name = ?, sql_text = ?, order_by = ?, count_sql_text = ?, params_schema = ?, updated_at = ?, preview_limit = ?, preview_offset = ?, default_limit = ?
		WHERE id = ?`),
		query.GroupID, query.ConnectionID, query.Name, query.SQLText, query.OrderBy, query.CountSQLText, query.ParamsSchema, query.UpdatedAt, query.PreviewLimit, query.PreviewOffset, query.DefaultLimit, query.ID,
	)
	return err
}

// DeleteQuery deletes a query
func (r *Registry) DeleteQuery(id string) error {
	_, err := r.db.Exec(r.db.Rebind("DELETE FROM queries WHERE id = ?"), id)
	return err
}
