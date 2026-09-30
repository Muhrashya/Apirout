package server

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/apirout/apirout/internal/console/auth"
	"github.com/apirout/apirout/internal/core/httputil"
	"github.com/apirout/apirout/internal/console/registry"

	"github.com/gorilla/mux"
)

// APIClientHandler handles API client management
type APIClientHandler struct {
	registry *registry.Registry
}

// NewAPIClientHandler creates a new API client handler
func NewAPIClientHandler(reg *registry.Registry) *APIClientHandler {
	return &APIClientHandler{registry: reg}
}

// Username validation: allow all printable ASCII characters (no spaces), 3-255 chars
var usernameRegex = regexp.MustCompile(`^[!-~]{3,255}$`)

// List returns all API clients (SUPER_ADMIN) or group-scoped clients (Group Admin)
func (h *APIClientHandler) List(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		httputil.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var clients []registry.APIClientWithCount
	var err error

	if claims.Role == "SUPER_ADMIN" {
		clients, err = h.registry.ListAPIClientsWithCount()
	} else {
		// Get groups where user has manage_endpoints permission
		groupIDs, gErr := h.registry.GetUserGroupsWithPermission(claims.UserID, registry.PermManageEndpoints)
		if gErr != nil {
			httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to check permissions: "+gErr.Error())
			return
		}
		if len(groupIDs) == 0 {
			httputil.WriteJSONData(w, http.StatusOK, []registry.APIClientWithCount{})
			return
		}
		clients, err = h.registry.ListAPIClientsWithCountByGroupIDs(groupIDs)
	}

	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to list API clients: "+err.Error())
		return
	}
	httputil.WriteJSONData(w, http.StatusOK, clients)
}

// CreateAPIClientRequest represents API client creation request
type CreateAPIClientRequest struct {
	GroupID            *string  `json:"group_id"`
	Username           string   `json:"username"`
	Info               string   `json:"info"`
	Password           string   `json:"password"`
	AllowedEndpointIDs []string `json:"allowed_endpoint_ids"`
}

// Create creates a new API client with endpoint allowlist
func (h *APIClientHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		httputil.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req CreateAPIClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Permission Check
	if claims.Role != "SUPER_ADMIN" {
		if req.GroupID == nil || *req.GroupID == "" {
			httputil.WriteJSONError(w, http.StatusBadRequest, "group_id is required for Group Admins")
			return
		}
		// Verify user manages this group
		canManage, err := h.registry.CanManageResource(claims.UserID, *req.GroupID, registry.PermManageEndpoints)
		if err != nil {
			httputil.WriteJSONError(w, http.StatusInternalServerError, "Database error")
			return
		}
		if !canManage {
			httputil.WriteJSONError(w, http.StatusForbidden, "You do not have permission to create API clients in this group")
			return
		}
	} else {
		// SUPER_ADMIN can create global clients (group_id=nil) or group clients
		// Treat empty string as nil
		if req.GroupID != nil && *req.GroupID == "" {
			req.GroupID = nil
		}
	}

	if req.Username == "" {
		httputil.WriteJSONError(w, http.StatusBadRequest, "username is required")
		return
	}

	if req.Password == "" {
		httputil.WriteJSONError(w, http.StatusBadRequest, "password is required")
		return
	}

	// Normalize and validate username format (allow mixed case)
	req.Username = strings.TrimSpace(req.Username)
	if !usernameRegex.MatchString(req.Username) {
		httputil.WriteJSONError(w, http.StatusBadRequest, "invalid username format")
		return
	}

	// Require at least one endpoint
	if len(req.AllowedEndpointIDs) == 0 {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Select at least one endpoint")
		return
	}

	// Validate all endpoint IDs exist and belong to the group (if scoped)
	for _, epID := range req.AllowedEndpointIDs {
		ep, err := h.registry.GetEndpointByID(epID)
		if err != nil {
			httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to validate endpoints")
			return
		}
		if ep == nil {
			httputil.WriteJSONError(w, http.StatusBadRequest, "Endpoint not found: "+epID)
			return
		}

		// If client is scoped to a group, compel endpoints to belong to that group
		// This enforces "only in their groups area" strictly
		if req.GroupID != nil {
			if ep.GroupID == nil || *ep.GroupID != *req.GroupID {
				httputil.WriteJSONError(w, http.StatusBadRequest, "Endpoint "+epID+" does not belong to the selected group")
				return
			}
		}
	}

	// Check if username already exists
	existing, _ := h.registry.GetAPIClientByUsername(req.Username)
	if existing != nil {
		httputil.WriteJSONError(w, http.StatusConflict, "Username already exists")
		return
	}

	client, err := h.registry.CreateAPIClientWithEndpoints(req.GroupID, req.Username, req.Info, req.Password, req.AllowedEndpointIDs)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to create API client: "+err.Error())
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, map[string]interface{}{
		"id":                      client.ID,
		"group_id":                client.GroupID,
		"username":                client.Username,
		"info":                    client.Info,
		"created_at":              client.CreatedAt,
		"allowed_endpoints_count": len(req.AllowedEndpointIDs),
	})
}

// UpdateAPIClientRequest represents API client update request
type UpdateAPIClientRequest struct {
	Info               string   `json:"info"`
	Password           string   `json:"password"`
	AllowedEndpointIDs []string `json:"allowed_endpoint_ids"`
}

// Update updates an API client's allowed endpoints
func (h *APIClientHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		httputil.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	existing, err := h.registry.GetAPIClientByID(id)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to get API client")
		return
	}
	if existing == nil {
		httputil.WriteJSONError(w, http.StatusNotFound, "API client not found")
		return
	}

	// Permission Check
	if claims.Role != "SUPER_ADMIN" {
		if existing.GroupID == nil {
			httputil.WriteJSONError(w, http.StatusForbidden, "You cannot manage global API clients")
			return
		}
		canManage, err := h.registry.CanManageResource(claims.UserID, *existing.GroupID, registry.PermManageEndpoints)
		if err != nil {
			httputil.WriteJSONError(w, http.StatusInternalServerError, "Database error")
			return
		}
		if !canManage {
			httputil.WriteJSONError(w, http.StatusForbidden, "Access denied for this group")
			return
		}
	}

	var req UpdateAPIClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.AllowedEndpointIDs) == 0 {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Select at least one endpoint")
		return
	}

	// Validate endpoints
	for _, epID := range req.AllowedEndpointIDs {
		ep, err := h.registry.GetEndpointByID(epID)
		if err != nil {
			httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to validate endpoints")
			return
		}
		if ep == nil {
			httputil.WriteJSONError(w, http.StatusBadRequest, "Endpoint not found: "+epID)
			return
		}

		// Enforce group scoping if client is scoped
		if existing.GroupID != nil {
			if ep.GroupID == nil || *ep.GroupID != *existing.GroupID {
				httputil.WriteJSONError(w, http.StatusBadRequest, "Endpoint "+epID+" does not belong to the client's group")
				return
			}
		}
	}

	if err := h.registry.UpdateAPIClientWithPassword(id, req.Info, req.Password, req.AllowedEndpointIDs); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to update API client: "+err.Error())
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]string{"message": "API client updated successfully"})
}

// Get returns a single API client with allowed endpoint IDs
func (h *APIClientHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		httputil.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	client, err := h.registry.GetAPIClientByID(id)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to get API client")
		return
	}
	if client == nil {
		httputil.WriteJSONError(w, http.StatusNotFound, "API client not found")
		return
	}

	// Permission Check
	if claims.Role != "SUPER_ADMIN" {
		if client.GroupID == nil {
			httputil.WriteJSONError(w, http.StatusForbidden, "Access denied")
			return
		}
		canManage, err := h.registry.CanManageResource(claims.UserID, *client.GroupID, registry.PermManageEndpoints)
		if err != nil {
			httputil.WriteJSONError(w, http.StatusInternalServerError, "Database error")
			return
		}
		if !canManage {
			httputil.WriteJSONError(w, http.StatusForbidden, "Access denied")
			return
		}
	}

	allowedIDs, err := h.registry.GetAPIClientAllowedEndpointIDs(id)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to get allowed endpoints")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"id":                   client.ID,
		"group_id":             client.GroupID,
		"username":             client.Username,
		"info":                 client.Info,
		"created_at":           client.CreatedAt,
		"allowed_endpoint_ids": allowedIDs,
	})
}

// Delete deletes an API client
func (h *APIClientHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		httputil.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	client, err := h.registry.GetAPIClientByID(id)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to get API client")
		return
	}
	if client == nil {
		httputil.WriteJSONError(w, http.StatusNotFound, "API client not found")
		return
	}

	// Permission Check
	if claims.Role != "SUPER_ADMIN" {
		if client.GroupID == nil {
			httputil.WriteJSONError(w, http.StatusForbidden, "You cannot delete global API clients")
			return
		}
		canManage, err := h.registry.CanManageResource(claims.UserID, *client.GroupID, registry.PermManageEndpoints)
		if err != nil {
			httputil.WriteJSONError(w, http.StatusInternalServerError, "Database error")
			return
		}
		if !canManage {
			httputil.WriteJSONError(w, http.StatusForbidden, "Access denied")
			return
		}
	}

	if err := h.registry.DeleteAPIClient(id); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to delete API client: "+err.Error())
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]string{"message": "API client deleted successfully"})
}
