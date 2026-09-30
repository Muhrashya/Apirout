package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/apirout/apirout/internal/console/registry"
	"github.com/apirout/apirout/internal/core/crypto"
	"github.com/apirout/apirout/internal/core/httputil"
	"github.com/apirout/apirout/internal/core/models"
	"github.com/gorilla/mux"
)

// AuthPluginsHandler handles Inbound Auth Plugins and Service Outbound Auth configurations
type AuthPluginsHandler struct {
	registry      *registry.Registry
	encryptionKey []byte
}

// NewAuthPluginsHandler creates a new AuthPluginsHandler
func NewAuthPluginsHandler(reg *registry.Registry, encryptionKey string) *AuthPluginsHandler {
	return &AuthPluginsHandler{
		registry:      reg,
		encryptionKey: crypto.NormalizeKey(encryptionKey),
	}
}

// List returns all auth plugin configs, optionally filtered by scope and entity_id
func (h *AuthPluginsHandler) List(w http.ResponseWriter, r *http.Request) {
	scope := r.URL.Query().Get("scope")
	entityID := r.URL.Query().Get("entity_id")

	plugins, err := h.registry.ListAuthPlugins(scope, entityID)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to list auth plugins: "+err.Error())
		return
	}
	if plugins == nil {
		plugins = []models.AuthPluginConfigWithNames{}
	}
	httputil.WriteJSONData(w, http.StatusOK, plugins)
}

// Get returns a single auth plugin config by ID
func (h *AuthPluginsHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	plugin, err := h.registry.GetAuthPluginByID(id)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to get auth plugin: "+err.Error())
		return
	}
	if plugin == nil {
		httputil.WriteJSONError(w, http.StatusNotFound, "Auth plugin not found")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, plugin)
}

// CreateAuthPluginRequest represents request payload for creating an auth plugin
type CreateAuthPluginRequest struct {
	PluginName string      `json:"plugin_name"`
	Scope      string      `json:"scope"`
	ServiceID  *string     `json:"service_id"`
	RouteID    *string     `json:"route_id"`
	ConsumerID *string     `json:"consumer_id"`
	Config     interface{} `json:"config"`
	Enabled    *bool       `json:"enabled"`
}

// Create creates a new auth plugin configuration
func (h *AuthPluginsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateAuthPluginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if req.PluginName == "" {
		httputil.WriteJSONError(w, http.StatusBadRequest, "plugin_name is required")
		return
	}

	// Validate plugin name
	valid := false
	for _, p := range models.ValidInboundPlugins() {
		if p == req.PluginName {
			valid = true
			break
		}
	}
	if !valid {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Unsupported plugin_name: "+req.PluginName)
		return
	}

	scope := req.Scope
	if scope == "" {
		if req.RouteID != nil && *req.RouteID != "" {
			scope = "route"
		} else if req.ServiceID != nil && *req.ServiceID != "" {
			scope = "service"
		} else if req.ConsumerID != nil && *req.ConsumerID != "" {
			scope = "consumer"
		} else {
			scope = "global"
		}
	}

	configStr := "{}"
	if req.Config != nil {
		switch c := req.Config.(type) {
		case string:
			if strings.TrimSpace(c) != "" {
				configStr = c
			}
		default:
			bytes, err := json.Marshal(c)
			if err == nil {
				configStr = string(bytes)
			}
		}
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	plugin := &models.AuthPluginConfig{
		PluginName: req.PluginName,
		Scope:      scope,
		ServiceID:  req.ServiceID,
		RouteID:    req.RouteID,
		ConsumerID: req.ConsumerID,
		Config:     configStr,
		Enabled:    enabled,
	}

	if err := h.registry.CreateAuthPlugin(plugin); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to create auth plugin: "+err.Error())
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, plugin)
}

// Update updates an existing auth plugin configuration
func (h *AuthPluginsHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	existing, err := h.registry.GetAuthPluginByID(id)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to retrieve auth plugin: "+err.Error())
		return
	}
	if existing == nil {
		httputil.WriteJSONError(w, http.StatusNotFound, "Auth plugin not found")
		return
	}

	var req CreateAuthPluginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if req.PluginName != "" {
		existing.PluginName = req.PluginName
	}
	if req.Scope != "" {
		existing.Scope = req.Scope
	}
	if req.ServiceID != nil {
		existing.ServiceID = req.ServiceID
	}
	if req.RouteID != nil {
		existing.RouteID = req.RouteID
	}
	if req.ConsumerID != nil {
		existing.ConsumerID = req.ConsumerID
	}
	if req.Config != nil {
		switch c := req.Config.(type) {
		case string:
			if strings.TrimSpace(c) != "" {
				existing.Config = c
			}
		default:
			bytes, err := json.Marshal(c)
			if err == nil {
				existing.Config = string(bytes)
			}
		}
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}

	if err := h.registry.UpdateAuthPlugin(existing); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to update auth plugin: "+err.Error())
		return
	}

	httputil.WriteJSON(w, http.StatusOK, existing)
}

// Delete removes an auth plugin configuration
func (h *AuthPluginsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := h.registry.DeleteAuthPlugin(id); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to delete auth plugin: "+err.Error())
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// --- Outbound Auth Handlers ---

// GetOutboundAuth returns the outbound auth config for a service
func (h *AuthPluginsHandler) GetOutboundAuth(w http.ResponseWriter, r *http.Request) {
	serviceID := mux.Vars(r)["id"]
	auth, err := h.registry.GetOutboundAuth(serviceID)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to get outbound auth: "+err.Error())
		return
	}
	if auth == nil {
		httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"service_id": serviceID,
			"auth_type":  "",
			"enabled":    false,
			"config":     "{}",
		})
		return
	}

	httputil.WriteJSON(w, http.StatusOK, auth)
}

// SetOutboundAuthRequest represents payload for setting outbound auth
type SetOutboundAuthRequest struct {
	AuthType string                 `json:"auth_type"`
	Config   map[string]interface{} `json:"config"`
	Enabled  bool                   `json:"enabled"`
}

// SetOutboundAuth configures or updates upstream authentication for a service
func (h *AuthPluginsHandler) SetOutboundAuth(w http.ResponseWriter, r *http.Request) {
	serviceID := mux.Vars(r)["id"]

	var req SetOutboundAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if req.AuthType == "" {
		httputil.WriteJSONError(w, http.StatusBadRequest, "auth_type is required")
		return
	}

	// Validate auth_type
	valid := false
	for _, t := range models.ValidOutboundTypes() {
		if t == req.AuthType {
			valid = true
			break
		}
	}
	if !valid {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Unsupported outbound auth_type: "+req.AuthType)
		return
	}

	if req.Config == nil {
		req.Config = make(map[string]interface{})
	}

	// Load existing outbound auth to preserve existing encrypted credentials if not modified
	existing, _ := h.registry.GetOutboundAuth(serviceID)
	var existingConfig map[string]interface{}
	if existing != nil && existing.Config != "" {
		_ = json.Unmarshal([]byte(existing.Config), &existingConfig)
	}

	// Encrypt sensitive fields based on auth_type
	switch req.AuthType {
	case models.OutboundBasic:
		if pass, ok := req.Config["password"].(string); ok && pass != "" {
			enc, err := crypto.EncryptPassword(h.encryptionKey, pass)
			if err != nil {
				httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to encrypt password: "+err.Error())
				return
			}
			req.Config["password_encrypted"] = enc
			delete(req.Config, "password")
		} else if existingConfig != nil && existingConfig["password_encrypted"] != nil {
			req.Config["password_encrypted"] = existingConfig["password_encrypted"]
		}

	case models.OutboundBearer:
		if token, ok := req.Config["token"].(string); ok && token != "" {
			enc, err := crypto.EncryptPassword(h.encryptionKey, token)
			if err != nil {
				httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to encrypt token: "+err.Error())
				return
			}
			req.Config["token_encrypted"] = enc
			delete(req.Config, "token")
		} else if existingConfig != nil && existingConfig["token_encrypted"] != nil {
			req.Config["token_encrypted"] = existingConfig["token_encrypted"]
		}

	case models.OutboundAPIKey:
		if key, ok := req.Config["key"].(string); ok && key != "" {
			enc, err := crypto.EncryptPassword(h.encryptionKey, key)
			if err != nil {
				httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to encrypt api key: "+err.Error())
				return
			}
			req.Config["key_encrypted"] = enc
			delete(req.Config, "key")
		} else if existingConfig != nil && existingConfig["key_encrypted"] != nil {
			req.Config["key_encrypted"] = existingConfig["key_encrypted"]
		}

	case models.OutboundAWSSigV4:
		if accessKey, ok := req.Config["access_key"].(string); ok && accessKey != "" {
			enc, err := crypto.EncryptPassword(h.encryptionKey, accessKey)
			if err != nil {
				httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to encrypt access key: "+err.Error())
				return
			}
			req.Config["access_key_encrypted"] = enc
			delete(req.Config, "access_key")
		} else if existingConfig != nil && existingConfig["access_key_encrypted"] != nil {
			req.Config["access_key_encrypted"] = existingConfig["access_key_encrypted"]
		}

		if secretKey, ok := req.Config["secret_key"].(string); ok && secretKey != "" {
			enc, err := crypto.EncryptPassword(h.encryptionKey, secretKey)
			if err != nil {
				httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to encrypt secret key: "+err.Error())
				return
			}
			req.Config["secret_key_encrypted"] = enc
			delete(req.Config, "secret_key")
		} else if existingConfig != nil && existingConfig["secret_key_encrypted"] != nil {
			req.Config["secret_key_encrypted"] = existingConfig["secret_key_encrypted"]
		}

	case models.OutboundOAuth2CC:
		if secret, ok := req.Config["client_secret"].(string); ok && secret != "" {
			enc, err := crypto.EncryptPassword(h.encryptionKey, secret)
			if err != nil {
				httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to encrypt client secret: "+err.Error())
				return
			}
			req.Config["client_secret_encrypted"] = enc
			delete(req.Config, "client_secret")
		} else if existingConfig != nil && existingConfig["client_secret_encrypted"] != nil {
			req.Config["client_secret_encrypted"] = existingConfig["client_secret_encrypted"]
		}
	}

	configBytes, err := json.Marshal(req.Config)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Failed to marshal config: "+err.Error())
		return
	}

	outboundAuth := &models.ServiceOutboundAuth{
		ServiceID: serviceID,
		AuthType:  req.AuthType,
		Config:    string(configBytes),
		Enabled:   req.Enabled,
	}

	if err := h.registry.SetOutboundAuth(outboundAuth); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to save outbound auth: "+err.Error())
		return
	}

	httputil.WriteJSON(w, http.StatusOK, outboundAuth)
}

// DeleteOutboundAuth removes outbound authentication configuration for a service
func (h *AuthPluginsHandler) DeleteOutboundAuth(w http.ResponseWriter, r *http.Request) {
	serviceID := mux.Vars(r)["id"]
	if err := h.registry.DeleteOutboundAuth(serviceID); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to delete outbound auth: "+err.Error())
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
