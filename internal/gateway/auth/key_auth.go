package auth

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/apirout/apirout/internal/gateway/registry"
)

// KeyAuthPlugin implements key-auth (API Key) authentication
type KeyAuthPlugin struct{}

// Name returns the plugin name
func (p *KeyAuthPlugin) Name() string { return "key-auth" }

// Authenticate validates an API key from header, query param, or body
func (p *KeyAuthPlugin) Authenticate(r *http.Request, config map[string]interface{}, reader *registry.Reader) (*AuthResult, error) {
	// Parse config with defaults
	keyNames := getConfigStringSlice(config, "key_names", []string{"X-API-Key", "api_key"})
	keyInHeader := getConfigBool(config, "key_in_header", true)
	keyInQuery := getConfigBool(config, "key_in_query", true)

	var rawKey string

	// 1. Check headers
	if keyInHeader {
		for _, name := range keyNames {
			if val := r.Header.Get(name); val != "" {
				rawKey = val
				break
			}
		}
	}

	// 2. Check query params
	if rawKey == "" && keyInQuery {
		for _, name := range keyNames {
			if val := r.URL.Query().Get(name); val != "" {
				rawKey = val
				break
			}
		}
	}

	// 3. Check Authorization Bearer with ak_ prefix
	if rawKey == "" {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
				if strings.HasPrefix(parts[1], "ak_") {
					rawKey = parts[1]
				}
			}
		}
	}

	// 4. Legacy X-API-Token
	if rawKey == "" {
		if val := r.Header.Get("X-API-Token"); val != "" && strings.HasPrefix(val, "ak_") {
			rawKey = val
		}
	}

	if rawKey == "" {
		return nil, fmt.Errorf("no API key found in request")
	}

	// Validate against consumer_credentials
	consumer, _, err := reader.ValidateAPIKey(rawKey)
	if err != nil {
		return nil, fmt.Errorf("invalid API key: %s", err.Error())
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

// --- Config helpers ---

func getConfigStringSlice(config map[string]interface{}, key string, defaultVal []string) []string {
	val, ok := config[key]
	if !ok {
		return defaultVal
	}

	switch v := val.(type) {
	case []string:
		return v
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		if len(result) > 0 {
			return result
		}
	case string:
		return strings.Split(v, ",")
	}
	return defaultVal
}

func getConfigBool(config map[string]interface{}, key string, defaultVal bool) bool {
	val, ok := config[key]
	if !ok {
		return defaultVal
	}
	if b, ok := val.(bool); ok {
		return b
	}
	return defaultVal
}

func getConfigString(config map[string]interface{}, key string, defaultVal string) string {
	val, ok := config[key]
	if !ok {
		return defaultVal
	}
	if s, ok := val.(string); ok && s != "" {
		return s
	}
	return defaultVal
}

func getConfigInt(config map[string]interface{}, key string, defaultVal int) int {
	val, ok := config[key]
	if !ok {
		return defaultVal
	}
	switch v := val.(type) {
	case float64:
		return int(v)
	case int:
		return v
	}
	return defaultVal
}
