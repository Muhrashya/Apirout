package server

import (
	"encoding/json"
	"testing"
)

func TestRedactOutboundAuthConfig(t *testing.T) {
	rawConfig := `{
		"username": "api_user",
		"password_encrypted": "U2FsdGVkX1+vupppZksvRf5pq5g5XjFRIipRkw++0C1=",
		"token_encrypted": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.e30.fake",
		"key_encrypted": "enc_api_key_secret_12345",
		"access_key_encrypted": "enc_access_key_xyz",
		"secret_key_encrypted": "enc_secret_key_abc",
		"client_secret_encrypted": "enc_client_secret_def",
		"prefix": "Bearer",
		"header_name": "X-API-Key"
	}`

	redacted := RedactOutboundAuthConfig(rawConfig)

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(redacted), &parsed); err != nil {
		t.Fatalf("failed to parse redacted config JSON: %v", err)
	}

	// Non-sensitive fields preserved
	if parsed["username"] != "api_user" {
		t.Errorf("expected username 'api_user', got: %v", parsed["username"])
	}
	if parsed["prefix"] != "Bearer" {
		t.Errorf("expected prefix 'Bearer', got: %v", parsed["prefix"])
	}
	if parsed["header_name"] != "X-API-Key" {
		t.Errorf("expected header_name 'X-API-Key', got: %v", parsed["header_name"])
	}

	// Sensitive ciphertexts should NOT be strings
	sensitiveKeys := []string{
		"password_encrypted",
		"token_encrypted",
		"key_encrypted",
		"access_key_encrypted",
		"secret_key_encrypted",
		"client_secret_encrypted",
	}

	for _, k := range sensitiveKeys {
		val := parsed[k]
		if _, isString := val.(string); isString {
			t.Errorf("key %s must not contain string ciphertext in output: %v", k, val)
		}
		if val != true {
			t.Errorf("key %s should be boolean true marker, got: %v", k, val)
		}
	}

	// Configured status boolean flags should all be true
	statusKeys := []string{
		"password_configured",
		"token_configured",
		"key_configured",
		"access_key_configured",
		"secret_key_configured",
		"client_secret_configured",
	}

	for _, k := range statusKeys {
		if parsed[k] != true {
			t.Errorf("expected %s == true, got: %v", k, parsed[k])
		}
	}
}

func TestRedactOutboundAuthConfig_Empty(t *testing.T) {
	if got := RedactOutboundAuthConfig(""); got != "{}" {
		t.Errorf("expected '{}' for empty input, got: %s", got)
	}
	if got := RedactOutboundAuthConfig("{}"); got != "{}" {
		t.Errorf("expected '{}' for '{}' input, got: %s", got)
	}
}
