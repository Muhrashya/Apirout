package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/apirout/apirout/internal/core/crypto"
	"github.com/apirout/apirout/internal/core/models"
)

// OutboundNegotiator defines the upstream outbound credential negotiation and request signing seam.
type OutboundNegotiator interface {
	Negotiate(req *http.Request, auth *models.ServiceOutboundAuth) error
	InjectAuth(req *http.Request, auth *models.ServiceOutboundAuth) error
}

// OutboundInjector handles outbound auth credential injection and signing into upstream requests
type OutboundInjector struct {
	encryptionKey   []byte
	oauth2CCManager *OAuth2CCManager
}

// NewOutboundInjector creates a new outbound auth injector/negotiator
func NewOutboundInjector(encryptionKey string) *OutboundInjector {
	return &OutboundInjector{
		encryptionKey:   crypto.NormalizeKey(encryptionKey),
		oauth2CCManager: NewOAuth2CCManager(),
	}
}

// SetOAuth2CCManager sets a custom OAuth2CCManager (for testing or custom clients)
func (o *OutboundInjector) SetOAuth2CCManager(manager *OAuth2CCManager) {
	o.oauth2CCManager = manager
}

// OAuth2CCManager returns the manager instance
func (o *OutboundInjector) OAuth2CCManager() *OAuth2CCManager {
	return o.oauth2CCManager
}

// Negotiate negotiates and injects upstream credentials into the outbound request
func (o *OutboundInjector) Negotiate(req *http.Request, auth *models.ServiceOutboundAuth) error {
	return o.InjectAuth(req, auth)
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

	switch strings.ToLower(auth.AuthType) {
	case models.OutboundBasic:
		return o.injectBasicAuth(req, config)
	case models.OutboundBearer:
		return o.injectBearerAuth(req, config)
	case models.OutboundAPIKey:
		return o.injectAPIKeyAuth(req, config)
	case models.OutboundAWSSigV4:
		return o.injectAWSSigV4(req, config)
	case models.OutboundOAuth2CC, "oauth2-client-credentials":
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

// injectAWSSigV4 performs authentic RFC-compliant AWS Signature Version 4 request signing
func (o *OutboundInjector) injectAWSSigV4(req *http.Request, config map[string]interface{}) error {
	accessKeyEnc := getConfigString(config, "access_key_encrypted", "")
	secretKeyEnc := getConfigString(config, "secret_key_encrypted", "")
	region := getConfigString(config, "region", "us-east-1")
	service := getConfigString(config, "service", "execute-api")
	if service == "execute-api" {
		if s := getConfigString(config, "service_name", ""); s != "" {
			service = s
		}
	}

	var accessKey string
	if accessKeyEnc != "" {
		dec, err := o.decrypt(accessKeyEnc)
		if err != nil {
			return fmt.Errorf("aws-sigv4: failed to decrypt access key: %w", err)
		}
		accessKey = dec
	} else {
		accessKey = getConfigString(config, "access_key", "")
		if accessKey == "" {
			accessKey = getConfigString(config, "access_key_id", "")
		}
	}

	if accessKey == "" {
		return fmt.Errorf("aws-sigv4: access key required")
	}

	var secretKey string
	if secretKeyEnc != "" {
		dec, err := o.decrypt(secretKeyEnc)
		if err != nil {
			return fmt.Errorf("aws-sigv4: failed to decrypt secret key: %w", err)
		}
		secretKey = dec
	} else {
		secretKey = getConfigString(config, "secret_key", "")
		if secretKey == "" {
			secretKey = getConfigString(config, "secret_access_key", "")
		}
	}

	if secretKey == "" {
		return fmt.Errorf("aws-sigv4: secret key required")
	}

	// Purge any legacy dummy stubs
	req.Header.Del("X-Apirout-AWS-Access-Key")
	req.Header.Del("X-Apirout-AWS-Region")
	req.Header.Del("X-Apirout-AWS-Service")

	return SignAWSSigV4(req, accessKey, secretKey, region, service, time.Now().UTC())
}

// injectOAuth2CC performs authentic OAuth2 Client Credentials token exchange and injects Bearer token
func (o *OutboundInjector) injectOAuth2CC(req *http.Request, config map[string]interface{}) error {
	tokenURL := getConfigString(config, "token_url", "")
	clientID := getConfigString(config, "client_id", "")
	clientSecretEnc := getConfigString(config, "client_secret_encrypted", "")
	scopes := getConfigString(config, "scopes", "")

	if tokenURL == "" {
		return fmt.Errorf("oauth2-cc: token_url required")
	}
	if clientID == "" {
		return fmt.Errorf("oauth2-cc: client_id required")
	}

	var clientSecret string
	if clientSecretEnc != "" {
		dec, err := o.decrypt(clientSecretEnc)
		if err != nil {
			return fmt.Errorf("oauth2-cc: failed to decrypt client secret: %w", err)
		}
		clientSecret = dec
	} else {
		clientSecret = getConfigString(config, "client_secret", "")
	}

	if clientSecret == "" {
		return fmt.Errorf("oauth2-cc: client_secret required")
	}

	// Purge legacy pseudo-header
	req.Header.Del("X-Apirout-OAuth2-Client")

	if o.oauth2CCManager == nil {
		o.oauth2CCManager = NewOAuth2CCManager()
	}

	token, err := o.oauth2CCManager.GetToken(tokenURL, clientID, clientSecret, scopes)
	if err != nil {
		return fmt.Errorf("oauth2-cc: token grant exchange failed: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	return nil
}
