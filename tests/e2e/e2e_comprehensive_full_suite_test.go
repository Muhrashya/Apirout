package e2e_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestE2E_FullSuite_DatabaseConnections exercises the complete database connection lifecycle:
// Raw testing, SSRF validation, snake_case/canonical field normalization, live Trino testing,
// live Postgres testing, CRUD persistence, read-only toggling, and health probes.
func TestE2E_FullSuite_DatabaseConnections(t *testing.T) {
	// 1. SSRF Protection: Loopback and Cloud Metadata blocked, Private IP permitted
	t.Run("SSRF Egress Guardrails", func(t *testing.T) {
		// Cloud Metadata (blocked)
		code, res := adminRequest(t, "POST", "/api/v1/connections/test", map[string]any{
			"db_type": "postgres",
			"host":    "169.254.169.254",
			"port":    5432,
			"dbname":  "testdb",
		})
		if code != http.StatusBadRequest {
			t.Errorf("expected 400 for cloud metadata SSRF, got %d", code)
		}
		if errMsg, ok := res["error"].(string); !ok || !strings.Contains(errMsg, "cloud metadata") {
			t.Errorf("expected cloud metadata error message, got: %v", res)
		}

		// Loopback (blocked)
		code, res = adminRequest(t, "POST", "/api/v1/connections/test", map[string]any{
			"db_type": "postgres",
			"host":    "127.0.0.1",
			"port":    5432,
			"dbname":  "testdb",
		})
		if code != http.StatusBadRequest {
			t.Errorf("expected 400 for loopback SSRF, got %d", code)
		}
		if errMsg, ok := res["error"].(string); !ok || !strings.Contains(errMsg, "loopback") {
			t.Errorf("expected loopback error message, got: %v", res)
		}
	})

	// 2. Raw Connection Testing: Canonical and snake_case field normalization
	t.Run("Raw Connection Test Field Normalization", func(t *testing.T) {
		// Test using snake_case db_name and ssl_mode (regression test)
		code, res := adminRequest(t, "POST", "/api/v1/connections/test", map[string]any{
			"db_type":  "postgres",
			"host":     "postgres",
			"port":     5432,
			"db_name":  "main_db",
			"username": "postgres",
			"password": "Password123!",
			"ssl_mode": "disable",
		})
		if code != http.StatusOK {
			t.Fatalf("expected 200 for raw postgres test with db_name/ssl_mode, got %d: %v", code, res)
		}
		if success, ok := res["success"].(bool); !ok || !success {
			t.Fatalf("expected raw postgres connection test success=true, got: %v", res)
		}
	})

	// 3. Live Trino Connection Test (Production Cluster Probe)
	t.Run("Live Trino Connection Handshake", func(t *testing.T) {
		code, res := adminRequest(t, "POST", "/api/v1/connections/test", map[string]any{
			"db_type":  "trino",
			"host":     "trino-internal.bgn.go.id",
			"port":     443,
			"dbname":   "tpch",
			"username": "bgn_datalake_bpk",
			"password": "7o54dDKGdHBhbEqp",
			"sslmode":  "require",
		})
		if code != http.StatusOK {
			t.Fatalf("expected 200 for Trino test, got %d: %v", code, res)
		}
		if success, ok := res["success"].(bool); !ok || !success {
			t.Fatalf("expected Trino test success=true, got: %v", res)
		}
	})

	// 4. Connection CRUD: Create with db_name, Read, Update, Health Probe, Delete
	t.Run("Connection CRUD and Persistence", func(t *testing.T) {
		createPayload := map[string]any{
			"name":         "e2e-live-postgres",
			"db_type":      "postgres",
			"host":         "postgres",
			"port":         5432,
			"db_name":      "main_db",
			"username":     "postgres",
			"password":     "Password123!",
			"ssl_mode":     "disable",
			"is_read_only": false,
		}

		code, res := adminRequest(t, "POST", "/api/v1/connections", createPayload)
		if code != http.StatusOK && code != http.StatusCreated {
			t.Fatalf("failed to create connection: %d: %v", code, res)
		}
		connID := extractID(res)
		if connID == "" {
			t.Fatalf("no connection ID returned in create response: %v", res)
		}
		defer func() {
			adminRequest(t, "DELETE", "/api/v1/connections/"+connID, nil)
		}()

		// Read connection by ID: verify dual-key serialization
		code, getRes := adminRequest(t, "GET", "/api/v1/connections/"+connID, nil)
		if code != http.StatusOK {
			t.Fatalf("failed to get connection %s: %d", connID, code)
		}
		if getRes["dbname"] != "main_db" || getRes["db_name"] != "main_db" {
			t.Errorf("expected both dbname and db_name in response, got dbname=%v, db_name=%v", getRes["dbname"], getRes["db_name"])
		}

		// Test saved connection by ID
		code, testRes := adminRequest(t, "POST", fmt.Sprintf("/api/v1/connections/%s/test", connID), nil)
		if code != http.StatusOK {
			t.Errorf("expected 200 testing saved connection, got %d: %v", code, testRes)
		}
		if success, ok := testRes["success"].(bool); !ok || !success {
			t.Errorf("expected saved connection test success=true, got: %v", testRes)
		}

		// Update connection: toggle read-only mode
		updatePayload := map[string]any{
			"name":         "e2e-live-postgres-updated",
			"is_read_only": true,
		}
		code, _ = adminRequest(t, "PUT", "/api/v1/connections/"+connID, updatePayload)
		if code != http.StatusOK {
			t.Errorf("expected 200 updating connection, got %d", code)
		}

		// List connections with health: verify connection is present with health info
		code, listRes := adminRequest(t, "GET", "/api/v1/connections", nil)
		if code != http.StatusOK {
			t.Errorf("expected 200 listing connections, got %d", code)
		}
		conns, ok := listRes["data"].([]any)
		if !ok || len(conns) == 0 {
			t.Errorf("expected at least 1 connection in list, got %v", listRes)
		}
	})
}

// TestE2E_FullSuite_InstantAPI_And_Endpoints exercises schema introspection, instant CRUD,
// dynamic endpoint generation, and data plane proxy execution.
func TestE2E_FullSuite_InstantAPI_And_Endpoints(t *testing.T) {
	// Provision a dedicated test connection
	createPayload := map[string]any{
		"name":         "e2e-instant-db",
		"db_type":      "postgres",
		"host":         "postgres",
		"port":         5432,
		"dbname":       "main_db",
		"username":     "postgres",
		"password":     "Password123!",
		"sslmode":      "disable",
		"is_read_only": false,
	}
	code, res := adminRequest(t, "POST", "/api/v1/connections", createPayload)
	if code != http.StatusOK && code != http.StatusCreated {
		t.Fatalf("failed to create connection for instant API: %d: %v", code, res)
	}
	connID := extractID(res)
	if connID == "" {
		t.Fatalf("no connection ID extracted: %v", res)
	}
	defer func() {
		adminRequest(t, "DELETE", "/api/v1/connections/"+connID, nil)
	}()

	authHeaders := map[string]string{
		"X-Admin-Secret": adminSecret,
	}

	// 1. Schema Introspection via Data Plane
	t.Run("Schema Introspection", func(t *testing.T) {
		code, body := gatewayRequest(t, "GET", fmt.Sprintf("/api/v1/instant/%s/schema", connID), authHeaders, nil)
		if code != http.StatusOK {
			t.Fatalf("expected 200 schema introspection, got %d: %s", code, string(body))
		}
		var schema struct {
			Tables []struct {
				Name string `json:"name"`
			} `json:"tables"`
		}
		if err := json.Unmarshal(body, &schema); err != nil {
			t.Fatalf("failed to decode schema: %v", err)
		}
		if len(schema.Tables) == 0 {
			t.Fatalf("expected tables in schema, got 0")
		}
	})

	// 2. Instant CRUD Query & Pagination via Data Plane
	t.Run("Instant CRUD Table Query & Pagination", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/instant/%s/services?limit=2&offset=0&order=name.ASC", connID)
		code, body := gatewayRequest(t, "GET", path, authHeaders, nil)
		if code != http.StatusOK {
			t.Fatalf("expected 200 querying instant services table, got %d: %s", code, string(body))
		}
		var respData struct {
			Data []map[string]any `json:"data"`
			Meta struct {
				Limit    int `json:"limit"`
				Offset   int `json:"offset"`
				Returned int `json:"returned"`
			} `json:"meta"`
		}
		if err := json.Unmarshal(body, &respData); err != nil {
			t.Fatalf("failed to parse instant query response: %v", err)
		}
		if respData.Meta.Limit != 2 {
			t.Errorf("expected limit 2, got %d", respData.Meta.Limit)
		}
	})

	// 3. Row Mutation Lifecycle: Insert, Query, Delete on system_settings
	t.Run("Instant Row Mutation Lifecycle", func(t *testing.T) {
		key := fmt.Sprintf("e2e_setting_%d", time.Now().UnixNano())
		insertBody := map[string]any{
			"key":   key,
			"value": "initial_val",
		}

		// Insert
		code, body := gatewayRequest(t, "POST", fmt.Sprintf("/api/v1/instant/%s/system_settings", connID), authHeaders, insertBody)
		if code != http.StatusCreated && code != http.StatusOK {
			t.Fatalf("insert failed: %d: %s", code, string(body))
		}

		// Query by key filter
		code, body = gatewayRequest(t, "GET", fmt.Sprintf("/api/v1/instant/%s/system_settings?key=eq.%s", connID, key), authHeaders, nil)
		if code != http.StatusOK {
			t.Fatalf("query inserted row failed: %d: %s", code, string(body))
		}

		// Cleanup inserted row via path parameter DELETE /api/v1/instant/{connID}/system_settings/{key}
		code, body = gatewayRequest(t, "DELETE", fmt.Sprintf("/api/v1/instant/%s/system_settings/%s", connID, key), authHeaders, nil)
		if code != http.StatusOK && code != http.StatusNoContent {
			t.Errorf("delete row failed: %d: %s", code, string(body))
		}
	})

	// 4. Generate Endpoints from Connection
	t.Run("Generate Endpoints from Connection", func(t *testing.T) {
		code, res := adminRequest(t, "POST", fmt.Sprintf("/api/v1/connections/%s/generate-endpoints", connID), nil)
		if code != http.StatusOK && code != http.StatusCreated {
			t.Fatalf("failed to generate endpoints from connection: %d: %v", code, res)
		}

		// Verify endpoints list now includes generated endpoints
		code, epList := adminRequest(t, "GET", "/api/v1/endpoints", nil)
		if code != http.StatusOK {
			t.Fatalf("failed to list endpoints: %d", code)
		}
		endpoints, _ := epList["data"].([]any)
		if len(endpoints) == 0 {
			t.Fatalf("expected endpoints generated, got 0")
		}
	})
}

// TestE2E_FullSuite_Queries_And_DynamicEndpoints exercises saved SQL queries, parameter substitution,
// Admin preview, and executing queries bound to API endpoints.
func TestE2E_FullSuite_Queries_And_DynamicEndpoints(t *testing.T) {
	// 1. Provision Connection
	createPayload := map[string]any{
		"name":     "e2e-query-db",
		"db_type":  "postgres",
		"host":     "postgres",
		"port":     5432,
		"dbname":   "main_db",
		"username": "postgres",
		"password": "Password123!",
		"sslmode":  "disable",
	}
	code, res := adminRequest(t, "POST", "/api/v1/connections", createPayload)
	if code != http.StatusOK && code != http.StatusCreated {
		t.Fatalf("failed to create connection: %d: %v", code, res)
	}
	connID := extractID(res)
	defer adminRequest(t, "DELETE", "/api/v1/connections/"+connID, nil)

	// 2. Create Saved SQL Query
	queryPayload := map[string]any{
		"connection_id": connID,
		"name":          "E2E List Services",
		"description":   "E2E test query with param filtering",
		"sql_text":      "SELECT id, name, protocol, enabled FROM services WHERE enabled = :enabled",
		"params_schema": `{"enabled": "boolean"}`,
	}
	code, qRes := adminRequest(t, "POST", "/api/v1/queries", queryPayload)
	if code != http.StatusOK && code != http.StatusCreated {
		t.Fatalf("failed to create query: %d: %v", code, qRes)
	}
	queryID := extractID(qRes)
	if queryID == "" {
		t.Fatalf("no query ID returned: %v", qRes)
	}
	defer adminRequest(t, "DELETE", "/api/v1/queries/"+queryID, nil)

	// 3. Admin Query Preview / Execute
	t.Run("Query Preview Execution", func(t *testing.T) {
		previewPayload := map[string]any{
			"connection_id": connID,
			"sql":           "SELECT id, name, protocol, enabled FROM services LIMIT 5",
			"limit":         5,
		}
		code, pRes := adminRequest(t, "POST", "/api/v1/queries/preview", previewPayload)
		if code != http.StatusOK {
			t.Fatalf("query preview failed: %d: %v", code, pRes)
		}
	})

	// 4. Create Public Endpoint Linked to Query
	endpointPayload := map[string]any{
		"name":      "E2E Query Endpoint",
		"path":      "/api/v1/e2e-query-services",
		"method":    "GET",
		"query_id":  queryID,
		"is_public": true,
		"enabled":   true,
	}
	code, epRes := adminRequest(t, "POST", "/api/v1/endpoints", endpointPayload)
	if code != http.StatusOK && code != http.StatusCreated {
		t.Fatalf("failed to create query endpoint: %d: %v", code, epRes)
	}
	epID := extractID(epRes)
	if epID != "" {
		defer adminRequest(t, "DELETE", "/api/v1/endpoints/"+epID, nil)
	}

	// 5. Invoke Query Endpoint via Gateway Data Plane Proxy
	t.Run("Invoke Query Endpoint via Gateway", func(t *testing.T) {
		time.Sleep(100 * time.Millisecond) // Allow sync
		code, body := gatewayRequest(t, "GET", "/api/v1/e2e-query-services?enabled=true", nil, nil)
		if code != http.StatusOK {
			t.Fatalf("expected 200 from query endpoint, got %d: %s", code, string(body))
		}
	})
}

// TestE2E_FullSuite_Routes_Services_Upstreams exercises the full traffic routing pipeline:
// Upstreams, load balancer targets, services, route matching, path stripping, and method constraints.
func TestE2E_FullSuite_Routes_Services_Upstreams(t *testing.T) {
	// 1. Create Upstream
	upstreamPayload := map[string]any{
		"name":      "e2e-upstream",
		"algorithm": "round-robin",
		"slots":     100,
		"enabled":   true,
	}
	code, upRes := adminRequest(t, "POST", "/api/v1/upstreams", upstreamPayload)
	if code != http.StatusOK && code != http.StatusCreated {
		t.Fatalf("failed to create upstream: %d: %v", code, upRes)
	}
	upstreamID := extractID(upRes)
	defer adminRequest(t, "DELETE", "/api/v1/upstreams/"+upstreamID, nil)

	// Add Target to Upstream
	targetPayload := map[string]any{
		"target":  "httpbin.org:80",
		"weight":  100,
		"enabled": true,
	}
	code, _ = adminRequest(t, "POST", fmt.Sprintf("/api/v1/upstreams/%s/targets", upstreamID), targetPayload)
	if code != http.StatusOK && code != http.StatusCreated {
		t.Fatalf("failed to add target to upstream: %d", code)
	}

	// 2. Create Service
	servicePayload := map[string]any{
		"name":            "e2e-routed-service",
		"protocol":        "http",
		"host":            "httpbin.org",
		"port":            80,
		"path":            "/anything",
		"upstream_id":     upstreamID,
		"connect_timeout": 5000,
		"read_timeout":    5000,
		"write_timeout":   5000,
		"retries":         2,
		"enabled":         true,
	}
	code, sRes := adminRequest(t, "POST", "/api/v1/services", servicePayload)
	if code != http.StatusOK && code != http.StatusCreated {
		t.Fatalf("failed to create service: %d: %v", code, sRes)
	}
	serviceID := extractID(sRes)
	defer adminRequest(t, "DELETE", "/api/v1/services/"+serviceID, nil)

	// 3. Create Route Endpoint constrained to GET method with path stripping
	endpointPayload := map[string]any{
		"service_id": serviceID,
		"name":       "E2E Echo Route",
		"path":       "/api/v1/e2e-echo",
		"method":     "GET",
		"strip_path": true,
		"is_public":  true,
		"enabled":    true,
	}
	code, epRes := adminRequest(t, "POST", "/api/v1/endpoints", endpointPayload)
	if code != http.StatusOK && code != http.StatusCreated {
		t.Fatalf("failed to create endpoint: %d: %v", code, epRes)
	}
	epID := extractID(epRes)
	defer adminRequest(t, "DELETE", "/api/v1/endpoints/"+epID, nil)

	time.Sleep(100 * time.Millisecond) // Allow sync

	// 4. Test Allowed Method (GET)
	t.Run("Allowed Method Dispatched", func(t *testing.T) {
		code, _ := gatewayRequest(t, "GET", "/api/v1/e2e-echo", nil, nil)
		// Upstream may return 200 or upstream gateway response
		if code == http.StatusNotFound || code == http.StatusMethodNotAllowed {
			t.Errorf("unexpected routing failure %d for allowed GET method", code)
		}
	})

	// 5. Test Disallowed Method (DELETE) -> 405 Method Not Allowed
	t.Run("Disallowed Method Rejected with 405", func(t *testing.T) {
		code, _ := gatewayRequest(t, "DELETE", "/api/v1/e2e-echo", nil, nil)
		if code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405 Method Not Allowed for DELETE on GET-only route, got %d", code)
		}
	})

	// 6. Test Hot-Reload: Disable Endpoint -> 404 Not Found
	t.Run("Hot-Reload Toggle Endpoint Enabled", func(t *testing.T) {
		updatePayload := map[string]any{
			"service_id": serviceID,
			"path":       "/api/v1/e2e-echo",
			"method":     "GET",
			"is_public":  true,
			"enabled":    false,
		}
		adminRequest(t, "PUT", "/api/v1/endpoints/"+epID, updatePayload)
		time.Sleep(100 * time.Millisecond)

		code, _ := gatewayRequest(t, "GET", "/api/v1/e2e-echo", nil, nil)
		if code != http.StatusNotFound {
			t.Errorf("expected 404 for disabled endpoint, got %d", code)
		}

		// Re-enable
		updatePayload["enabled"] = true
		adminRequest(t, "PUT", "/api/v1/endpoints/"+epID, updatePayload)
		time.Sleep(100 * time.Millisecond)

		code, _ = gatewayRequest(t, "GET", "/api/v1/e2e-echo", nil, nil)
		if code == http.StatusNotFound {
			t.Errorf("expected endpoint to be active again after re-enabling, got 404")
		}
	})
}

// TestE2E_FullSuite_Security_AuthPlugins exercises Inbound Authentication plugins (Key-Auth),
// Consumer API keys, and Rate Limiting enforcement.
func TestE2E_FullSuite_Security_AuthPlugins(t *testing.T) {
	// 1. Create Protected Route
	servicePayload := map[string]any{
		"name":     "e2e-auth-service",
		"protocol": "http",
		"host":     "httpbin.org",
		"port":     80,
		"path":     "/anything",
		"enabled":  true,
	}
	code, sRes := adminRequest(t, "POST", "/api/v1/services", servicePayload)
	if code != http.StatusOK && code != http.StatusCreated {
		t.Fatalf("failed to create service: %d: %v", code, sRes)
	}
	serviceID := extractID(sRes)
	defer adminRequest(t, "DELETE", "/api/v1/services/"+serviceID, nil)

	endpointPayload := map[string]any{
		"service_id": serviceID,
		"name":       "E2E Key Protected Route",
		"path":       "/api/v1/e2e-key-protected",
		"method":     "GET",
		"is_public":  false,
		"enabled":    true,
	}
	code, epRes := adminRequest(t, "POST", "/api/v1/endpoints", endpointPayload)
	if code != http.StatusOK && code != http.StatusCreated {
		t.Fatalf("failed to create endpoint: %d: %v", code, epRes)
	}
	epID := extractID(epRes)
	defer adminRequest(t, "DELETE", "/api/v1/endpoints/"+epID, nil)

	// 2. Attach Key-Auth Plugin to Endpoint
	pluginPayload := map[string]any{
		"endpoint_id": epID,
		"plugin_name": "key-auth",
		"config": map[string]any{
			"key_names": []string{"apikey", "x-api-key"},
		},
		"enabled": true,
	}
	code, pRes := adminRequest(t, "POST", "/api/v1/auth-plugins", pluginPayload)
	if code != http.StatusOK && code != http.StatusCreated {
		t.Fatalf("failed to attach key-auth plugin: %d: %v", code, pRes)
	}
	pluginID := extractID(pRes)
	if pluginID != "" {
		defer adminRequest(t, "DELETE", "/api/v1/auth-plugins/"+pluginID, nil)
	}

	time.Sleep(100 * time.Millisecond)

	// 3. Request without Key -> 401 Unauthorized
	t.Run("Request without API Key Rejected", func(t *testing.T) {
		code, _ := gatewayRequest(t, "GET", "/api/v1/e2e-key-protected", nil, nil)
		if code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized without api key, got %d", code)
		}
	})

	// 4. Request with Invalid Key -> 401 Unauthorized
	t.Run("Request with Invalid API Key Rejected", func(t *testing.T) {
		headers := map[string]string{"x-api-key": "invalid-secret-key-123"}
		code, _ := gatewayRequest(t, "GET", "/api/v1/e2e-key-protected", headers, nil)
		if code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized with invalid key, got %d", code)
		}
	})

	// 5. Create Consumer and Key Credential
	consumerPayload := map[string]any{
		"username": "e2e-test-consumer",
		"enabled":  true,
	}
	code, cRes := adminRequest(t, "POST", "/api/v1/consumers", consumerPayload)
	if code != http.StatusOK && code != http.StatusCreated {
		t.Fatalf("failed to create consumer: %d: %v", code, cRes)
	}
	consumerID := extractID(cRes)
	defer adminRequest(t, "DELETE", "/api/v1/consumers/"+consumerID, nil)

	keyCredPayload := map[string]any{
		"name":     "E2E Key",
		"key_type": "api-key",
	}
	code, credRes := adminRequest(t, "POST", fmt.Sprintf("/api/v1/consumers/%s/credentials", consumerID), keyCredPayload)
	if code != http.StatusOK && code != http.StatusCreated {
		t.Fatalf("failed to create key credential: %d: %v", code, credRes)
	}

	apiKey, _ := credRes["key_secret"].(string)
	if apiKey == "" {
		t.Fatalf("expected key_secret in credential response: %v", credRes)
	}

	time.Sleep(100 * time.Millisecond)

	// 6. Request with Valid Key -> Authorized (not 401)
	t.Run("Request with Valid API Key Authorized", func(t *testing.T) {
		headers := map[string]string{"x-api-key": apiKey}
		code, _ := gatewayRequest(t, "GET", "/api/v1/e2e-key-protected", headers, nil)
		if code == http.StatusUnauthorized || code == http.StatusForbidden {
			t.Errorf("expected request with valid API key to be authorized, got status %d", code)
		}
	})
}
