package auth

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/apirout/apirout/internal/gateway/registry"
	"golang.org/x/crypto/bcrypt"
)

// BasicAuthPlugin implements HTTP Basic Authentication
type BasicAuthPlugin struct{}

// Name returns the plugin name
func (p *BasicAuthPlugin) Name() string { return "basic-auth" }

// Authenticate validates HTTP Basic Auth credentials against consumer_credentials
func (p *BasicAuthPlugin) Authenticate(r *http.Request, config map[string]interface{}, reader *registry.Reader) (*AuthResult, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("missing Authorization header")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Basic") {
		return nil, fmt.Errorf("invalid Authorization header format, expected Basic")
	}

	decoded, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid base64 encoding in Authorization header")
	}

	colonIdx := strings.IndexByte(string(decoded), ':')
	if colonIdx < 0 {
		return nil, fmt.Errorf("invalid Basic auth format, expected username:password")
	}

	username := string(decoded[:colonIdx])
	password := string(decoded[colonIdx+1:])

	if username == "" || password == "" {
		return nil, fmt.Errorf("username and password are required")
	}

	// Look up consumer by username via api_clients table
	client, err := reader.GetAPIClientByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("authentication failed")
	}
	if client == nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Verify password against stored hash
	if err := bcrypt.CompareHashAndPassword([]byte(client.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	groupID := ""
	if client.GroupID != nil {
		groupID = *client.GroupID
	}

	return &AuthResult{
		ConsumerID: client.ID,
		GroupID:    groupID,
		AuthType:  "basic-auth",
	}, nil
}
