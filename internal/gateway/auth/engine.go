package auth

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/apirout/apirout/internal/core/models"
	"github.com/apirout/apirout/internal/gateway/apiclientauth"
	"github.com/apirout/apirout/internal/gateway/registry"
	"github.com/apirout/apirout/internal/gateway/responder"
)

// Error represents a typed authentication error with status code and canonical error code
type Error struct {
	StatusCode int
	Code       responder.ErrorCode
	Message    string
}

func (e *Error) Error() string {
	return e.Message
}

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

// Engine manages auth plugin resolution, execution, and fallback authentication
type Engine struct {
	plugins       map[string]Plugin
	reader        *registry.Reader
	jwtSecret     string
	apiClientAuth *apiclientauth.Service
}

// NewEngine creates a new auth engine with all registered plugins
func NewEngine(reader *registry.Reader, jwtSecret string, apiClientAuth ...*apiclientauth.Service) *Engine {
	var acAuth *apiclientauth.Service
	if len(apiClientAuth) > 0 && apiClientAuth[0] != nil {
		acAuth = apiClientAuth[0]
	} else if jwtSecret != "" && reader != nil {
		acAuth = apiclientauth.NewService(jwtSecret, reader, 24*time.Hour)
	}

	engine := &Engine{
		plugins:       make(map[string]Plugin),
		reader:        reader,
		jwtSecret:     jwtSecret,
		apiClientAuth: acAuth,
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
// Resolution priority:
//   1. Public endpoint check (bypass)
//   2. Route-level plugins (most specific)
//   3. Service-level plugins
//   4. Global plugins (least specific)
//   5. Fallback credential evaluation (API key or JWT token)
//
// Multiple plugins at the same level use OR logic (first match wins).
func (e *Engine) Authenticate(r *http.Request, endpoint *models.Endpoint, serviceID *string) (*AuthResult, error) {
	// Skip auth for public endpoints
	if endpoint.IsPublic {
		return nil, nil
	}

	if e.reader == nil {
		return e.authenticateFallback(r)
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

	// 4. No auth plugins configured — evaluate fallback API key / JWT authentication
	return e.authenticateFallback(r)
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
		errStr := lastErr.Error()
		code := responder.ErrCodeUnauthorized
		if strings.Contains(strings.ToLower(errStr), "invalid api key") {
			code = responder.ErrCodeInvalidKey
		} else if strings.Contains(strings.ToLower(errStr), "expired") {
			code = responder.ErrCodeExpiredToken
		}
		return nil, &Error{
			StatusCode: http.StatusUnauthorized,
			Code:       code,
			Message:    errStr,
		}
	}
	return nil, &Error{
		StatusCode: http.StatusUnauthorized,
		Code:       responder.ErrCodeUnauthorized,
		Message:    "authentication required",
	}
}

// authenticateFallback handles default API key and JWT bearer token authentication
func (e *Engine) authenticateFallback(r *http.Request) (*AuthResult, error) {
	jwtToken, rawAPIKey := ExtractTokenOrKey(r)

	if rawAPIKey != "" {
		if e.reader == nil {
			return nil, &Error{
				StatusCode: http.StatusUnauthorized,
				Code:       responder.ErrCodeInvalidKey,
				Message:    "Invalid API key: registry reader unavailable",
			}
		}
		consumer, _, err := e.reader.ValidateAPIKey(rawAPIKey)
		if err != nil {
			return nil, &Error{
				StatusCode: http.StatusUnauthorized,
				Code:       responder.ErrCodeInvalidKey,
				Message:    "Invalid API key: " + err.Error(),
			}
		}
		groupID := ""
		if consumer.GroupID != nil {
			groupID = *consumer.GroupID
		}
		return &AuthResult{
			ConsumerID: consumer.ID,
			GroupID:    groupID,
			AuthType:  "key-auth",
		}, nil
	}

	if jwtToken != "" {
		if e.apiClientAuth != nil {
			claims, err := e.apiClientAuth.ValidateToken(jwtToken)
			if err != nil {
				return nil, &Error{
					StatusCode: http.StatusUnauthorized,
					Code:       responder.ErrCodeExpiredToken,
					Message:    "Invalid or expired token",
				}
			}
			return &AuthResult{
				ConsumerID: claims.APIClientID,
				GroupID:    claims.GroupID,
				AuthType:  "jwt",
			}, nil
		}
	}

	return nil, &Error{
		StatusCode: http.StatusUnauthorized,
		Code:       responder.ErrCodeUnauthorized,
		Message:    "Authentication required (provide X-API-Key or Authorization Bearer token)",
	}
}

// ExtractTokenOrKey parses API keys or JWT tokens from standard headers and query parameters
func ExtractTokenOrKey(r *http.Request) (token string, apiKey string) {
	// 1. Direct API Key header
	if key := r.Header.Get("X-API-Key"); key != "" {
		return "", key
	}

	// 2. Query param ?api_key=...
	if key := r.URL.Query().Get("api_key"); key != "" {
		return "", key
	}

	// 3. Authorization header: Bearer ak_... OR Bearer <jwt>
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			bearerVal := parts[1]
			if strings.HasPrefix(bearerVal, "ak_") {
				return "", bearerVal
			}
			return bearerVal, ""
		}
	}

	// 4. Legacy X-API-Token
	if apiTokenHeader := r.Header.Get("X-API-Token"); apiTokenHeader != "" {
		if strings.HasPrefix(apiTokenHeader, "ak_") {
			return "", apiTokenHeader
		}
		return apiTokenHeader, ""
	}

	return "", ""
}

// HasPlugins checks if any auth plugins are configured for the given endpoint/service
func (e *Engine) HasPlugins(endpointID string, serviceID *string) bool {
	if e.reader == nil {
		return false
	}
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
