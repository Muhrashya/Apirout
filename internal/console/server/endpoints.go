package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/apirout/apirout/internal/console/auth"
	"github.com/apirout/apirout/internal/console/publisher"
	"github.com/apirout/apirout/internal/console/registry"
	"github.com/apirout/apirout/internal/core/cache"
	"github.com/apirout/apirout/internal/core/httputil"

	"github.com/gorilla/mux"
)

// EndpointHandler handles route/endpoint management
type EndpointHandler struct {
	registry      *registry.Registry
	endpointCache *cache.EndpointCache
	publisher     *publisher.RedisPublisher
}

// NewEndpointHandler creates a new endpoint handler
func NewEndpointHandler(reg *registry.Registry, epCache *cache.EndpointCache, pub *publisher.RedisPublisher) *EndpointHandler {
	return &EndpointHandler{registry: reg, endpointCache: epCache, publisher: pub}
}

// List returns ALL endpoints with metadata (filtered by user's groups)
func (h *EndpointHandler) List(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		httputil.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	serviceID := r.URL.Query().Get("service_id")
	if serviceID != "" {
		endpoints, err := h.registry.ListEndpointsByServiceID(serviceID)
		if err != nil {
			httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to list endpoints for service")
			return
		}
		httputil.WriteJSONData(w, http.StatusOK, endpoints)
		return
	}

	var endpoints []registry.EndpointWithMetadata
	var err error

	if claims.Role == "SUPER_ADMIN" {
		endpoints, err = h.registry.ListEndpointsWithMetadata()
	} else {
		groupIDs, gErr := h.registry.GetUserGroupIDs(claims.UserID)
		if gErr != nil {
			httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to get user groups")
			return
		}
		endpoints, err = h.registry.ListEndpointsWithMetadataByGroupIDs(groupIDs)
	}

	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to list endpoints")
		return
	}
	httputil.WriteJSONData(w, http.StatusOK, endpoints)
}

// Get returns an endpoint by ID
func (h *EndpointHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	endpoint, err := h.registry.GetEndpointByID(id)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to get endpoint")
		return
	}
	if endpoint == nil {
		httputil.WriteJSONError(w, http.StatusNotFound, "Endpoint not found")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, endpoint)
}

// CreateEndpointRequest represents endpoint/route creation request
type CreateEndpointRequest struct {
	ServiceID                *string `json:"service_id"`
	Name                     string  `json:"name"`
	GroupID                  *string `json:"group_id"`
	Method                   string  `json:"method"`
	Path                     string  `json:"path"`
	Protocols                string  `json:"protocols"`
	StripPath                bool    `json:"strip_path"`
	PreserveHost             bool    `json:"preserve_host"`
	RegexPriority            int     `json:"regex_priority"`
	HTTPSRedirectStatusCode  int     `json:"https_redirect_status_code"`
	QueryID                  *string `json:"query_id"`
	Enabled                  bool    `json:"enabled"`
	IsPublic                 bool    `json:"is_public"`
	RateLimitEnabled         bool    `json:"rate_limit_enabled"`
	RateLimitRPM             int     `json:"rate_limit_rpm"`
	RateLimitBurst           int     `json:"rate_limit_burst"`
	DefaultLimit             int     `json:"default_limit"`
	MaxLimit                 int     `json:"max_limit"`
	ReturnTotalCount         bool    `json:"return_total_count"`
	AllowDynamicFilters      bool    `json:"allow_dynamic_filters"`
	AllowedDomains           string  `json:"allowed_domains"`
	AllowedIPs               string  `json:"allowed_ips"`
	CorsOrigins              string  `json:"cors_origins"`
	Description              *string `json:"description"`
}

// Create creates a new endpoint
func (h *EndpointHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		httputil.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req CreateEndpointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Path == "" {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Path is required")
		return
	}

	// Normalize method and path
	req.Method = strings.ToUpper(strings.TrimSpace(req.Method))
	req.Path = strings.TrimSpace(req.Path)

	if req.Method == "" {
		req.Method = "GET"
	}

	// Ensure path starts with /api/
	if len(req.Path) < 5 || req.Path[:5] != "/api/" {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Path must start with /api/")
		return
	}

	// If no group provided, fallback to first user group or null
	if (req.GroupID == nil || *req.GroupID == "") && claims.Role != "SUPER_ADMIN" {
		groupIDs, _ := h.registry.GetUserGroupIDs(claims.UserID)
		if len(groupIDs) > 0 {
			req.GroupID = &groupIDs[0]
		}
	}

	// Check group permission for non-SUPER_ADMIN
	if claims.Role != "SUPER_ADMIN" && req.GroupID != nil && *req.GroupID != "" {
		canManage, err := h.registry.CanManageResource(claims.UserID, *req.GroupID, registry.PermManageEndpoints)
		if err != nil {
			httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to check permissions")
			return
		}
		if !canManage {
			h.logForbiddenAction(r, claims.UserID, "create_endpoint", req.GroupID)
			httputil.WriteJSONError(w, http.StatusForbidden, "No permission for this group")
			return
		}
	}

	// Check for duplicate endpoint (same method + path)
	existing, err := h.registry.GetEndpointByMethodAndPath(req.Method, req.Path)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if existing != nil {
		httputil.WriteJSONError(w, http.StatusConflict, "Endpoint already exists for this method and path")
		return
	}

	if req.Name == "" {
		req.Name = req.Path
	}
	if req.Protocols == "" {
		req.Protocols = "http,https"
	}
	if req.HTTPSRedirectStatusCode == 0 {
		req.HTTPSRedirectStatusCode = 426
	}

	// Store creator
	endpoint := &registry.Endpoint{
		ServiceID:               req.ServiceID,
		Name:                    req.Name,
		GroupID:                 req.GroupID,
		Method:                  req.Method,
		Path:                    req.Path,
		Protocols:               req.Protocols,
		StripPath:               req.StripPath,
		PreserveHost:            req.PreserveHost,
		RegexPriority:           req.RegexPriority,
		HTTPSRedirectStatusCode: req.HTTPSRedirectStatusCode,
		QueryID:                 req.QueryID,
		Enabled:                 req.Enabled,
		IsPublic:                req.IsPublic,
		RateLimitEnabled:        req.RateLimitEnabled,
		RateLimitRPM:            req.RateLimitRPM,
		RateLimitBurst:          req.RateLimitBurst,
		DefaultLimit:            req.DefaultLimit,
		MaxLimit:                req.MaxLimit,
		ReturnTotalCount:        req.ReturnTotalCount,
		AllowDynamicFilters:     req.AllowDynamicFilters,
		AllowedDomains:          req.AllowedDomains,
		AllowedIPs:              req.AllowedIPs,
		CorsOrigins:             req.CorsOrigins,
		Description:             req.Description,
		CreatedByUserID:         nil,
	}

	if claims.UserID != "" && claims.UserID != "00000000-0000-0000-0000-000000000001" {
		if u, _ := h.registry.GetUserByID(claims.UserID); u != nil {
			endpoint.CreatedByUserID = &u.ID
		}
	}

	if err := h.registry.CreateEndpoint(endpoint); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to create endpoint: "+err.Error())
		return
	}

	// Invalidate cache for new endpoint
	if h.endpointCache != nil {
		h.endpointCache.Invalidate(endpoint.Path)
	}

	if h.publisher != nil {
		_ = h.publisher.PublishEndpointChanged("CREATE", endpoint.ID, endpoint.Path)
	}

	httputil.WriteJSON(w, http.StatusCreated, endpoint)
}

// Update updates an endpoint
func (h *EndpointHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		httputil.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	existing, err := h.registry.GetEndpointByID(id)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to get endpoint")
		return
	}
	if existing == nil {
		httputil.WriteJSONError(w, http.StatusNotFound, "Endpoint not found")
		return
	}

	// Check group permission for non-SUPER_ADMIN
	if claims.Role != "SUPER_ADMIN" && existing.GroupID != nil {
		canManage, err := h.registry.CanManageResource(claims.UserID, *existing.GroupID, registry.PermManageEndpoints)
		if err != nil {
			httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to check permissions")
			return
		}
		if !canManage {
			h.logForbiddenAction(r, claims.UserID, "update_endpoint", existing.GroupID)
			httputil.WriteJSONError(w, http.StatusForbidden, "No permission for this group")
			return
		}
	}

	var req CreateEndpointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Method != "" {
		existing.Method = req.Method
	}
	if req.Path != "" {
		if len(req.Path) < 5 || req.Path[:5] != "/api/" {
			httputil.WriteJSONError(w, http.StatusBadRequest, "Path must start with /api/")
			return
		}
		existing.Path = req.Path
	}
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Protocols != "" {
		existing.Protocols = req.Protocols
	}
	existing.ServiceID = req.ServiceID
	existing.StripPath = req.StripPath
	existing.PreserveHost = req.PreserveHost
	existing.RegexPriority = req.RegexPriority
	if req.HTTPSRedirectStatusCode > 0 {
		existing.HTTPSRedirectStatusCode = req.HTTPSRedirectStatusCode
	}
	if req.GroupID != nil {
		existing.GroupID = req.GroupID
	}
	existing.QueryID = req.QueryID
	existing.Enabled = req.Enabled
	existing.IsPublic = req.IsPublic
	existing.RateLimitEnabled = req.RateLimitEnabled
	existing.RateLimitRPM = req.RateLimitRPM
	existing.RateLimitBurst = req.RateLimitBurst
	existing.DefaultLimit = req.DefaultLimit
	existing.MaxLimit = req.MaxLimit
	existing.ReturnTotalCount = req.ReturnTotalCount
	existing.AllowDynamicFilters = req.AllowDynamicFilters
	if req.AllowedDomains != "" {
		existing.AllowedDomains = req.AllowedDomains
	}
	existing.AllowedIPs = req.AllowedIPs
	if req.CorsOrigins != "" {
		existing.CorsOrigins = req.CorsOrigins
	}
	existing.Description = req.Description

	if err := h.registry.UpdateEndpoint(existing); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to update endpoint: "+err.Error())
		return
	}

	// Invalidate cache
	if h.endpointCache != nil {
		h.endpointCache.Invalidate(existing.Path)
	}

	if h.publisher != nil {
		_ = h.publisher.PublishEndpointChanged("UPDATE", existing.ID, existing.Path)
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]string{"message": "Endpoint updated successfully"})
}

// Delete deletes an endpoint
func (h *EndpointHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		httputil.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	existing, err := h.registry.GetEndpointByID(id)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to get endpoint")
		return
	}
	if existing == nil {
		httputil.WriteJSONError(w, http.StatusNotFound, "Endpoint not found")
		return
	}

	canDelete := h.canDeleteEndpoint(claims, existing)
	if !canDelete {
		h.logForbiddenAction(r, claims.UserID, "delete_endpoint", existing.GroupID)
		httputil.WriteJSONError(w, http.StatusForbidden, "No permission to delete this endpoint")
		return
	}

	if err := h.registry.DeleteEndpoint(id); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to delete endpoint: "+err.Error())
		return
	}

	// Invalidate cache
	if h.endpointCache != nil {
		h.endpointCache.Invalidate(existing.Path)
	}

	if h.publisher != nil {
		_ = h.publisher.PublishEndpointChanged("DELETE", existing.ID, existing.Path)
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]string{"message": "Endpoint deleted successfully"})
}

// canDeleteEndpoint checks if user can delete an endpoint
func (h *EndpointHandler) canDeleteEndpoint(claims *auth.Claims, endpoint *registry.Endpoint) bool {
	if claims.Role == "SUPER_ADMIN" {
		return true
	}
	if endpoint.CreatedByUserID != nil && *endpoint.CreatedByUserID == claims.UserID {
		return true
	}
	if endpoint.GroupID == nil {
		return false
	}
	canManage, err := h.registry.CanManageResource(claims.UserID, *endpoint.GroupID, registry.PermManageEndpoints)
	if err != nil {
		return false
	}
	return canManage
}

// logForbiddenAction logs forbidden action attempt
func (h *EndpointHandler) logForbiddenAction(r *http.Request, userID, action string, groupID *string) {
	clientIP := httputil.GetClientIP(r)
	gid := ""
	if groupID != nil {
		gid = *groupID
	}
	h.registry.CreateAuditLog(&registry.AuditLog{
		EventType: "forbidden_action",
		UserID:    &userID,
		IPAddress: &clientIP,
		Details:   `{"action":"` + action + `","group_id":"` + gid + `"}`,
		Success:   false,
	})
}
