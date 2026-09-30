package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"hash"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/apirout/apirout/internal/gateway/registry"
)

// HMACAuthPlugin implements HMAC signature authentication
type HMACAuthPlugin struct{}

// Name returns the plugin name
func (p *HMACAuthPlugin) Name() string { return "hmac-auth" }

// Authenticate validates HMAC signature from request headers
func (p *HMACAuthPlugin) Authenticate(r *http.Request, config map[string]interface{}, reader *registry.Reader) (*AuthResult, error) {
	clockSkew := getConfigInt(config, "clock_skew", 300)
	algorithms := getConfigStringSlice(config, "algorithms", []string{"hmac-sha256"})

	// Check for HMAC signature header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "hmac ") {
		return nil, fmt.Errorf("no HMAC authorization header found")
	}

	// Parse HMAC auth params: hmac username="...", algorithm="...", headers="...", signature="..."
	params := parseHMACParams(authHeader[5:])

	username := params["username"]
	algorithm := params["algorithm"]
	signature := params["signature"]
	headersStr := params["headers"]

	if username == "" || signature == "" {
		return nil, fmt.Errorf("incomplete HMAC authorization header")
	}

	// Validate algorithm
	if algorithm == "" {
		algorithm = algorithms[0]
	}

	validAlgo := false
	for _, a := range algorithms {
		if strings.EqualFold(a, algorithm) {
			validAlgo = true
			break
		}
	}
	if !validAlgo {
		return nil, fmt.Errorf("algorithm %s not allowed", algorithm)
	}

	// Validate date header for clock skew
	dateStr := r.Header.Get("Date")
	if dateStr != "" {
		requestDate, err := time.Parse(time.RFC1123, dateStr)
		if err == nil {
			skew := math.Abs(time.Since(requestDate).Seconds())
			if skew > float64(clockSkew) {
				return nil, fmt.Errorf("request date outside clock skew tolerance (%ds)", clockSkew)
			}
		}
	}

	// Look up consumer by username
	client, err := reader.GetAPIClientByUsername(username)
	if err != nil || client == nil {
		return nil, fmt.Errorf("unknown HMAC consumer")
	}

	// Build signing string from headers
	headers := strings.Split(headersStr, " ")
	if len(headers) == 0 || (len(headers) == 1 && headers[0] == "") {
		headers = []string{"date"}
	}

	var signingParts []string
	for _, h := range headers {
		h = strings.TrimSpace(h)
		if strings.EqualFold(h, "request-line") {
			signingParts = append(signingParts, fmt.Sprintf("%s %s HTTP/1.1", r.Method, r.URL.RequestURI()))
		} else {
			signingParts = append(signingParts, fmt.Sprintf("%s: %s", strings.ToLower(h), r.Header.Get(h)))
		}
	}
	signingString := strings.Join(signingParts, "\n")

	// Compute HMAC using the consumer's password hash as the shared secret
	var hashFunc func() hash.Hash
	switch strings.ToLower(algorithm) {
	case "hmac-sha256":
		hashFunc = sha256.New
	case "hmac-sha512":
		hashFunc = sha512.New
	default:
		hashFunc = sha256.New
	}

	mac := hmac.New(hashFunc, []byte(client.PasswordHash))
	mac.Write([]byte(signingString))
	expectedSig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedSig)) {
		return nil, fmt.Errorf("invalid HMAC signature")
	}

	groupID := ""
	if client.GroupID != nil {
		groupID = *client.GroupID
	}

	return &AuthResult{
		ConsumerID: client.ID,
		GroupID:    groupID,
		AuthType:  "hmac-auth",
	}, nil
}

// parseHMACParams extracts key="value" pairs from an HMAC authorization string
func parseHMACParams(s string) map[string]string {
	params := make(map[string]string)
	parts := strings.Split(s, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		eqIdx := strings.Index(part, "=")
		if eqIdx < 0 {
			continue
		}
		key := strings.TrimSpace(part[:eqIdx])
		val := strings.TrimSpace(part[eqIdx+1:])
		val = strings.Trim(val, "\"")
		params[key] = val
	}
	return params
}
