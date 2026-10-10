package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/apirout/apirout/internal/console/auth"
	"github.com/apirout/apirout/internal/core/httputil"
	"github.com/apirout/apirout/internal/console/registry"

	"github.com/gorilla/mux"
)

// UserHandler handles user management
type UserHandler struct {
	registry *registry.Registry
	authSvc  *auth.Service
}

// NewUserHandler creates a new user handler
func NewUserHandler(reg *registry.Registry, authSvc ...*auth.Service) *UserHandler {
	var svc *auth.Service
	if len(authSvc) > 0 {
		svc = authSvc[0]
	}
	return &UserHandler{registry: reg, authSvc: svc}
}

// List returns all users
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil || !auth.IsSuperAdmin(claims.Role) {
		httputil.WriteJSONError(w, http.StatusForbidden, "Only Super Admin can list users")
		return
	}

	users, err := h.registry.ListUsers()
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to list users")
		return
	}
	httputil.WriteJSONData(w, http.StatusOK, users)
}

// CreateUserRequest represents user creation request
type CreateUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// Create creates a new user
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Email and password are required")
		return
	}

	req.Role = strings.ToUpper(strings.TrimSpace(req.Role))
	if req.Role == "" {
		req.Role = auth.RoleDeveloper
	}

	// Validate role
	if !auth.IsValidRole(req.Role) {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid role. Must be SUPER_ADMIN, ADMIN, DEVELOPER, or MAINTAINER")
		return
	}

	user, err := h.registry.CreateUser(req.Email, req.Password, req.Role)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to create user: "+err.Error())
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, user)
}

// UpdateUserRequest represents user update request
type UpdateUserRequest struct {
	Email    string  `json:"email"`
	Role     string  `json:"role"`
	Password *string `json:"password,omitempty"`
}

// Update updates a user
func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Email == "" {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Email is required")
		return
	}

	req.Role = strings.ToUpper(strings.TrimSpace(req.Role))
	if req.Role != "" && !auth.IsValidRole(req.Role) {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid role. Must be SUPER_ADMIN, ADMIN, DEVELOPER, or MAINTAINER")
		return
	}

	targetUser, err := h.registry.GetUserByID(id)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to fetch user: "+err.Error())
		return
	}
	if targetUser == nil {
		httputil.WriteJSONError(w, http.StatusNotFound, "User not found")
		return
	}

	// Guardrail: Block demoting the last active Super Admin
	if targetUser.Role == auth.RoleSuperAdmin && req.Role != "" && req.Role != auth.RoleSuperAdmin {
		users, err := h.registry.ListUsers()
		if err != nil {
			httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to list users: "+err.Error())
			return
		}
		superAdminCount := 0
		for _, u := range users {
			if u.Role == auth.RoleSuperAdmin {
				superAdminCount++
			}
		}
		if superAdminCount <= 1 {
			httputil.WriteJSONError(w, http.StatusBadRequest, "Operation blocked: cannot demote the last active Super Admin account")
			return
		}
	}

	password := ""
	if req.Password != nil {
		password = *req.Password
	}

	if err := h.registry.UpdateUser(id, req.Email, req.Role, password); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to update user: "+err.Error())
		return
	}

	if h.authSvc != nil {
		h.authSvc.InvalidateUserCache(id)
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]string{"message": "User updated successfully"})
}

// Delete deletes a user
func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	// Guardrail 1: Block self-deletion
	claims := auth.GetUserFromContext(r.Context())
	if claims != nil && claims.UserID == id {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Self-deletion is prohibited: you cannot delete your own user account")
		return
	}

	// Guardrail 2: Block deleting the last remaining Super Admin
	targetUser, err := h.registry.GetUserByID(id)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to fetch user: "+err.Error())
		return
	}
	if targetUser == nil {
		httputil.WriteJSONError(w, http.StatusNotFound, "User not found")
		return
	}

	if targetUser.Role == auth.RoleSuperAdmin {
		users, err := h.registry.ListUsers()
		if err != nil {
			httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to list users: "+err.Error())
			return
		}
		superAdminCount := 0
		for _, u := range users {
			if u.Role == auth.RoleSuperAdmin {
				superAdminCount++
			}
		}
		if superAdminCount <= 1 {
			httputil.WriteJSONError(w, http.StatusBadRequest, "Operation blocked: cannot delete the last active Super Admin account")
			return
		}
	}

	if err := h.registry.DeleteUser(id); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to delete user: "+err.Error())
		return
	}

	if h.authSvc != nil {
		h.authSvc.InvalidateUserCache(id)
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]string{"message": "User deleted successfully"})
}
