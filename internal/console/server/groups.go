package server

import (
	"encoding/json"
	"net/http"

	"github.com/apirout/apirout/internal/console/auth"
	"github.com/apirout/apirout/internal/core/httputil"
	"github.com/apirout/apirout/internal/console/registry"

	"github.com/gorilla/mux"
)

// GroupHandler handles group management
type GroupHandler struct {
	registry *registry.Registry
}

// NewGroupHandler creates a new group handler
func NewGroupHandler(reg *registry.Registry) *GroupHandler {
	return &GroupHandler{registry: reg}
}

// List returns all groups
func (h *GroupHandler) List(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		httputil.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var groups []registry.Group
	var err error

	if claims.Role == "SUPER_ADMIN" {
		groups, err = h.registry.ListGroups()
	} else {
		// Non-superadmin: return only their groups
		groupIDs, gErr := h.registry.GetUserGroupIDs(claims.UserID)
		if gErr != nil {
			httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to get user groups")
			return
		}
		if len(groupIDs) == 0 {
			httputil.WriteJSONData(w, http.StatusOK, []registry.Group{})
			return
		}
		groups, err = h.registry.ListGroupsByIDs(groupIDs)
	}

	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to list groups")
		return
	}
	httputil.WriteJSONData(w, http.StatusOK, groups)
}

// CreateGroupRequest represents group creation request
type CreateGroupRequest struct {
	Name string `json:"name"`
}

// Create creates a new group (SUPER_ADMIN only)
func (h *GroupHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil || claims.Role != "SUPER_ADMIN" {
		httputil.WriteJSONError(w, http.StatusForbidden, "SUPER_ADMIN access required")
		return
	}

	var req CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Name is required")
		return
	}

	group, err := h.registry.CreateGroup(req.Name)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to create group: "+err.Error())
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, group)
}

// Update updates a group (SUPER_ADMIN only)
func (h *GroupHandler) Update(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil || claims.Role != "SUPER_ADMIN" {
		httputil.WriteJSONError(w, http.StatusForbidden, "SUPER_ADMIN access required")
		return
	}

	id := mux.Vars(r)["id"]

	var req CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Name is required")
		return
	}

	if err := h.registry.UpdateGroup(id, req.Name); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to update group: "+err.Error())
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]string{"message": "Group updated successfully"})
}

// Delete deletes a group (SUPER_ADMIN only)
func (h *GroupHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil || claims.Role != "SUPER_ADMIN" {
		httputil.WriteJSONError(w, http.StatusForbidden, "SUPER_ADMIN access required")
		return
	}

	id := mux.Vars(r)["id"]

	if err := h.registry.DeleteGroup(id); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to delete group: "+err.Error())
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]string{"message": "Group deleted successfully"})
}

// ListMembers returns all members of a group
func (h *GroupHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		httputil.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	groupID := mux.Vars(r)["id"]

	// Allow SUPER_ADMIN or any member of the group
	if claims.Role != "SUPER_ADMIN" {
		perm, err := h.registry.GetUserGroupPermission(claims.UserID, groupID)
		if err != nil {
			httputil.WriteJSONError(w, http.StatusInternalServerError, "Database error: "+err.Error())
			return
		}
		if perm == nil {
			httputil.WriteJSONError(w, http.StatusForbidden, "Access denied")
			return
		}
	}

	members, err := h.registry.GetGroupMembers(groupID)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to list members")
		return
	}

	httputil.WriteJSONData(w, http.StatusOK, members)
}

// AddMemberRequest represents add member request
type AddMemberRequest struct {
	UserID      string `json:"user_id"`
	RoleInGroup string `json:"role_in_group"`
}

// AddMember adds a user to a group
func (h *GroupHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		httputil.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	groupID := mux.Vars(r)["id"]

	// Allow SUPER_ADMIN or Group Admin
	if claims.Role != "SUPER_ADMIN" {
		perm, err := h.registry.GetUserGroupPermission(claims.UserID, groupID)
		if err != nil {
			httputil.WriteJSONError(w, http.StatusInternalServerError, "Database error: "+err.Error())
			return
		}
		if perm == nil || !perm.CanManageEndpoints {
			httputil.WriteJSONError(w, http.StatusForbidden, "Group Admin access required")
			return
		}
	}

	var req AddMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.UserID == "" {
		httputil.WriteJSONError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	if req.RoleInGroup == "" {
		req.RoleInGroup = "MEMBER"
	}

	if req.RoleInGroup != "ADMIN" && req.RoleInGroup != "MEMBER" {
		httputil.WriteJSONError(w, http.StatusBadRequest, "role_in_group must be ADMIN or MEMBER")
		return
	}

	// Check if user exists
	user, err := h.registry.GetUserByID(req.UserID)
	if err != nil || user == nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, "User not found")
		return
	}

	// Check if already a member
	existing, err := h.registry.GetUserGroupPermission(req.UserID, groupID)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if existing != nil {
		httputil.WriteJSONError(w, http.StatusConflict, "User is already a member of this group")
		return
	}

	if err := h.registry.AddUserToGroup(req.UserID, groupID, req.RoleInGroup); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to add member: "+err.Error())
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, map[string]string{"message": "Member added successfully"})
}

// UpdateMemberRequest represents update member role request
type UpdateMemberRequest struct {
	RoleInGroup string `json:"role_in_group"`
}

// UpdateMember updates a member's role in a group
func (h *GroupHandler) UpdateMember(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		httputil.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	vars := mux.Vars(r)
	groupID := vars["id"]
	userID := vars["userId"]

	// Allow SUPER_ADMIN or Group Admin
	if claims.Role != "SUPER_ADMIN" {
		perm, err := h.registry.GetUserGroupPermission(claims.UserID, groupID)
		if err != nil {
			httputil.WriteJSONError(w, http.StatusInternalServerError, "Database error: "+err.Error())
			return
		}
		if perm == nil || !perm.CanManageEndpoints {
			httputil.WriteJSONError(w, http.StatusForbidden, "Group Admin access required")
			return
		}
	}

	var req UpdateMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.RoleInGroup != "ADMIN" && req.RoleInGroup != "MEMBER" {
		httputil.WriteJSONError(w, http.StatusBadRequest, "role_in_group must be ADMIN or MEMBER")
		return
	}

	if err := h.registry.UpdateUserGroupRole(userID, groupID, req.RoleInGroup); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to update member: "+err.Error())
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]string{"message": "Member role updated successfully"})
}

// RemoveMember removes a user from a group
func (h *GroupHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		httputil.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	vars := mux.Vars(r)
	groupID := vars["id"]
	userID := vars["userId"]

	// Allow SUPER_ADMIN or Group Admin
	if claims.Role != "SUPER_ADMIN" {
		perm, err := h.registry.GetUserGroupPermission(claims.UserID, groupID)
		if err != nil {
			httputil.WriteJSONError(w, http.StatusInternalServerError, "Database error: "+err.Error())
			return
		}
		if perm == nil || !perm.CanManageEndpoints {
			httputil.WriteJSONError(w, http.StatusForbidden, "Group Admin access required")
			return
		}
	}

	if err := h.registry.RemoveUserFromGroup(userID, groupID); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to remove member: "+err.Error())
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]string{"message": "Member removed successfully"})
}

// ResetMemberPasswordRequest represents reset member password request
type ResetMemberPasswordRequest struct {
	Password string `json:"password"`
}

// ResetMemberPassword resets a member's password
func (h *GroupHandler) ResetMemberPassword(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		httputil.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	vars := mux.Vars(r)
	groupID := vars["id"]
	userID := vars["userId"]

	// Allow SUPER_ADMIN or Group Admin
	if claims.Role != "SUPER_ADMIN" {
		perm, err := h.registry.GetUserGroupPermission(claims.UserID, groupID)
		if err != nil {
			httputil.WriteJSONError(w, http.StatusInternalServerError, "Database error: "+err.Error())
			return
		}
		if perm == nil || !perm.CanManageEndpoints {
			httputil.WriteJSONError(w, http.StatusForbidden, "Group Admin access required")
			return
		}
	}

	var req ResetMemberPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Password == "" {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Password is required")
		return
	}

	// Verify target user is in the group (security check)
	targetPerm, err := h.registry.GetUserGroupPermission(userID, groupID)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if targetPerm == nil {
		httputil.WriteJSONError(w, http.StatusNotFound, "User is not a member of this group")
		return
	}

	if err := h.registry.UpdateUserPassword(userID, req.Password); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to reset password: "+err.Error())
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]string{"message": "Password reset successfully"})
}
