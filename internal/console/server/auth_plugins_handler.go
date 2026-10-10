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
	for i := range plugins {
		plugins[i].Config = RedactInboundAuthConfig(plugins[i].Config)
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
	redacted := *plugin
	redacted.Config = RedactInboundAuthConfig(plugin.Config)
	httputil.WriteJSON(w, http.StatusOK, redacted)
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

	// Guard against duplicate plugin of same type on the entity
	var entityID string
	if req.RouteID != nil && *req.RouteID != "" {
		entityID = *req.RouteID
	} else if req.ServiceID != nil && *req.ServiceID != "" {
		entityID = *req.ServiceID
	} else if req.ConsumerID != nil && *req.ConsumerID != "" {
		entityID = *req.ConsumerID
	}

	if entityID != "" || scope == "global" {
		existingPlugins, err := h.registry.ListAuthPlugins(scope, entityID)
		if err == nil {
			for _, ep := range existingPlugins {
				if ep.PluginName == req.PluginName {
					httputil.WriteJSONError(w, http.StatusBadRequest, "a "+req.PluginName+" plugin is already attached to this "+scope)
					return
				}
			}
		}
	}

	encConfig, err := EncryptInboundAuthConfig(configStr, h.encryptionKey)
	if err == nil {
		plugin.Config = encConfig
	}

	if err := h.registry.CreateAuthPlugin(plugin); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to create auth plugin: "+err.Error())
		return
	}

	respPlugin := *plugin
	respPlugin.Config = RedactInboundAuthConfig(plugin.Config)
	httputil.WriteJSON(w, http.StatusCreated, respPlugin)
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
		configStr := existing.Config
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

		var newMap map[string]interface{}
		var oldMap map[string]interface{}
		_ = json.Unmarshal([]byte(configStr), &newMap)
		_ = json.Unmarshal([]byte(existing.Config), &oldMap)

		if newMap != nil && oldMap != nil {
			for _, encKey := range []string{"password_encrypted", "secret_encrypted", "secret_key_encrypted", "api_key_encrypted"} {
				if val, exists := newMap[encKey]; !exists || val == true {
					if oldVal, ok := oldMap[encKey]; ok {
						newMap[encKey] = oldVal
					}
				}
			}
			newBytes, _ := json.Marshal(newMap)
			configStr = string(newBytes)
		}

		encConfig, err := EncryptInboundAuthConfig(configStr, h.encryptionKey)
		if err == nil {
			existing.Config = encConfig
		} else {
			existing.Config = configStr
		}
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}

	// Guard against duplicate plugin of same type on the entity
	if req.PluginName != "" {
		var entityID string
		if existing.RouteID != nil && *existing.RouteID != "" {
			entityID = *existing.RouteID
		} else if existing.ServiceID != nil && *existing.ServiceID != "" {
			entityID = *existing.ServiceID
		} else if existing.ConsumerID != nil && *existing.ConsumerID != "" {
			entityID = *existing.ConsumerID
		}

		if entityID != "" || existing.Scope == "global" {
			existingPlugins, err := h.registry.ListAuthPlugins(existing.Scope, entityID)
			if err == nil {
				for _, ep := range existingPlugins {
					if ep.ID != existing.ID && ep.PluginName == existing.PluginName {
						httputil.WriteJSONError(w, http.StatusBadRequest, "a "+existing.PluginName+" plugin is already attached to this "+existing.Scope)
						return
					}
				}
			}
		}
	}

	if err := h.registry.UpdateAuthPlugin(existing); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to update auth plugin: "+err.Error())
		return
	}

	respPlugin := *existing
	respPlugin.Config = RedactInboundAuthConfig(existing.Config)
	httputil.WriteJSON(w, http.StatusOK, respPlugin)
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

	redacted := *auth
	redacted.Config = RedactOutboundAuthConfig(auth.Config)
	httputil.WriteJSON(w, http.StatusOK, redacted)
}

// RedactOutboundAuthConfig parses the config JSON and replaces encrypted ciphertext fields
// with boolean status indicators, preventing ciphertext strings from ever leaving the server.
func RedactOutboundAuthConfig(configStr string) string {
	if configStr == "" {
		return "{}"
	}
	var cfgMap map[string]interface{}
	if err := json.Unmarshal([]byte(configStr), &cfgMap); err != nil {
		return "{}"
	}

	sensitiveFields := []struct {
		encKey    string
		statusKey string
	}{
		{"password_encrypted", "password_configured"},
		{"token_encrypted", "token_configured"},
		{"key_encrypted", "key_configured"},
		{"access_key_encrypted", "access_key_configured"},
		{"secret_key_encrypted", "secret_key_configured"},
		{"client_secret_encrypted", "client_secret_configured"},
	}

	for _, sf := range sensitiveFields {
		if val, exists := cfgMap[sf.encKey]; exists && val != nil && val != "" && val != false {
			cfgMap[sf.statusKey] = true
			cfgMap[sf.encKey] = true
		}
	}

	redactedBytes, err := json.Marshal(cfgMap)
	if err != nil {
		return "{}"
	}
	return string(redactedBytes)
}

// RedactInboundAuthConfig redacts sensitive inbound secrets from plugin configuration JSON strings
func RedactInboundAuthConfig(configStr string) string {
	if configStr == "" {
		return "{}"
	}
	var cfgMap map[string]interface{}
	if err := json.Unmarshal([]byte(configStr), &cfgMap); err != nil {
		return "{}"
	}

	sensitiveFields := []struct {
		plainKey  string
		encKey    string
		statusKey string
	}{
		{"password", "password_encrypted", "password_configured"},
		{"secret", "secret_encrypted", "secret_configured"},
		{"secret_key", "secret_key_encrypted", "secret_key_configured"},
		{"key", "key_encrypted", "key_configured"},
		{"api_key", "api_key_encrypted", "api_key_configured"},
	}

	for _, sf := range sensitiveFields {
		hasSecret := false
		if val, exists := cfgMap[sf.plainKey]; exists && val != nil && val != "" && val != false {
			hasSecret = true
			delete(cfgMap, sf.plainKey)
		}
		if val, exists := cfgMap[sf.encKey]; exists && val != nil && val != "" && val != false {
			hasSecret = true
			cfgMap[sf.encKey] = true
		}
		if hasSecret {
			cfgMap[sf.statusKey] = true
		}
	}

	redactedBytes, err := json.Marshal(cfgMap)
	if err != nil {
		return "{}"
	}
	return string(redactedBytes)
}

// EncryptInboundAuthConfig encrypts plaintext secrets in inbound plugin configuration JSON before database storage
func EncryptInboundAuthConfig(configStr string, key []byte) (string, error) {
	if configStr == "" {
		return "{}", nil
	}
	var cfgMap map[string]interface{}
	if err := json.Unmarshal([]byte(configStr), &cfgMap); err != nil {
		return configStr, nil
	}

	sensitiveFields := []struct {
		plainKey string
		encKey   string
	}{
		{"password", "password_encrypted"},
		{"secret", "secret_encrypted"},
		{"secret_key", "secret_key_encrypted"},
		{"key", "key_encrypted"},
		{"api_key", "api_key_encrypted"},
	}

	for _, sf := range sensitiveFields {
		if val, exists := cfgMap[sf.plainKey]; exists {
			if strVal, ok := val.(string); ok && strVal != "" {
				encrypted, err := crypto.EncryptPassword(key, strVal)
				if err != nil {
					return "", err
				}
				cfgMap[sf.encKey] = encrypted
				delete(cfgMap, sf.plainKey)
			}
		}
	}

	bytes, err := json.Marshal(cfgMap)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// SetOutboundAuthRequest represents payload for setting outbound auth
type SetOutboundAuthRequest struct {
	AuthType string          `json:"auth_type"`
	Config   json.RawMessage `json:"config"`
	Enabled  bool            `json:"enabled"`
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

	configMap := make(map[string]interface{})
	if len(req.Config) > 0 {
		// First try unmarshaling as map
		if err := json.Unmarshal(req.Config, &configMap); err != nil {
			// If not a direct map, try unmarshaling as a JSON string
			var configStr string
			if strErr := json.Unmarshal(req.Config, &configStr); strErr == nil && configStr != "" {
				_ = json.Unmarshal([]byte(configStr), &configMap)
			}
		}
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
		if pass, ok := configMap["password"].(string); ok && pass != "" {
			enc, err := crypto.EncryptPassword(h.encryptionKey, pass)
			if err != nil {
				httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to encrypt password: "+err.Error())
				return
			}
			configMap["password_encrypted"] = enc
			delete(configMap, "password")
		} else if existingConfig != nil && existingConfig["password_encrypted"] != nil {
			configMap["password_encrypted"] = existingConfig["password_encrypted"]
		}

	case models.OutboundBearer:
		if token, ok := configMap["token"].(string); ok && token != "" {
			enc, err := crypto.EncryptPassword(h.encryptionKey, token)
			if err != nil {
				httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to encrypt token: "+err.Error())
				return
			}
			configMap["token_encrypted"] = enc
			delete(configMap, "token")
		} else if existingConfig != nil && existingConfig["token_encrypted"] != nil {
			configMap["token_encrypted"] = existingConfig["token_encrypted"]
		}

	case models.OutboundAPIKey:
		if key, ok := configMap["key"].(string); ok && key != "" {
			enc, err := crypto.EncryptPassword(h.encryptionKey, key)
			if err != nil {
				httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to encrypt api key: "+err.Error())
				return
			}
			configMap["key_encrypted"] = enc
			delete(configMap, "key")
		} else if existingConfig != nil && existingConfig["key_encrypted"] != nil {
			configMap["key_encrypted"] = existingConfig["key_encrypted"]
		}

	case models.OutboundAWSSigV4:
		if accessKey, ok := configMap["access_key"].(string); ok && accessKey != "" {
			enc, err := crypto.EncryptPassword(h.encryptionKey, accessKey)
			if err != nil {
				httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to encrypt access key: "+err.Error())
				return
			}
			configMap["access_key_encrypted"] = enc
			delete(configMap, "access_key")
		} else if existingConfig != nil && existingConfig["access_key_encrypted"] != nil {
			configMap["access_key_encrypted"] = existingConfig["access_key_encrypted"]
		}

		if secretKey, ok := configMap["secret_key"].(string); ok && secretKey != "" {
			enc, err := crypto.EncryptPassword(h.encryptionKey, secretKey)
			if err != nil {
				httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to encrypt secret key: "+err.Error())
				return
			}
			configMap["secret_key_encrypted"] = enc
			delete(configMap, "secret_key")
		} else if existingConfig != nil && existingConfig["secret_key_encrypted"] != nil {
			configMap["secret_key_encrypted"] = existingConfig["secret_key_encrypted"]
		}

	case models.OutboundOAuth2CC:
		if secret, ok := configMap["client_secret"].(string); ok && secret != "" {
			enc, err := crypto.EncryptPassword(h.encryptionKey, secret)
			if err != nil {
				httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to encrypt client secret: "+err.Error())
				return
			}
			configMap["client_secret_encrypted"] = enc
			delete(configMap, "client_secret")
		} else if existingConfig != nil && existingConfig["client_secret_encrypted"] != nil {
			configMap["client_secret_encrypted"] = existingConfig["client_secret_encrypted"]
		}
	}

	configBytes, err := json.Marshal(configMap)
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

	respAuth := *outboundAuth
	respAuth.Config = RedactOutboundAuthConfig(outboundAuth.Config)
	httputil.WriteJSON(w, http.StatusOK, respAuth)
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
