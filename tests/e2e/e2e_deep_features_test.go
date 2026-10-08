package e2e_test

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testJWTSecret = "super-secret-jwt-key-min-32-characters!"

// Helper to generate signed HS256 JWT tokens for testing
func signTestJWT(claims jwt.MapClaims, secret string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		panic(fmt.Sprintf("failed to sign test JWT: %v", err))
	}
	return signed
}

func TestE2E_KeyAuth_QueryParamAndBearerVariations(t *testing.T) {
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1000000)
	svcName := "key-query-svc-" + suffix
	routePath := "/api/v1/key-query-" + suffix
	username := "key-query-user-" + suffix

	// 1. Create Consumer
	code, consRes := adminRequest(t, "POST", "/api/v1/consumers", map[string]any{
		"username": username,
		"status":   "active",
	})
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to create consumer: %d %v", code, consRes)
	}
	consumerID := extractID(consRes)
	t.Cleanup(func() {
		if consumerID != "" {
			_, _ = adminRequest(t, "DELETE", "/api/v1/consumers/"+consumerID, nil)
		}
	})

	// 2. Create API Key
	code, credRes := adminRequest(t, "POST", fmt.Sprintf("/api/v1/consumers/%s/credentials", consumerID), map[string]any{
		"name":     "Query Param Key",
		"key_type": "api-key",
	})
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to create credential: %d %v", code, credRes)
	}
	apiKey := fmt.Sprintf("%v", credRes["key_secret"])
	if apiKey == "" || apiKey == "<nil>" {
		if d, ok := credRes["data"].(map[string]any); ok {
			apiKey = fmt.Sprintf("%v", d["key_secret"])
		}
	}

	// 3. Create Service pointing to status health
	code, svcRes := adminRequest(t, "POST", "/api/v1/services", map[string]any{
		"name":     svcName,
		"protocol": "http",
		"host":     "127.0.0.1",
		"port":     8100,
		"path":     "/health",
		"enabled":  true,
	})
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to create service: %d %v", code, svcRes)
	}
	svcID := extractID(svcRes)
	t.Cleanup(func() {
		if svcID != "" {
			_, _ = adminRequest(t, "DELETE", "/api/v1/services/"+svcID, nil)
		}
	})

	// 4. Create Protected Route
	code, epRes := adminRequest(t, "POST", "/api/v1/endpoints", map[string]any{
		"service_id": svcID,
		"method":     "GET",
		"path":       routePath,
		"strip_path": true,
		"is_public":  false,
		"enabled":    true,
	})
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to create endpoint: %d %v", code, epRes)
	}
	epID := extractID(epRes)
	t.Cleanup(func() {
		if epID != "" {
			_, _ = adminRequest(t, "DELETE", "/api/v1/endpoints/"+epID, nil)
		}
	})

	// 5. Attach key-auth plugin to Route
	code, pRes := adminRequest(t, "POST", "/api/v1/auth-plugins", map[string]any{
		"plugin_name": "key-auth",
		"scope":       "route",
		"route_id":    epID,
		"enabled":     true,
		"config":      map[string]any{},
	})
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to attach key-auth plugin: %d %v", code, pRes)
	}
	pID := extractID(pRes)
	t.Cleanup(func() {
		if pID != "" {
			_, _ = adminRequest(t, "DELETE", "/api/v1/auth-plugins/"+pID, nil)
		}
	})

	time.Sleep(200 * time.Millisecond)

	t.Run("Default Query Param api_key Accepted", func(t *testing.T) {
		url := routePath + "?api_key=" + apiKey
		code, _, body := gatewayRequestWithHeaders(t, "GET", url, nil, nil)
		if code != http.StatusOK {
			t.Fatalf("expected 200 OK with ?api_key, got %d: %s", code, string(body))
		}
	})

	t.Run("Unconfigured Query Param apikey Rejected by Default", func(t *testing.T) {
		url := routePath + "?apikey=" + apiKey
		code, _, _ := gatewayRequestWithHeaders(t, "GET", url, nil, nil)
		if code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized with unconfigured ?apikey, got %d", code)
		}
	})

	t.Run("Configured Custom key_names Accepts apikey", func(t *testing.T) {
		// Create a second route with custom key_names: ["apikey"]
		customRoutePath := routePath + "-custom"
		code, cEpRes := adminRequest(t, "POST", "/api/v1/endpoints", map[string]any{
			"service_id": svcID,
			"method":     "GET",
			"path":       customRoutePath,
			"strip_path": true,
			"is_public":  false,
			"enabled":    true,
		})
		if code != http.StatusCreated && code != http.StatusOK {
			t.Fatalf("failed to create custom endpoint: %d %v", code, cEpRes)
		}
		cEpID := extractID(cEpRes)
		t.Cleanup(func() {
			if cEpID != "" {
				_, _ = adminRequest(t, "DELETE", "/api/v1/endpoints/"+cEpID, nil)
			}
		})

		code, cPRes := adminRequest(t, "POST", "/api/v1/auth-plugins", map[string]any{
			"plugin_name": "key-auth",
			"scope":       "route",
			"route_id":    cEpID,
			"enabled":     true,
			"config": map[string]any{
				"key_names": []string{"apikey", "api_key"},
			},
		})
		if code != http.StatusCreated && code != http.StatusOK {
			t.Fatalf("failed to attach custom key-auth plugin: %d %v", code, cPRes)
		}
		cPID := extractID(cPRes)
		t.Cleanup(func() {
			if cPID != "" {
				_, _ = adminRequest(t, "DELETE", "/api/v1/auth-plugins/"+cPID, nil)
			}
		})

		time.Sleep(200 * time.Millisecond)
		url := customRoutePath + "?apikey=" + apiKey
		code, _, body := gatewayRequestWithHeaders(t, "GET", url, nil, nil)
		if code != http.StatusOK {
			t.Fatalf("expected 200 OK with custom key_names ?apikey, got %d: %s", code, string(body))
		}
	})

	t.Run("Query Param api_key Invalid Key Rejected", func(t *testing.T) {
		url := routePath + "?api_key=ak_invalid_key_9999"
		code, _, _ := gatewayRequestWithHeaders(t, "GET", url, nil, nil)
		if code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized with invalid query key, got %d", code)
		}
	})

	t.Run("Header Authorization Bearer ak_ Prefix", func(t *testing.T) {
		headers := map[string]string{"Authorization": "Bearer " + apiKey}
		code, _, body := gatewayRequestWithHeaders(t, "GET", routePath, headers, nil)
		if code != http.StatusOK {
			t.Fatalf("expected 200 OK with Authorization: Bearer ak_..., got %d: %s", code, string(body))
		}
	})

	t.Run("Header Legacy X-API-Token ak_ Prefix", func(t *testing.T) {
		headers := map[string]string{"X-API-Token": apiKey}
		code, _, body := gatewayRequestWithHeaders(t, "GET", routePath, headers, nil)
		if code != http.StatusOK {
			t.Fatalf("expected 200 OK with X-API-Token, got %d: %s", code, string(body))
		}
	})

	t.Run("Missing Key Credentials Rejected", func(t *testing.T) {
		code, _, _ := gatewayRequestWithHeaders(t, "GET", routePath, nil, nil)
		if code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized with missing credentials, got %d", code)
		}
	})
}

func TestE2E_JWTAuth_LivePluginLifecycle(t *testing.T) {
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1000000)
	svcName := "jwt-live-svc-" + suffix
	routePath := "/api/v1/jwt-live-" + suffix
	username := "jwt-user-" + suffix

	// 1. Create Consumer
	code, consRes := adminRequest(t, "POST", "/api/v1/consumers", map[string]any{
		"username": username,
		"status":   "active",
	})
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to create consumer: %d %v", code, consRes)
	}
	consumerID := extractID(consRes)
	t.Cleanup(func() {
		if consumerID != "" {
			_, _ = adminRequest(t, "DELETE", "/api/v1/consumers/"+consumerID, nil)
		}
	})

	// 2. Create Service
	code, svcRes := adminRequest(t, "POST", "/api/v1/services", map[string]any{
		"name":     svcName,
		"protocol": "http",
		"host":     "127.0.0.1",
		"port":     8100,
		"path":     "/health",
		"enabled":  true,
	})
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to create service: %d %v", code, svcRes)
	}
	svcID := extractID(svcRes)
	t.Cleanup(func() {
		if svcID != "" {
			_, _ = adminRequest(t, "DELETE", "/api/v1/services/"+svcID, nil)
		}
	})

	// 3. Create Protected Route
	code, epRes := adminRequest(t, "POST", "/api/v1/endpoints", map[string]any{
		"service_id": svcID,
		"method":     "GET",
		"path":       routePath,
		"strip_path": true,
		"is_public":  false,
		"enabled":    true,
	})
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to create endpoint: %d %v", code, epRes)
	}
	epID := extractID(epRes)
	t.Cleanup(func() {
		if epID != "" {
			_, _ = adminRequest(t, "DELETE", "/api/v1/endpoints/"+epID, nil)
		}
	})

	// 4. Attach jwt-auth plugin to Route
	code, pRes := adminRequest(t, "POST", "/api/v1/auth-plugins", map[string]any{
		"plugin_name": "jwt-auth",
		"scope":       "route",
		"route_id":    epID,
		"enabled":     true,
		"config": map[string]any{
			"algorithms": []string{"HS256"},
		},
	})
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to attach jwt-auth plugin: %d %v", code, pRes)
	}
	pID := extractID(pRes)
	t.Cleanup(func() {
		if pID != "" {
			_, _ = adminRequest(t, "DELETE", "/api/v1/auth-plugins/"+pID, nil)
		}
	})

	time.Sleep(200 * time.Millisecond)

	t.Run("Valid Signed JWT Token Accepted", func(t *testing.T) {
		token := signTestJWT(jwt.MapClaims{
			"sub":           username,
			"api_client_id": consumerID,
			"exp":           time.Now().Add(time.Hour).Unix(),
		}, testJWTSecret)

		headers := map[string]string{"Authorization": "Bearer " + token}
		code, _, body := gatewayRequestWithHeaders(t, "GET", routePath, headers, nil)
		if code != http.StatusOK {
			t.Fatalf("expected 200 OK with valid JWT, got %d: %s", code, string(body))
		}
	})

	t.Run("Expired JWT Token Rejected", func(t *testing.T) {
		expiredToken := signTestJWT(jwt.MapClaims{
			"sub":           username,
			"api_client_id": consumerID,
			"exp":           time.Now().Add(-time.Hour).Unix(),
		}, testJWTSecret)

		headers := map[string]string{"Authorization": "Bearer " + expiredToken}
		code, _, _ := gatewayRequestWithHeaders(t, "GET", routePath, headers, nil)
		if code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized with expired JWT, got %d", code)
		}
	})

	t.Run("Tampered Secret JWT Token Rejected", func(t *testing.T) {
		tamperedToken := signTestJWT(jwt.MapClaims{
			"sub":           username,
			"api_client_id": consumerID,
			"exp":           time.Now().Add(time.Hour).Unix(),
		}, "wrong-tampered-secret-key-32-chars!")

		headers := map[string]string{"Authorization": "Bearer " + tamperedToken}
		code, _, _ := gatewayRequestWithHeaders(t, "GET", routePath, headers, nil)
		if code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized with tampered JWT secret, got %d", code)
		}
	})

	t.Run("Missing Authorization Header Rejected", func(t *testing.T) {
		code, _, _ := gatewayRequestWithHeaders(t, "GET", routePath, nil, nil)
		if code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized with missing JWT header, got %d", code)
		}
	})
}

func TestE2E_ServiceScopedAuth_PluginInheritance(t *testing.T) {
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1000000)
	svcName := "svc-scope-svc-" + suffix
	routePath1 := "/api/v1/svc-scope-route1-" + suffix
	routePath2 := "/api/v1/svc-scope-route2-" + suffix
	username := "svc-scope-user-" + suffix

	// 1. Create Consumer & Key
	code, consRes := adminRequest(t, "POST", "/api/v1/consumers", map[string]any{
		"username": username,
		"status":   "active",
	})
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to create consumer: %d %v", code, consRes)
	}
	consumerID := extractID(consRes)
	t.Cleanup(func() {
		if consumerID != "" {
			_, _ = adminRequest(t, "DELETE", "/api/v1/consumers/"+consumerID, nil)
		}
	})

	code, credRes := adminRequest(t, "POST", fmt.Sprintf("/api/v1/consumers/%s/credentials", consumerID), map[string]any{
		"name":     "Service Scope Key",
		"key_type": "api-key",
	})
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to create credential: %d %v", code, credRes)
	}
	apiKey := fmt.Sprintf("%v", credRes["key_secret"])
	if apiKey == "" || apiKey == "<nil>" {
		if d, ok := credRes["data"].(map[string]any); ok {
			apiKey = fmt.Sprintf("%v", d["key_secret"])
		}
	}

	// 2. Create Service
	code, svcRes := adminRequest(t, "POST", "/api/v1/services", map[string]any{
		"name":     svcName,
		"protocol": "http",
		"host":     "127.0.0.1",
		"port":     8100,
		"path":     "/health",
		"enabled":  true,
	})
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to create service: %d %v", code, svcRes)
	}
	svcID := extractID(svcRes)
	t.Cleanup(func() {
		if svcID != "" {
			_, _ = adminRequest(t, "DELETE", "/api/v1/services/"+svcID, nil)
		}
	})

	// 3. Attach key-auth plugin with scope: "service"
	code, pRes := adminRequest(t, "POST", "/api/v1/auth-plugins", map[string]any{
		"plugin_name": "key-auth",
		"scope":       "service",
		"service_id":  svcID,
		"enabled":     true,
		"config":      map[string]any{},
	})
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to attach service-scoped key-auth plugin: %d %v", code, pRes)
	}
	pID := extractID(pRes)
	t.Cleanup(func() {
		if pID != "" {
			_, _ = adminRequest(t, "DELETE", "/api/v1/auth-plugins/"+pID, nil)
		}
	})

	// 4. Create Route 1 under this Service (without any route-level plugin)
	code, ep1Res := adminRequest(t, "POST", "/api/v1/endpoints", map[string]any{
		"service_id": svcID,
		"method":     "GET",
		"path":       routePath1,
		"strip_path": true,
		"is_public":  false,
		"enabled":    true,
	})
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to create route 1: %d %v", code, ep1Res)
	}
	ep1ID := extractID(ep1Res)
	t.Cleanup(func() {
		if ep1ID != "" {
			_, _ = adminRequest(t, "DELETE", "/api/v1/endpoints/"+ep1ID, nil)
		}
	})

	// 5. Create Route 2 under this Service (without any route-level plugin)
	code, ep2Res := adminRequest(t, "POST", "/api/v1/endpoints", map[string]any{
		"service_id": svcID,
		"method":     "GET",
		"path":       routePath2,
		"strip_path": true,
		"is_public":  false,
		"enabled":    true,
	})
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to create route 2: %d %v", code, ep2Res)
	}
	ep2ID := extractID(ep2Res)
	t.Cleanup(func() {
		if ep2ID != "" {
			_, _ = adminRequest(t, "DELETE", "/api/v1/endpoints/"+ep2ID, nil)
		}
	})

	time.Sleep(200 * time.Millisecond)

	// Route 1 checks
	t.Run("Route 1 Unauthenticated Rejected by Service Plugin", func(t *testing.T) {
		code, _, _ := gatewayRequestWithHeaders(t, "GET", routePath1, nil, nil)
		if code != http.StatusUnauthorized {
			t.Fatalf("expected 401 on Route 1 without key, got %d", code)
		}
	})

	t.Run("Route 1 Authenticated Accepted by Inherited Service Plugin", func(t *testing.T) {
		headers := map[string]string{"X-API-Key": apiKey}
		code, _, body := gatewayRequestWithHeaders(t, "GET", routePath1, headers, nil)
		if code != http.StatusOK {
			t.Fatalf("expected 200 on Route 1 with service key, got %d: %s", code, string(body))
		}
	})

	// Route 2 checks
	t.Run("Route 2 Unauthenticated Rejected by Service Plugin", func(t *testing.T) {
		code, _, _ := gatewayRequestWithHeaders(t, "GET", routePath2, nil, nil)
		if code != http.StatusUnauthorized {
			t.Fatalf("expected 401 on Route 2 without key, got %d", code)
		}
	})

	t.Run("Route 2 Authenticated Accepted by Inherited Service Plugin", func(t *testing.T) {
		headers := map[string]string{"X-API-Key": apiKey}
		code, _, body := gatewayRequestWithHeaders(t, "GET", routePath2, headers, nil)
		if code != http.StatusOK {
			t.Fatalf("expected 200 on Route 2 with service key, got %d: %s", code, string(body))
		}
	})
}

func TestE2E_BasicAuth_MalformedAndEdgeCases(t *testing.T) {
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1000000)
	svcName := "basic-edge-svc-" + suffix
	routePath := "/api/v1/basic-edge-" + suffix
	username := "basic-edge-user-" + suffix
	password := "GoodPassword2026!"

	// 1. Create Consumer with password
	code, consRes := adminRequest(t, "POST", "/api/v1/consumers", map[string]any{
		"username": username,
		"status":   "active",
		"password": password,
	})
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to create consumer: %d %v", code, consRes)
	}
	consumerID := extractID(consRes)
	t.Cleanup(func() {
		if consumerID != "" {
			_, _ = adminRequest(t, "DELETE", "/api/v1/consumers/"+consumerID, nil)
		}
	})

	// 2. Create Service
	code, svcRes := adminRequest(t, "POST", "/api/v1/services", map[string]any{
		"name":     svcName,
		"protocol": "http",
		"host":     "127.0.0.1",
		"port":     8100,
		"path":     "/health",
		"enabled":  true,
	})
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to create service: %d %v", code, svcRes)
	}
	svcID := extractID(svcRes)
	t.Cleanup(func() {
		if svcID != "" {
			_, _ = adminRequest(t, "DELETE", "/api/v1/services/"+svcID, nil)
		}
	})

	// 3. Create Protected Route
	code, epRes := adminRequest(t, "POST", "/api/v1/endpoints", map[string]any{
		"service_id": svcID,
		"method":     "GET",
		"path":       routePath,
		"strip_path": true,
		"is_public":  false,
		"enabled":    true,
	})
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to create endpoint: %d %v", code, epRes)
	}
	epID := extractID(epRes)
	t.Cleanup(func() {
		if epID != "" {
			_, _ = adminRequest(t, "DELETE", "/api/v1/endpoints/"+epID, nil)
		}
	})

	// 4. Attach basic-auth plugin to Route
	code, pRes := adminRequest(t, "POST", "/api/v1/auth-plugins", map[string]any{
		"plugin_name": "basic-auth",
		"scope":       "route",
		"route_id":    epID,
		"enabled":     true,
		"config":      map[string]any{},
	})
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to attach basic-auth plugin: %d %v", code, pRes)
	}
	pID := extractID(pRes)
	t.Cleanup(func() {
		if pID != "" {
			_, _ = adminRequest(t, "DELETE", "/api/v1/auth-plugins/"+pID, nil)
		}
	})

	time.Sleep(200 * time.Millisecond)

	t.Run("Valid Basic Auth Succeeded", func(t *testing.T) {
		validAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))
		headers := map[string]string{"Authorization": validAuth}
		code, _, body := gatewayRequestWithHeaders(t, "GET", routePath, headers, nil)
		if code != http.StatusOK {
			t.Fatalf("expected 200 OK with valid basic auth, got %d: %s", code, string(body))
		}
	})

	t.Run("Invalid Password Rejected", func(t *testing.T) {
		badAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":WrongPass123!"))
		headers := map[string]string{"Authorization": badAuth}
		code, _, _ := gatewayRequestWithHeaders(t, "GET", routePath, headers, nil)
		if code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized with bad password, got %d", code)
		}
	})

	t.Run("Malformed Base64 Header Rejected", func(t *testing.T) {
		headers := map[string]string{"Authorization": "Basic !!!not-valid-base64!!!"}
		code, _, _ := gatewayRequestWithHeaders(t, "GET", routePath, headers, nil)
		if code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized with invalid base64, got %d", code)
		}
	})

	t.Run("Missing Colon In Decoded Credentials Rejected", func(t *testing.T) {
		noColonAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("justusername"))
		headers := map[string]string{"Authorization": noColonAuth}
		code, _, _ := gatewayRequestWithHeaders(t, "GET", routePath, headers, nil)
		if code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized without colon, got %d", code)
		}
	})
}

func TestE2E_ConsumerRouteAccessRestrictions_RBAC(t *testing.T) {
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1000000)
	svcName := "rbac-svc-" + suffix
	routePathA := "/api/v1/rbac-a-" + suffix
	routePathB := "/api/v1/rbac-b-" + suffix
	username := "rbac-restricted-user-" + suffix

	// 1. Create Service
	code, svcRes := adminRequest(t, "POST", "/api/v1/services", map[string]any{
		"name":     svcName,
		"protocol": "http",
		"host":     "127.0.0.1",
		"port":     8100,
		"path":     "/health",
		"enabled":  true,
	})
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to create service: %d %v", code, svcRes)
	}
	svcID := extractID(svcRes)
	t.Cleanup(func() {
		if svcID != "" {
			_, _ = adminRequest(t, "DELETE", "/api/v1/services/"+svcID, nil)
		}
	})

	// 2. Create Route A (protected)
	code, epARes := adminRequest(t, "POST", "/api/v1/endpoints", map[string]any{
		"service_id": svcID,
		"method":     "GET",
		"path":       routePathA,
		"strip_path": true,
		"is_public":  false,
		"enabled":    true,
	})
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to create route A: %d %v", code, epARes)
	}
	epAID := extractID(epARes)
	t.Cleanup(func() {
		if epAID != "" {
			_, _ = adminRequest(t, "DELETE", "/api/v1/endpoints/"+epAID, nil)
		}
	})

	// 3. Create Route B (protected)
	code, epBRes := adminRequest(t, "POST", "/api/v1/endpoints", map[string]any{
		"service_id": svcID,
		"method":     "GET",
		"path":       routePathB,
		"strip_path": true,
		"is_public":  false,
		"enabled":    true,
	})
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to create route B: %d %v", code, epBRes)
	}
	epBID := extractID(epBRes)
	t.Cleanup(func() {
		if epBID != "" {
			_, _ = adminRequest(t, "DELETE", "/api/v1/endpoints/"+epBID, nil)
		}
	})

	// 4. Create Consumer Restricted to Route A ONLY
	code, consRes := adminRequest(t, "POST", "/api/v1/consumers", map[string]any{
		"username":     username,
		"status":       "active",
		"endpoint_ids": []string{epAID},
	})
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to create consumer: %d %v", code, consRes)
	}
	consumerID := extractID(consRes)
	t.Cleanup(func() {
		if consumerID != "" {
			_, _ = adminRequest(t, "DELETE", "/api/v1/consumers/"+consumerID, nil)
		}
	})

	// 5. Create API key for Consumer
	code, credRes := adminRequest(t, "POST", fmt.Sprintf("/api/v1/consumers/%s/credentials", consumerID), map[string]any{
		"name":     "RBAC Key",
		"key_type": "api-key",
	})
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to create credential: %d %v", code, credRes)
	}
	apiKey := fmt.Sprintf("%v", credRes["key_secret"])
	if apiKey == "" || apiKey == "<nil>" {
		if d, ok := credRes["data"].(map[string]any); ok {
			apiKey = fmt.Sprintf("%v", d["key_secret"])
		}
	}

	time.Sleep(200 * time.Millisecond)

	headers := map[string]string{"X-API-Key": apiKey}

	t.Run("Access to Authorized Route A Permitted (200 OK)", func(t *testing.T) {
		code, _, body := gatewayRequestWithHeaders(t, "GET", routePathA, headers, nil)
		if code != http.StatusOK {
			t.Fatalf("expected 200 OK on authorized route A, got %d: %s", code, string(body))
		}
	})

	t.Run("Access to Unauthorized Route B Forbidden (403 Forbidden)", func(t *testing.T) {
		code, _, body := gatewayRequestWithHeaders(t, "GET", routePathB, headers, nil)
		if code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden on unauthorized route B, got %d: %s", code, string(body))
		}
		if !strings.Contains(string(body), "FORBIDDEN") && !strings.Contains(string(body), "forbidden") {
			t.Errorf("expected forbidden message in error response, got %s", string(body))
		}
	})
}

func TestE2E_PublicAndDisabledRouteLifecycle(t *testing.T) {
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1000000)
	svcName := "pub-dis-svc-" + suffix
	publicRoutePath := "/api/v1/public-test-" + suffix
	disabledRoutePath := "/api/v1/disabled-test-" + suffix

	// 1. Create Service
	code, svcRes := adminRequest(t, "POST", "/api/v1/services", map[string]any{
		"name":     svcName,
		"protocol": "http",
		"host":     "127.0.0.1",
		"port":     8100,
		"path":     "/health",
		"enabled":  true,
	})
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to create service: %d %v", code, svcRes)
	}
	svcID := extractID(svcRes)
	t.Cleanup(func() {
		if svcID != "" {
			_, _ = adminRequest(t, "DELETE", "/api/v1/services/"+svcID, nil)
		}
	})

	// 2. Create Public Route
	code, pubRes := adminRequest(t, "POST", "/api/v1/endpoints", map[string]any{
		"service_id": svcID,
		"method":     "GET",
		"path":       publicRoutePath,
		"strip_path": true,
		"is_public":  true, // Public!
		"enabled":    true,
	})
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to create public endpoint: %d %v", code, pubRes)
	}
	pubID := extractID(pubRes)
	t.Cleanup(func() {
		if pubID != "" {
			_, _ = adminRequest(t, "DELETE", "/api/v1/endpoints/"+pubID, nil)
		}
	})

	// 3. Create Disabled Route
	code, disRes := adminRequest(t, "POST", "/api/v1/endpoints", map[string]any{
		"service_id": svcID,
		"method":     "GET",
		"path":       disabledRoutePath,
		"strip_path": true,
		"is_public":  true,
		"enabled":    false, // Disabled!
	})
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to create disabled endpoint: %d %v", code, disRes)
	}
	disID := extractID(disRes)
	t.Cleanup(func() {
		if disID != "" {
			_, _ = adminRequest(t, "DELETE", "/api/v1/endpoints/"+disID, nil)
		}
	})

	time.Sleep(200 * time.Millisecond)

	t.Run("Public Route Without Any Credentials Allowed (200 OK)", func(t *testing.T) {
		code, _, body := gatewayRequestWithHeaders(t, "GET", publicRoutePath, nil, nil)
		if code != http.StatusOK {
			t.Fatalf("expected 200 OK for public route without auth, got %d: %s", code, string(body))
		}
	})

	t.Run("Disabled Route Inaccessible (404 Not Found)", func(t *testing.T) {
		code, _, body := gatewayRequestWithHeaders(t, "GET", disabledRoutePath, nil, nil)
		if code != http.StatusNotFound {
			t.Fatalf("expected 404 Not Found for disabled route, got %d: %s", code, string(body))
		}
	})
}

func TestE2E_SecurityGuardrails_IPAndDomainRestrictions(t *testing.T) {
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1000000)
	svcName := "guardrails-svc-" + suffix
	ipRoutePath := "/api/v1/ip-restricted-" + suffix
	domainRoutePath := "/api/v1/domain-restricted-" + suffix

	// 1. Create Service
	code, svcRes := adminRequest(t, "POST", "/api/v1/services", map[string]any{
		"name":     svcName,
		"protocol": "http",
		"host":     "127.0.0.1",
		"port":     8100,
		"path":     "/health",
		"enabled":  true,
	})
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to create service: %d %v", code, svcRes)
	}
	svcID := extractID(svcRes)
	t.Cleanup(func() {
		if svcID != "" {
			_, _ = adminRequest(t, "DELETE", "/api/v1/services/"+svcID, nil)
		}
	})

	// 2. Create IP-Restricted Route (only allowed from non-local 10.254.254.1/32)
	code, ipRes := adminRequest(t, "POST", "/api/v1/endpoints", map[string]any{
		"service_id":  svcID,
		"method":      "GET",
		"path":        ipRoutePath,
		"strip_path":  true,
		"is_public":   true,
		"allowed_ips": "10.254.254.1/32", // Blocks loopback
		"enabled":     true,
	})
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to create IP-restricted endpoint: %d %v", code, ipRes)
	}
	ipID := extractID(ipRes)
	t.Cleanup(func() {
		if ipID != "" {
			_, _ = adminRequest(t, "DELETE", "/api/v1/endpoints/"+ipID, nil)
		}
	})

	// 3. Create Domain-Restricted Route (only allowed for specific virtual host)
	code, domRes := adminRequest(t, "POST", "/api/v1/endpoints", map[string]any{
		"service_id":      svcID,
		"method":          "GET",
		"path":            domainRoutePath,
		"strip_path":      true,
		"is_public":       true,
		"allowed_domains": "secure.internal.org",
		"enabled":         true,
	})
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to create domain-restricted endpoint: %d %v", code, domRes)
	}
	domID := extractID(domRes)
	t.Cleanup(func() {
		if domID != "" {
			_, _ = adminRequest(t, "DELETE", "/api/v1/endpoints/"+domID, nil)
		}
	})

	time.Sleep(200 * time.Millisecond)

	t.Run("Disallowed Client IP Rejected with 403 Forbidden", func(t *testing.T) {
		code, _, body := gatewayRequestWithHeaders(t, "GET", ipRoutePath, nil, nil)
		if code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for disallowed IP, got %d: %s", code, string(body))
		}
		if !strings.Contains(string(body), "IP address forbidden") {
			t.Errorf("expected IP address forbidden error message, got %s", string(body))
		}
	})

	t.Run("Disallowed Host Rejected with 403 Forbidden", func(t *testing.T) {
		// Default request host is localhost or 127.0.0.1
		code, _, body := gatewayRequestWithHeaders(t, "GET", domainRoutePath, nil, nil)
		if code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for default host, got %d: %s", code, string(body))
		}
		if !strings.Contains(string(body), "Domain not authorized") {
			t.Errorf("expected Domain not authorized message, got %s", string(body))
		}
	})

	t.Run("Authorized Host Accepted with 200 OK", func(t *testing.T) {
		headers := map[string]string{"Host": "secure.internal.org"}
		code, _, body := gatewayRequestWithHeaders(t, "GET", domainRoutePath, headers, nil)
		if code != http.StatusOK {
			t.Fatalf("expected 200 OK with authorized Host, got %d: %s", code, string(body))
		}
	})
}

func TestE2E_CORSPreflightAndOrigins(t *testing.T) {
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1000000)
	svcName := "cors-svc-" + suffix
	routePath := "/api/v1/cors-test-" + suffix
	allowedOrigin := "https://client-portal.example.com"

	// 1. Create Service
	code, svcRes := adminRequest(t, "POST", "/api/v1/services", map[string]any{
		"name":     svcName,
		"protocol": "http",
		"host":     "127.0.0.1",
		"port":     8100,
		"path":     "/health",
		"enabled":  true,
	})
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to create service: %d %v", code, svcRes)
	}
	svcID := extractID(svcRes)
	t.Cleanup(func() {
		if svcID != "" {
			_, _ = adminRequest(t, "DELETE", "/api/v1/services/"+svcID, nil)
		}
	})

	// 2. Create Route with CORS origins
	code, epRes := adminRequest(t, "POST", "/api/v1/endpoints", map[string]any{
		"service_id":   svcID,
		"method":       "GET",
		"path":         routePath,
		"strip_path":   true,
		"is_public":    true,
		"cors_origins": allowedOrigin,
		"enabled":      true,
	})
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to create endpoint: %d %v", code, epRes)
	}
	epID := extractID(epRes)
	t.Cleanup(func() {
		if epID != "" {
			_, _ = adminRequest(t, "DELETE", "/api/v1/endpoints/"+epID, nil)
		}
	})

	time.Sleep(200 * time.Millisecond)

	t.Run("Preflight OPTIONS Request Returns 204 with CORS Headers", func(t *testing.T) {
		headers := map[string]string{
			"Origin":                        allowedOrigin,
			"Access-Control-Request-Method": "GET",
		}
		code, respHeaders, _ := gatewayRequestWithHeaders(t, "OPTIONS", routePath, headers, nil)
		if code != http.StatusNoContent {
			t.Fatalf("expected 204 No Content on preflight OPTIONS, got %d", code)
		}
		if got := respHeaders.Get("Access-Control-Allow-Origin"); got == "" {
			t.Errorf("expected non-empty Access-Control-Allow-Origin header")
		}
		if got := respHeaders.Get("Access-Control-Allow-Methods"); !strings.Contains(got, "GET") {
			t.Errorf("expected Access-Control-Allow-Methods to contain GET, got %s", got)
		}
	})

	t.Run("GET Request with Origin Header Receives CORS Allow-Origin", func(t *testing.T) {
		headers := map[string]string{
			"Origin": allowedOrigin,
		}
		code, respHeaders, body := gatewayRequestWithHeaders(t, "GET", routePath, headers, nil)
		if code != http.StatusOK {
			t.Fatalf("expected 200 OK on GET with Origin, got %d: %s", code, string(body))
		}
		if got := respHeaders.Get("Access-Control-Allow-Origin"); got == "" {
			t.Errorf("expected Access-Control-Allow-Origin header on GET response")
		}
	})
}
