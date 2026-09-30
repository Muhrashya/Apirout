package registry

import (
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
)

// Group represents a resource group
type Group struct {
	ID          string    `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Description *string   `db:"description" json:"description"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

// UserGroup represents a user-group assignment with permissions
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

// ListGroups returns all groups
func (r *Registry) ListGroups() ([]Group, error) {
	var groups []Group
	err := r.db.Select(&groups, "SELECT * FROM groups ORDER BY name")
	return groups, err
}

// ListGroupsByIDs returns groups for given IDs
func (r *Registry) ListGroupsByIDs(groupIDs []string) ([]Group, error) {
	if len(groupIDs) == 0 {
		return []Group{}, nil
	}

	query, args, err := sqlx.In("SELECT * FROM groups WHERE id IN (?) ORDER BY name", groupIDs)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)

	var groups []Group
	err = r.db.Select(&groups, query, args...)
	return groups, err
}

// GetGroupByID returns a group by ID
func (r *Registry) GetGroupByID(id string) (*Group, error) {
	var group Group
	err := r.db.Get(&group, r.db.Rebind("SELECT * FROM groups WHERE id = ?"), id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &group, err
}

// CreateGroup creates a new group
func (r *Registry) CreateGroup(name string) (*Group, error) {
	id := generateID()
	now := time.Now()
	_, err := r.db.Exec(
		r.db.Rebind("INSERT INTO groups (id, name, created_at) VALUES (?, ?, ?)"),
		id, name, now,
	)
	if err != nil {
		return nil, err
	}
	return &Group{ID: id, Name: name, CreatedAt: now}, nil
}

// UpdateGroup updates a group
func (r *Registry) UpdateGroup(id, name string) error {
	_, err := r.db.Exec(r.db.Rebind("UPDATE groups SET name = ? WHERE id = ?"), name, id)
	return err
}

// DeleteGroup deletes a group
func (r *Registry) DeleteGroup(id string) error {
	_, err := r.db.Exec(r.db.Rebind("DELETE FROM groups WHERE id = ?"), id)
	return err
}

// AssignUserToGroup assigns a user to a group
func (r *Registry) AssignUserToGroup(userID, groupID string, permissions UserGroup) error {
	id := generateID()
	var query string
	if r.dbType == "postgres" {
		query = `
			INSERT INTO user_groups 
			(id, user_id, group_id, can_manage_connections, can_manage_queries, can_manage_endpoints, 
			 can_execute_endpoints, can_view_docs, can_view_dashboard, can_view_audit_logs)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (user_id, group_id) DO UPDATE SET
			can_manage_connections = EXCLUDED.can_manage_connections,
			can_manage_queries = EXCLUDED.can_manage_queries,
			can_manage_endpoints = EXCLUDED.can_manage_endpoints,
			can_execute_endpoints = EXCLUDED.can_execute_endpoints,
			can_view_docs = EXCLUDED.can_view_docs,
			can_view_dashboard = EXCLUDED.can_view_dashboard,
			can_view_audit_logs = EXCLUDED.can_view_audit_logs`
	} else if r.dbType == "mysql" {
		query = `
			INSERT INTO user_groups 
			(id, user_id, group_id, can_manage_connections, can_manage_queries, can_manage_endpoints, 
			 can_execute_endpoints, can_view_docs, can_view_dashboard, can_view_audit_logs)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
			can_manage_connections = VALUES(can_manage_connections),
			can_manage_queries = VALUES(can_manage_queries),
			can_manage_endpoints = VALUES(can_manage_endpoints),
			can_execute_endpoints = VALUES(can_execute_endpoints),
			can_view_docs = VALUES(can_view_docs),
			can_view_dashboard = VALUES(can_view_dashboard),
			can_view_audit_logs = VALUES(can_view_audit_logs)`
	} else {
		query = `
			INSERT OR REPLACE INTO user_groups 
			(id, user_id, group_id, can_manage_connections, can_manage_queries, can_manage_endpoints, 
			 can_execute_endpoints, can_view_docs, can_view_dashboard, can_view_audit_logs)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	}

	_, err := r.db.Exec(r.db.Rebind(query),
		id, userID, groupID,
		permissions.CanManageConnections, permissions.CanManageQueries, permissions.CanManageEndpoints,
		permissions.CanExecuteEndpoints, permissions.CanViewDocs, permissions.CanViewDashboard, permissions.CanViewAuditLogs,
	)
	return err
}

// RemoveUserFromGroup removes a user from a group
func (r *Registry) RemoveUserFromGroup(userID, groupID string) error {
	_, err := r.db.Exec(r.db.Rebind("DELETE FROM user_groups WHERE user_id = ? AND group_id = ?"), userID, groupID)
	return err
}

// GetUserGroups returns all groups a user belongs to
func (r *Registry) GetUserGroups(userID string) ([]UserGroup, error) {
	var userGroups []UserGroup
	err := r.db.Select(&userGroups, r.db.Rebind("SELECT * FROM user_groups WHERE user_id = ?"), userID)
	return userGroups, err
}

// GetUserGroupIDs returns all group IDs a user belongs to
func (r *Registry) GetUserGroupIDs(userID string) ([]string, error) {
	var groupIDs []string
	err := r.db.Select(&groupIDs, r.db.Rebind("SELECT group_id FROM user_groups WHERE user_id = ?"), userID)
	return groupIDs, err
}

// GroupMember represents a user's membership in a group
type GroupMember struct {
	UserID      string `db:"user_id" json:"user_id"`
	Email       string `db:"email" json:"email"`
	RoleInGroup string `db:"role_in_group" json:"role_in_group"`
}

// GetGroupMembers returns all members of a group
func (r *Registry) GetGroupMembers(groupID string) ([]GroupMember, error) {
	var members []GroupMember
	// Use boolean compatible check for role
	err := r.db.Select(&members, r.db.Rebind(`
		SELECT ug.user_id, u.email, 
			CASE WHEN ug.can_manage_endpoints THEN 'ADMIN' ELSE 'MEMBER' END as role_in_group
		FROM user_groups ug
		JOIN users u ON u.id = ug.user_id
		WHERE ug.group_id = ?
		ORDER BY u.email
	`), groupID)
	return members, err
}

// AddUserToGroup adds a user to a group with a role
func (r *Registry) AddUserToGroup(userID, groupID, roleInGroup string) error {
	id := generateID()
	isAdmin := roleInGroup == "ADMIN"
	// Use TRUE literals instead of 1 for boolean columns
	_, err := r.db.Exec(r.db.Rebind(`
		INSERT INTO user_groups 
		(id, user_id, group_id, can_manage_connections, can_manage_queries, can_manage_endpoints, 
		 can_execute_endpoints, can_view_docs, can_view_dashboard, can_view_audit_logs)
		VALUES (?, ?, ?, ?, ?, ?, TRUE, TRUE, TRUE, ?)`),
		id, userID, groupID,
		isAdmin, isAdmin, isAdmin, isAdmin,
	)
	return err
}

// UpdateUserGroupRole updates a user's role in a group
func (r *Registry) UpdateUserGroupRole(userID, groupID, roleInGroup string) error {
	isAdmin := roleInGroup == "ADMIN"
	_, err := r.db.Exec(r.db.Rebind(`
		UPDATE user_groups SET 
			can_manage_connections = ?, 
			can_manage_queries = ?, 
			can_manage_endpoints = ?,
			can_view_audit_logs = ?
		WHERE user_id = ? AND group_id = ?`),
		isAdmin, isAdmin, isAdmin, isAdmin, userID, groupID,
	)
	return err
}
