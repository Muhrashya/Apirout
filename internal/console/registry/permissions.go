package registry

import "database/sql"

// Permission types for resource management
const (
	PermManageQueries     = "can_manage_queries"
	PermManageEndpoints   = "can_manage_endpoints"
	PermManageAPIClients  = "can_manage_api_clients"
	PermManageConnections = "can_manage_connections"
	PermExecuteEndpoints  = "can_execute_endpoints"
)

// GetUserGroupPermission returns user's permission for a specific group
func (r *Registry) GetUserGroupPermission(userID, groupID string) (*UserGroup, error) {
	var ug UserGroup
	err := r.db.Get(&ug, r.db.Rebind("SELECT * FROM user_groups WHERE user_id = ? AND group_id = ?"), userID, groupID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &ug, err
}

// CanManageResource checks if user can manage a resource type in a group
func (r *Registry) CanManageResource(userID, groupID, permission string) (bool, error) {
	if groupID == "" {
		return false, nil
	}

	ug, err := r.GetUserGroupPermission(userID, groupID)
	if err != nil {
		return false, err
	}
	if ug == nil {
		return false, nil // User not in group
	}

	switch permission {
	case PermManageQueries:
		return ug.CanManageQueries, nil
	case PermManageEndpoints:
		return ug.CanManageEndpoints, nil
	case PermManageAPIClients:
		return ug.CanManageQueries, nil // Reuse queries permission for API clients
	case PermManageConnections:
		return ug.CanManageConnections, nil
	case PermExecuteEndpoints:
		return ug.CanExecuteEndpoints, nil
	default:
		return false, nil
	}
}

// GetUserGroupsWithPermission returns groups where user has specific permission
func (r *Registry) GetUserGroupsWithPermission(userID, permission string) ([]string, error) {
	var groupIDs []string
	var query string

	switch permission {
	case PermManageQueries:
		query = "SELECT group_id FROM user_groups WHERE user_id = ? AND can_manage_queries = TRUE"
	case PermManageEndpoints:
		query = "SELECT group_id FROM user_groups WHERE user_id = ? AND can_manage_endpoints = TRUE"
	case PermManageAPIClients:
		query = "SELECT group_id FROM user_groups WHERE user_id = ? AND can_manage_queries = TRUE"
	default:
		query = "SELECT group_id FROM user_groups WHERE user_id = ?"
	}

	err := r.db.Select(&groupIDs, r.db.Rebind(query), userID)
	return groupIDs, err
}
