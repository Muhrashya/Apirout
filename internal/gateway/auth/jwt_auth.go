package auth

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/apirout/apirout/internal/gateway/registry"
)

// JWTAuthPlugin implements JWT Bearer token authentication
type JWTAuthPlugin struct {
	secret []byte // default HMAC secret
}

// Name returns the plugin name
func (p *JWTAuthPlugin) Name() string { return "jwt-auth" }

// Authenticate validates a JWT token from the Authorization header
func (p *JWTAuthPlugin) Authenticate(r *http.Request, config map[string]interface{}, reader *registry.Reader) (*AuthResult, error) {
	// Get header names to check (default: Authorization)
	headerNames := getConfigStringSlice(config, "header_names", []string{"Authorization"})
	algorithms := getConfigStringSlice(config, "algorithms", []string{"HS256"})
	keyClaimName := getConfigString(config, "key_claim_name", "iss")

	var tokenString string

	// Extract token from headers
	for _, headerName := range headerNames {
		headerVal := r.Header.Get(headerName)
		if headerVal == "" {
			continue
		}

		if strings.EqualFold(headerName, "Authorization") {
			parts := strings.SplitN(headerVal, " ", 2)
			if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
				tokenVal := parts[1]
				// Skip API keys (ak_ prefix) — those are handled by key-auth
				if !strings.HasPrefix(tokenVal, "ak_") {
					tokenString = tokenVal
					break
				}
			}
		} else {
			tokenString = headerVal
			break
		}
	}

	if tokenString == "" {
		return nil, fmt.Errorf("no JWT token found in request")
	}

	// Parse and validate the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		alg, ok := token.Header["alg"].(string)
		if !ok || alg == "" {
			return nil, fmt.Errorf("missing or invalid alg header")
		}

		// Verify algorithm is allowed
		allowed := false
		for _, a := range algorithms {
			if strings.EqualFold(a, alg) {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, fmt.Errorf("algorithm %s not allowed", alg)
		}

		// Ensure the signing method matches HMAC to prevent algorithm confusion
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return p.secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid or expired token: %s", err.Error())
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Try to resolve consumer ID from claims
	consumerID := ""
	groupID := ""

	// Check for api_client_id claim (Apirout internal format)
	if cid, ok := claims["api_client_id"].(string); ok && cid != "" {
		consumerID = cid
	}

	// Check for key claim (e.g., "iss" for consumer resolution)
	if consumerID == "" {
		if issuer, ok := claims[keyClaimName].(string); ok && issuer != "" {
			// Try to resolve consumer by username/issuer
			client, err := reader.GetAPIClientByUsername(issuer)
			if err == nil && client != nil {
				consumerID = client.ID
				if client.GroupID != nil {
					groupID = *client.GroupID
				}
			}
		}
	}

	// Check for group_id claim
	if gid, ok := claims["group_id"].(string); ok && gid != "" {
		groupID = gid
	}

	if consumerID == "" {
		// JWT is valid but we couldn't resolve a consumer — use sub claim as consumer ID
		if sub, ok := claims["sub"].(string); ok && sub != "" {
			consumerID = sub
		} else {
			return nil, fmt.Errorf("cannot resolve consumer from JWT claims")
		}
	}

	// Build claims map for downstream use
	claimsMap := make(map[string]interface{})
	for k, v := range claims {
		claimsMap[k] = v
	}

	return &AuthResult{
		ConsumerID: consumerID,
		GroupID:    groupID,
		AuthType:  "jwt",
		Claims:    claimsMap,
	}, nil
}
