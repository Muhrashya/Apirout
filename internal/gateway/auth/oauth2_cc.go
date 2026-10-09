package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

type cachedOAuthToken struct {
	accessToken string
	expiresAt   time.Time
}

// OAuth2CCManager coordinates OAuth2 Client Credentials token exchanges, caching, and concurrency deduplication
type OAuth2CCManager struct {
	client       *http.Client
	requestGroup singleflight.Group
	cacheMu      sync.RWMutex
	cache        map[string]*cachedOAuthToken
}

// NewOAuth2CCManager creates a new OAuth2 client credentials manager
func NewOAuth2CCManager(client ...*http.Client) *OAuth2CCManager {
	httpClient := &http.Client{Timeout: 10 * time.Second}
	if len(client) > 0 && client[0] != nil {
		httpClient = client[0]
	}
	return &OAuth2CCManager{
		client: httpClient,
		cache:  make(map[string]*cachedOAuthToken),
	}
}

// GetToken returns a valid OAuth2 access token, exchanging with upstream or returning from TTL cache
func (m *OAuth2CCManager) GetToken(tokenURL, clientID, clientSecret, scopes string) (string, error) {
	if tokenURL == "" {
		return "", fmt.Errorf("oauth2-cc: token_url required")
	}
	if clientID == "" {
		return "", fmt.Errorf("oauth2-cc: client_id required")
	}
	if clientSecret == "" {
		return "", fmt.Errorf("oauth2-cc: client_secret required")
	}

	cacheKey := fmt.Sprintf("%s|%s|%s", tokenURL, clientID, scopes)

	// 1. Check in-memory cache with proactive 60s expiration buffer
	m.cacheMu.RLock()
	cached, found := m.cache[cacheKey]
	if found && cached != nil {
		// Valid if expiry is at least 60 seconds into the future
		if time.Now().Add(60 * time.Second).Before(cached.expiresAt) {
			token := cached.accessToken
			m.cacheMu.RUnlock()
			return token, nil
		}
	}
	m.cacheMu.RUnlock()

	// 2. Perform deduplicated token exchange via singleflight
	tokenInterface, err, _ := m.requestGroup.Do(cacheKey, func() (interface{}, error) {
		// Double check cache in case a concurrent call resolved it
		m.cacheMu.RLock()
		cached, found := m.cache[cacheKey]
		if found && cached != nil && time.Now().Add(60*time.Second).Before(cached.expiresAt) {
			token := cached.accessToken
			m.cacheMu.RUnlock()
			return token, nil
		}
		m.cacheMu.RUnlock()

		// Prepare POST request with grant_type=client_credentials
		form := url.Values{}
		form.Set("grant_type", "client_credentials")
		form.Set("client_id", clientID)
		form.Set("client_secret", clientSecret)
		if scopes != "" {
			form.Set("scope", scopes)
		}

		httpReq, err := http.NewRequest("POST", tokenURL, strings.NewReader(form.Encode()))
		if err != nil {
			return nil, fmt.Errorf("failed to create oauth2 request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		httpReq.SetBasicAuth(clientID, clientSecret)

		resp, err := m.client.Do(httpReq)
		if err != nil {
			return nil, fmt.Errorf("oauth2 token request failed: %w", err)
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read oauth2 response body: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("oauth2 token endpoint returned HTTP %d: %s", resp.StatusCode, string(respBody))
		}

		var payload struct {
			AccessToken string `json:"access_token"`
			ExpiresIn   int64  `json:"expires_in"`
			TokenType   string `json:"token_type"`
		}
		if err := json.Unmarshal(respBody, &payload); err != nil {
			return nil, fmt.Errorf("failed to parse oauth2 token json: %w", err)
		}

		if payload.AccessToken == "" {
			return nil, fmt.Errorf("oauth2 token response missing access_token")
		}

		expiresIn := payload.ExpiresIn
		if expiresIn <= 0 {
			expiresIn = 3600 // Default to 1 hour if not specified
		}

		expiresAt := time.Now().Add(time.Duration(expiresIn) * time.Second)

		m.cacheMu.Lock()
		m.cache[cacheKey] = &cachedOAuthToken{
			accessToken: payload.AccessToken,
			expiresAt:   expiresAt,
		}
		m.cacheMu.Unlock()

		return payload.AccessToken, nil
	})

	if err != nil {
		return "", err
	}
	return tokenInterface.(string), nil
}

// ClearCache evicts all cached OAuth2 tokens
func (m *OAuth2CCManager) ClearCache() {
	m.cacheMu.Lock()
	m.cache = make(map[string]*cachedOAuthToken)
	m.cacheMu.Unlock()
}
