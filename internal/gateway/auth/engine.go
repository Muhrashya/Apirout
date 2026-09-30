package auth

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/apirout/apirout/internal/core/models"
	"github.com/apirout/apirout/internal/gateway/registry"
)

// AuthResult contains the authenticated consumer identity
type AuthResult struct {
	ConsumerID string
	GroupID    string
	AuthType  string
	Claims    map[string]interface{}
}

// Plugin defines the interface for inbound auth plugins
type Plugin interface {
	Name() string
	Authenticate(r *http.Request, config map[string]interface{}, reader *registry.Reader) (*AuthResult, error)
}

// Engine manages auth plugin resolution and execution
type Engine struct {
	plugins map[string]Plugin
	reader  *registry.Reader
}

// NewEngine creates a new auth engine with all registered plugins
func NewEngine(reader *registry.Reader, jwtSecret string) *Engine {
	engine := &Engine{
		plugins: make(map[string]Plugin),
		reader:  reader,
	}

	// Register built-in plugins
	engine.Register(&KeyAuthPlugin{})
	engine.Register(&BasicAuthPlugin{})
	engine.Register(&JWTAuthPlugin{secret: []byte(jwtSecret)})
	engine.Register(&HMACAuthPlugin{})

	return engine
}

// Register adds a plugin to the engine
func (e *Engine) Register(p Plugin) {
	e.plugins[p.Name()] = p
}

// Authenticate runs the auth chain for a request against an endpoint.
// Resolution priority (Kong-compatible):
//   1. Route-level plugins (most specific)
//   2. Service-level plugins
//   3. Global plugins (least specific)
//
// Multiple plugins at the same level use OR logic (Kong default).
func (e *Engine) Authenticate(r *http.Request, endpoint *models.Endpoint, serviceID *string) (*AuthResult, error) {
	// Skip auth for public endpoints
	if endpoint.IsPublic {
		return nil, nil
	}

	// 1. Try Route-level plugins
	routePlugins, _ := e.reader.GetAuthPluginsForRoute(endpoint.ID)
	if len(routePlugins) > 0 {
		return e.tryPlugins(r, routePlugins)
	}

	// 2. Try Service-level plugins
	if serviceID != nil && *serviceID != "" {
		servicePlugins, _ := e.reader.GetAuthPluginsForService(*serviceID)
		if len(servicePlugins) > 0 {
			return e.tryPlugins(r, servicePlugins)
		}
	}

	// 3. Try Global plugins
	globalPlugins, _ := e.reader.GetGlobalAuthPlugins()
	if len(globalPlugins) > 0 {
		return e.tryPlugins(r, globalPlugins)
	}

	// 4. No auth plugins configured — fall back to legacy auth
	return nil, nil
}

// tryPlugins attempts authentication with OR logic (first success wins)
func (e *Engine) tryPlugins(r *http.Request, plugins []models.AuthPluginConfig) (*AuthResult, error) {
	var lastErr error

	for _, pluginCfg := range plugins {
		plugin, exists := e.plugins[pluginCfg.PluginName]
		if !exists {
			continue
		}

		// Parse config JSON
		var config map[string]interface{}
		if err := json.Unmarshal([]byte(pluginCfg.Config), &config); err != nil {
			config = make(map[string]interface{})
		}

		result, err := plugin.Authenticate(r, config, e.reader)
		if err == nil && result != nil {
			return result, nil // First success wins (OR logic)
		}
		if err != nil {
			lastErr = err
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("authentication required")
}

// HasPlugins checks if any auth plugins are configured for the given endpoint/service
func (e *Engine) HasPlugins(endpointID string, serviceID *string) bool {
	routePlugins, _ := e.reader.GetAuthPluginsForRoute(endpointID)
	if len(routePlugins) > 0 {
		return true
	}

	if serviceID != nil && *serviceID != "" {
		servicePlugins, _ := e.reader.GetAuthPluginsForService(*serviceID)
		if len(servicePlugins) > 0 {
			return true
		}
	}

	globalPlugins, _ := e.reader.GetGlobalAuthPlugins()
	return len(globalPlugins) > 0
}
