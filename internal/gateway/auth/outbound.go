package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/apirout/apirout/internal/core/crypto"
	"github.com/apirout/apirout/internal/core/models"
)

// OutboundInjector handles outbound auth credential injection into upstream requests
type OutboundInjector struct {
	encryptionKey []byte
}

// NewOutboundInjector creates a new outbound auth injector
func NewOutboundInjector(encryptionKey string) *OutboundInjector {
	return &OutboundInjector{encryptionKey: crypto.NormalizeKey(encryptionKey)}
}

// InjectAuth injects authentication credentials into the outbound request based on config
func (o *OutboundInjector) InjectAuth(req *http.Request, auth *models.ServiceOutboundAuth) error {
	if auth == nil || !auth.Enabled {
		return nil
	}

	var config map[string]interface{}
	if err := json.Unmarshal([]byte(auth.Config), &config); err != nil {
		return fmt.Errorf("invalid outbound auth config: %w", err)
	}

	switch auth.AuthType {
	case models.OutboundBasic:
		return o.injectBasicAuth(req, config)
	case models.OutboundBearer:
		return o.injectBearerAuth(req, config)
	case models.OutboundAPIKey:
		return o.injectAPIKeyAuth(req, config)
	case models.OutboundAWSSigV4:
		return o.injectAWSSigV4(req, config)
	case models.OutboundOAuth2CC:
		return o.injectOAuth2CC(req, config)
	default:
		return fmt.Errorf("unsupported outbound auth type: %s", auth.AuthType)
	}
}

func (o *OutboundInjector) decrypt(encrypted string) (string, error) {
	return crypto.DecryptPassword(o.encryptionKey, encrypted)
}

// injectBasicAuth adds HTTP Basic Authorization header
func (o *OutboundInjector) injectBasicAuth(req *http.Request, config map[string]interface{}) error {
	username := getConfigString(config, "username", "")
	passwordEnc := getConfigString(config, "password_encrypted", "")

	if username == "" || passwordEnc == "" {
		return fmt.Errorf("basic auth: username and password required")
	}

	password, err := o.decrypt(passwordEnc)
	if err != nil {
		return fmt.Errorf("basic auth: failed to decrypt password: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	req.Header.Set("Authorization", "Basic "+encoded)
	return nil
}

// injectBearerAuth adds Bearer token Authorization header
func (o *OutboundInjector) injectBearerAuth(req *http.Request, config map[string]interface{}) error {
	tokenEnc := getConfigString(config, "token_encrypted", "")
	prefix := getConfigString(config, "prefix", "Bearer")

	if tokenEnc == "" {
		return fmt.Errorf("bearer auth: token required")
	}

	token, err := o.decrypt(tokenEnc)
	if err != nil {
		return fmt.Errorf("bearer auth: failed to decrypt token: %w", err)
	}

	req.Header.Set("Authorization", prefix+" "+token)
	return nil
}

// injectAPIKeyAuth adds API key as header or query parameter
func (o *OutboundInjector) injectAPIKeyAuth(req *http.Request, config map[string]interface{}) error {
	headerName := getConfigString(config, "header_name", "X-API-Key")
	keyEnc := getConfigString(config, "key_encrypted", "")
	inQuery := getConfigBool(config, "in_query", false)
	queryParam := getConfigString(config, "query_param", "api_key")

	if keyEnc == "" {
		return fmt.Errorf("api-key auth: key required")
	}

	key, err := o.decrypt(keyEnc)
	if err != nil {
		return fmt.Errorf("api-key auth: failed to decrypt key: %w", err)
	}

	if inQuery {
		q := req.URL.Query()
		q.Set(queryParam, key)
		req.URL.RawQuery = q.Encode()
	} else {
		req.Header.Set(headerName, key)
	}
	return nil
}

// injectAWSSigV4 adds AWS Signature V4 headers (simplified — adds Access Key header)
func (o *OutboundInjector) injectAWSSigV4(req *http.Request, config map[string]interface{}) error {
	accessKeyEnc := getConfigString(config, "access_key_encrypted", "")
	region := getConfigString(config, "region", "us-east-1")
	service := getConfigString(config, "service", "execute-api")

	if accessKeyEnc == "" {
		return fmt.Errorf("aws-sigv4: access key required")
	}

	accessKey, err := o.decrypt(accessKeyEnc)
	if err != nil {
		return fmt.Errorf("aws-sigv4: failed to decrypt access key: %w", err)
	}

	// Simplified: set headers for downstream AWS SDK processing
	req.Header.Set("X-Apirout-AWS-Access-Key", accessKey)
	req.Header.Set("X-Apirout-AWS-Region", region)
	req.Header.Set("X-Apirout-AWS-Service", service)

	return nil
}

// injectOAuth2CC adds OAuth2 Client Credentials token
func (o *OutboundInjector) injectOAuth2CC(req *http.Request, config map[string]interface{}) error {
	cachedTokenEnc := getConfigString(config, "cached_token", "")
	if cachedTokenEnc != "" {
		token, err := o.decrypt(cachedTokenEnc)
		if err == nil && token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
			return nil
		}
	}

	// Fallback: use client_id header for the upstream to handle auth
	clientID := getConfigString(config, "client_id", "")
	if clientID != "" {
		req.Header.Set("X-Apirout-OAuth2-Client", clientID)
	}

	return nil
}
