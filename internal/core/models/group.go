package models

import "time"

// Group represents a resource group for RBAC isolation
type Group struct {
	ID          string    `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Description *string   `db:"description" json:"description"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

// UserGroup represents a user-group assignment with granular permissions
type UserGroup struct {
	ID                   string `db:"id" json:"id"`
	UserID               string `db:"user_id" json:"user_id"`
	GroupID              string `db:"group_id" json:"group_id"`
	CanManageConnections bool   `db:"can_manage_connections" json:"can_manage_connections"`
	CanManageQueries     bool   `db:"can_manage_queries" json:"can_manage_queries"`
	CanManageEndpoints   bool   `db:"can_manage_endpoints" json:"can_manage_endpoints"`
	CanExecuteEndpoints  bool   `db:"can_execute_endpoints" json:"can_execute_endpoints"`
	CanViewDocs          bool   `db:"can_view_docs" json:"can_view_docs"`
	CanViewDashboard     bool   `db:"can_view_dashboard" json:"can_view_dashboard"`
	CanViewAuditLogs     bool   `db:"can_view_audit_logs" json:"can_view_audit_logs"`
}
