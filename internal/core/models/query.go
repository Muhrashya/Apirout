package models

import "time"

// Query represents a saved SQL query definition
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
