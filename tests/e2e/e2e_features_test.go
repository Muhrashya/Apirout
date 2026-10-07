package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

const (
	gatewayURL  = "http://localhost:8000"
	adminURL    = "http://localhost:8001"
	guiURL      = "http://localhost:8002"
	statusURL   = "http://localhost:8100"
	adminSecret = "MasterSecretKey2026!"
)

// Helper to make authenticated admin requests
func adminRequest(t *testing.T, method, path string, body any) (int, map[string]any) {
	t.Helper()
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("failed to marshal request body: %v", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, adminURL+path, bodyReader)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("X-Apirout-Admin-Secret", adminSecret)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("admin request %s %s failed: %v", method, path, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]any
	if len(respBody) > 0 && respBody[0] == '{' {
		_ = json.Unmarshal(respBody, &result)
	}

	return resp.StatusCode, result
}

// extractID extracts an ID string from response map (checks root "id" then data.id)
func extractID(res map[string]any) string {
	if res == nil {
		return ""
	}
	if id, ok := res["id"]; ok && id != nil && fmt.Sprintf("%v", id) != "<nil>" {
		return fmt.Sprintf("%v", id)
	}
	if data, ok := res["data"].(map[string]any); ok && data["id"] != nil && fmt.Sprintf("%v", data["id"]) != "<nil>" {
		return fmt.Sprintf("%v", data["id"])
	}
	return ""
}

// Helper to make gateway requests
func gatewayRequest(t *testing.T, method, path string, headers map[string]string, body any) (int, []byte) {
	t.Helper()
	statusCode, _, respBody := gatewayRequestWithHeaders(t, method, path, headers, body)
	return statusCode, respBody
}

// Helper to make authenticated admin requests with custom content type and raw bytes
func adminRequestRaw(t *testing.T, method, path, contentType string, body []byte) (int, map[string]any, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, adminURL+path, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("X-Apirout-Admin-Secret", adminSecret)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("admin raw request %s %s failed: %v", method, path, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]any
	if len(respBody) > 0 && respBody[0] == '{' {
		_ = json.Unmarshal(respBody, &result)
	}
	return resp.StatusCode, result, respBody
}

// Helper to make gateway requests with response headers
func gatewayRequestWithHeaders(t *testing.T, method, path string, headers map[string]string, body any) (int, http.Header, []byte) {
	t.Helper()
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("failed to marshal request body: %v", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, gatewayURL+path, bodyReader)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("gateway request %s %s failed: %v", method, path, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	// Automated verification on highest HTTP seam: every gateway response MUST carry X-Request-ID and Server-Timing
	if reqID := resp.Header.Get("X-Request-ID"); reqID == "" {
		t.Errorf("gateway response missing X-Request-ID header for %s %s", method, path)
	}
	if st := resp.Header.Get("Server-Timing"); !strings.Contains(st, "gateway;dur=") {
		t.Errorf("gateway response missing Server-Timing header for %s %s: got %q", method, path, st)
	}

	return resp.StatusCode, resp.Header, respBody
}

func TestE2E_HealthAndTelemetry(t *testing.T) {
	t.Run("Status Health Check", func(t *testing.T) {
		resp, err := http.Get(statusURL + "/health")
		if err != nil {
			t.Fatalf("status healthcheck failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
	})

	t.Run("Gateway Health Check", func(t *testing.T) {
		statusCode, headers, _ := gatewayRequestWithHeaders(t, "GET", "/health", nil, nil)
		if statusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", statusCode)
		}
		if headers.Get("X-Request-ID") == "" {
			t.Errorf("expected X-Request-ID header on /health")
		}
		if !strings.Contains(headers.Get("Server-Timing"), "gateway;dur=") {
			t.Errorf("expected Server-Timing header on /health, got: %s", headers.Get("Server-Timing"))
		}
	})

	t.Run("Gateway Correlation ID Propagation", func(t *testing.T) {
		customID := "custom-correlation-id-" + fmt.Sprintf("%d", time.Now().UnixNano())
		headers := map[string]string{"X-Request-ID": customID}
		statusCode, respHeaders, _ := gatewayRequestWithHeaders(t, "GET", "/health", headers, nil)
		if statusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", statusCode)
		}
		if got := respHeaders.Get("X-Request-ID"); got != customID {
			t.Errorf("expected preserved X-Request-ID %s, got %s", customID, got)
		}
	})

	t.Run("Admin API Health Check", func(t *testing.T) {
		resp, err := http.Get(adminURL + "/health")
		if err != nil {
			t.Fatalf("admin healthcheck failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
	})

	t.Run("Web Console GUI", func(t *testing.T) {
		resp, err := http.Get(guiURL + "/")
		if err != nil {
			t.Fatalf("web console GUI check failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
	})

	t.Run("Prometheus Metrics Endpoint", func(t *testing.T) {
		resp, err := http.Get(statusURL + "/metrics")
		if err != nil {
			t.Fatalf("metrics scrape failed: %v", err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(body), "apirout_http_requests_total") {
			t.Errorf("expected metric apirout_http_requests_total in scrape")
		}
	})
}

func TestE2E_AdminControlPlaneLifecycle(t *testing.T) {
	// 1. Authentication rejection without secret
	t.Run("Unauthenticated Request Rejected", func(t *testing.T) {
		resp, err := http.Get(adminURL + "/api/v1/services")
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusForbidden {
			t.Errorf("expected 401 or 403, got %d", resp.StatusCode)
		}
	})

	// 2. Services CRUD
	var testServiceID string
	t.Run("Service CRUD", func(t *testing.T) {
		createPayload := map[string]any{
			"name":        "e2e-test-service",
			"description": "Service created during E2E verification",
			"protocol":    "http",
			"host":        "httpbin.org",
			"port":        80,
			"path":        "/anything",
			"enabled":     true,
		}
		status, res := adminRequest(t, "POST", "/api/v1/services", createPayload)
		if status != http.StatusCreated && status != http.StatusOK {
			t.Fatalf("expected 201/200, got %d: %v", status, res)
		}
		if data, ok := res["data"].(map[string]any); ok {
			testServiceID = fmt.Sprintf("%v", data["id"])
		} else if id, ok := res["id"]; ok {
			testServiceID = fmt.Sprintf("%v", id)
		}
		if testServiceID == "" {
			t.Fatalf("could not extract created service ID: %v", res)
		}

		// Read
		status, res = adminRequest(t, "GET", "/api/v1/services/"+testServiceID, nil)
		if status != http.StatusOK {
			t.Errorf("expected 200 reading service, got %d", status)
		}

		// Update
		updatePayload := map[string]any{
			"name":        "e2e-test-service-updated",
			"description": "Updated E2E description",
			"enabled":     true,
		}
		status, _ = adminRequest(t, "PUT", "/api/v1/services/"+testServiceID, updatePayload)
		if status != http.StatusOK {
			t.Errorf("expected 200 updating service, got %d", status)
		}
	})

	// 3. Endpoint / Route CRUD
	var testEndpointID string
	t.Run("Endpoint CRUD", func(t *testing.T) {
		if testServiceID == "" {
			t.Skip("skipping endpoint tests because service creation failed")
		}

		endpointPayload := map[string]any{
			"service_id": testServiceID,
			"method":     "GET",
			"path":       "/api/e2e-test-route",
			"strip_path": true,
			"enabled":    true,
		}
		status, res := adminRequest(t, "POST", "/api/v1/endpoints", endpointPayload)
		if status != http.StatusCreated && status != http.StatusOK {
			t.Fatalf("expected 201/200 creating endpoint, got %d: %v", status, res)
		}
		if data, ok := res["data"].(map[string]any); ok {
			testEndpointID = fmt.Sprintf("%v", data["id"])
		} else if id, ok := res["id"]; ok {
			testEndpointID = fmt.Sprintf("%v", id)
		}

		// List endpoints
		status, listRes := adminRequest(t, "GET", "/api/v1/endpoints", nil)
		if status != http.StatusOK {
			t.Errorf("expected 200 listing endpoints, got %d", status)
		}
		dataArr, _ := listRes["data"].([]any)
		if len(dataArr) == 0 {
			t.Errorf("expected at least 1 endpoint in list")
		}
	})

	// 4. Consumers & API Key Auth Plugin
	var testConsumerID string
	t.Run("Consumer & Key Auth", func(t *testing.T) {
		consumerPayload := map[string]any{
			"username": "e2e-test-user",
			"status":   "active",
		}
		status, res := adminRequest(t, "POST", "/api/v1/consumers", consumerPayload)
		if status != http.StatusCreated && status != http.StatusOK {
			t.Fatalf("expected 201/200 creating consumer, got %d: %v", status, res)
		}
		if data, ok := res["data"].(map[string]any); ok {
			testConsumerID = fmt.Sprintf("%v", data["id"])
		} else if id, ok := res["id"]; ok {
			testConsumerID = fmt.Sprintf("%v", id)
		}

		// Generate API Key Credential
		if testConsumerID != "" {
			keyPayload := map[string]any{
				"name":     "E2E API Key",
				"key_type": "api-key",
				"secret":   "e2e-secret-key-123456",
			}
			status, credRes := adminRequest(t, "POST", fmt.Sprintf("/api/v1/consumers/%s/credentials", testConsumerID), keyPayload)
			if status != http.StatusCreated && status != http.StatusOK {
				t.Errorf("credential creation failed: %d %v", status, credRes)
			}
		}
	})

	// 5. Upstreams CRUD
	var testUpstreamID string
	t.Run("Upstream CRUD", func(t *testing.T) {
		upstreamPayload := map[string]any{
			"name":      "e2e-test-upstream",
			"algorithm": "round-robin",
			"slots":     1000,
			"enabled":   true,
		}
		status, res := adminRequest(t, "POST", "/api/v1/upstreams", upstreamPayload)
		if status != http.StatusCreated && status != http.StatusOK {
			t.Fatalf("expected 201/200 creating upstream, got %d: %v", status, res)
		}
		if data, ok := res["data"].(map[string]any); ok {
			testUpstreamID = fmt.Sprintf("%v", data["id"])
		} else if id, ok := res["id"]; ok {
			testUpstreamID = fmt.Sprintf("%v", id)
		}

		// List upstreams
		status, _ = adminRequest(t, "GET", "/api/v1/upstreams", nil)
		if status != http.StatusOK {
			t.Errorf("expected 200 listing upstreams, got %d", status)
		}
	})

	// 6. GitOps Metadata, Dashboard Stats, OpenAPI & Audit Logs
	t.Run("GitOps Export, Stats, OpenAPI & Audit Logs", func(t *testing.T) {
		// GitOps Metadata Export
		req, _ := http.NewRequest("GET", adminURL+"/api/v1/metadata/export", nil)
		req.Header.Set("X-Apirout-Admin-Secret", adminSecret)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("metadata export failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 metadata export, got %d", resp.StatusCode)
		}

		// Dashboard Stats
		status, statsRes := adminRequest(t, "GET", "/api/v1/dashboard/stats", nil)
		if status != http.StatusOK {
			t.Errorf("expected 200 dashboard stats, got %d", status)
		}
		if statsRes == nil {
			t.Errorf("expected non-nil dashboard stats response")
		}

		// OpenAPI Docs
		status, _ = adminRequest(t, "GET", "/api/v1/openapi.json", nil)
		if status != http.StatusOK {
			t.Errorf("expected 200 openapi, got %d", status)
		}

		// Audit Logs List
		status, auditRes := adminRequest(t, "GET", "/api/v1/audit-logs", nil)
		if status != http.StatusOK {
			t.Errorf("expected 200 audit logs, got %d", status)
		}
		if auditRes == nil {
			t.Errorf("expected non-nil audit logs")
		}
	})

	// Cleanup
	t.Cleanup(func() {
		if testEndpointID != "" {
			_, _ = adminRequest(t, "DELETE", "/api/v1/endpoints/"+testEndpointID, nil)
		}
		if testServiceID != "" {
			_, _ = adminRequest(t, "DELETE", "/api/v1/services/"+testServiceID, nil)
		}
		if testConsumerID != "" {
			_, _ = adminRequest(t, "DELETE", "/api/v1/consumers/"+testConsumerID, nil)
		}
		if testUpstreamID != "" {
			_, _ = adminRequest(t, "DELETE", "/api/v1/upstreams/"+testUpstreamID, nil)
		}
	})
}

func TestE2E_InstantCRUDAndSQLDialect(t *testing.T) {
	// First fetch existing target connection ID
	status, connRes := adminRequest(t, "GET", "/api/v1/connections", nil)
	if status != http.StatusOK {
		t.Fatalf("failed to query connections: %d", status)
	}
	conns, _ := connRes["data"].([]any)
	if len(conns) == 0 {
		t.Fatalf("no target connections found in registry")
	}
	firstConn := conns[0].(map[string]any)
	connID := fmt.Sprintf("%v", firstConn["id"])

	authHeaders := map[string]string{
		"X-Admin-Secret": adminSecret,
	}

	t.Run("Schema Introspection", func(t *testing.T) {
		code, body := gatewayRequest(t, "GET", fmt.Sprintf("/api/v1/instant/%s/schema", connID), authHeaders, nil)
		if code != http.StatusOK {
			t.Fatalf("expected 200 schema, got %d: %s", code, string(body))
		}
		var schema struct {
			Tables []struct {
				Name string `json:"name"`
			} `json:"tables"`
		}
		if err := json.Unmarshal(body, &schema); err != nil {
			t.Fatalf("failed to parse schema: %v", err)
		}
		if len(schema.Tables) == 0 {
			t.Errorf("expected tables in schema, got 0")
		}
	})

	t.Run("Instant CRUD Table Query & Pagination", func(t *testing.T) {
		// Query services table with limit and offset
		path := fmt.Sprintf("/api/v1/instant/%s/services?limit=2&offset=0&order=name.ASC", connID)
		code, body := gatewayRequest(t, "GET", path, authHeaders, nil)
		if code != http.StatusOK {
			t.Fatalf("expected 200 query table, got %d: %s", code, string(body))
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
			t.Fatalf("failed to parse query response: %v", err)
		}

		if respData.Meta.Limit != 2 {
			t.Errorf("expected limit 2, got %d", respData.Meta.Limit)
		}
	})

	t.Run("Dynamic Filter Operators", func(t *testing.T) {
		// Test eq, neq, like, is.null
		testCases := []string{
			"enabled=eq.true",
			"protocol=neq.grpc",
			"name=like.*Postgres*",
			"upstream_id=is.null",
		}

		for _, tc := range testCases {
			path := fmt.Sprintf("/api/v1/instant/%s/services?%s", connID, tc)
			code, body := gatewayRequest(t, "GET", path, authHeaders, nil)
			if code != http.StatusOK {
				t.Errorf("filter %q failed with status %d: %s", tc, code, string(body))
			}
		}
	})

	t.Run("Row Mutation Lifecycle (Insert, Get, Update, Delete)", func(t *testing.T) {
		testSettingKey := fmt.Sprintf("e2e_setting_%d", time.Now().Unix())

		// 1. Insert into system_settings
		insertBody := map[string]any{
			"key":   testSettingKey,
			"value": "e2e_initial_value",
		}
		code, body := gatewayRequest(t, "POST", fmt.Sprintf("/api/v1/instant/%s/system_settings", connID), authHeaders, insertBody)
		if code != http.StatusCreated && code != http.StatusOK {
			t.Fatalf("insert failed with %d: %s", code, string(body))
		}

		// 2. Get single row
		code, body = gatewayRequest(t, "GET", fmt.Sprintf("/api/v1/instant/%s/system_settings/%s", connID, testSettingKey), authHeaders, nil)
		if code != http.StatusOK {
			t.Errorf("get single row failed with %d: %s", code, string(body))
		}

		// 3. Update row
		updateBody := map[string]any{
			"value": "e2e_updated_value",
		}
		code, body = gatewayRequest(t, "PUT", fmt.Sprintf("/api/v1/instant/%s/system_settings/%s", connID, testSettingKey), authHeaders, updateBody)
		if code != http.StatusOK {
			t.Errorf("update row failed with %d: %s", code, string(body))
		}

		// 4. Delete row
		code, body = gatewayRequest(t, "DELETE", fmt.Sprintf("/api/v1/instant/%s/system_settings/%s", connID, testSettingKey), authHeaders, nil)
		if code != http.StatusOK {
			t.Errorf("delete row failed with %d: %s", code, string(body))
		}

		// 5. Verify deleted row returns 404
		code, body = gatewayRequest(t, "GET", fmt.Sprintf("/api/v1/instant/%s/system_settings/%s", connID, testSettingKey), authHeaders, nil)
		if code != http.StatusNotFound {
			t.Errorf("expected 404 after delete, got %d", code)
		}
		var errResp map[string]any
		if err := json.Unmarshal(body, &errResp); err == nil {
			if errResp["code"] != "RECORD_NOT_FOUND" {
				t.Errorf("expected code RECORD_NOT_FOUND, got %v", errResp["code"])
			}
			if errResp["status"] != false {
				t.Errorf("expected status false in hybrid envelope, got %v", errResp["status"])
			}
		} else {
			t.Errorf("failed to unmarshal 404 response: %v", err)
		}
	})

	t.Run("Query Preview with Dialect Pagination", func(t *testing.T) {
		previewPayload := map[string]any{
			"connection_id": connID,
			"sql":           "SELECT id, name FROM services ORDER BY name ASC",
			"limit":         5,
			"offset":        0,
		}
		status, res := adminRequest(t, "POST", "/api/v1/queries/preview", previewPayload)
		if status != http.StatusOK {
			t.Fatalf("query preview failed with status %d: %v", status, res)
		}
		if res["data"] == nil {
			t.Errorf("expected preview data in response")
		}
	})

	t.Run("First-Time Setup Discovery and Lockout", func(t *testing.T) {
		// 1. Unauthenticated check on setup-status
		resp, err := http.Get(adminURL + "/api/v1/auth/setup-status")
		if err != nil {
			t.Fatalf("failed to query setup-status: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK from setup-status, got %d", resp.StatusCode)
		}

		var setupStatus map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&setupStatus); err != nil {
			t.Fatalf("failed to decode setup-status json: %v", err)
		}

		// The existing cluster has a super admin seeded on boot
		if setupStatus["has_super_admin"] != true || setupStatus["setup_required"] != false {
			t.Fatalf("expected has_super_admin=true and setup_required=false, got %+v", setupStatus)
		}

		// 2. Submitting setup against already-configured instance must return 409 Conflict
		setupPayload := map[string]string{
			"email":            "intruder@apirout.io",
			"password":         "IntruderPassword2026!",
			"confirm_password": "IntruderPassword2026!",
		}
		payloadBytes, _ := json.Marshal(setupPayload)
		postResp, err := http.Post(adminURL+"/api/v1/auth/setup", "application/json", bytes.NewReader(payloadBytes))
		if err != nil {
			t.Fatalf("failed to post setup: %v", err)
		}
		defer postResp.Body.Close()

		if postResp.StatusCode != http.StatusConflict {
			t.Fatalf("expected 409 Conflict when super admin already exists, got %d", postResp.StatusCode)
		}
	})
}

func TestE2E_DeclarativeStateEngineGitOps(t *testing.T) {
	// Step 1: Fault Injection - Atomic Rollback on Missing Dependency
	t.Run("Atomic Rollback on Missing Dependency", func(t *testing.T) {
		invalidYAML := `
version: "api.apirout.io/v1"
services:
  - name: "orphan-service-e2e"
    upstream_name: "phantom-upstream-nonexistent"
`
		status, resMap, rawBody := adminRequestRaw(t, "POST", "/api/v1/metadata/import", "application/x-yaml", []byte(invalidYAML))
		if status != http.StatusUnprocessableEntity {
			t.Fatalf("expected 422 Unprocessable Entity, got %d: %s", status, string(rawBody))
		}
		if resMap["entity"] != "Service" || resMap["identifier"] != "orphan-service-e2e" {
			t.Errorf("expected structured error on Service orphan-service-e2e, got: %+v", resMap)
		}

		// Verify zero orphaned records: Service must NOT exist in datastore
		code, getRes := adminRequest(t, "GET", "/api/v1/services", nil)
		if code == http.StatusOK {
			data, _ := getRes["data"].([]any)
			for _, item := range data {
				svc := item.(map[string]any)
				if svc["name"] == "orphan-service-e2e" {
					t.Fatalf("orphan-service-e2e was persisted despite validation failure (atomic rollback failed)!")
				}
			}
		}
	})

	// Step 2: Valid Multi-Tier Declarative Ingestion
	t.Run("Valid Multi-Tier Declarative Ingestion", func(t *testing.T) {
		validYAML := `
version: "api.apirout.io/v1"
groups:
  - name: "e2e-gitops-grp"
    description: "Group created via declarative gitops"
upstreams:
  - name: "e2e-gitops-upstream"
    algorithm: "round-robin"
    targets:
      - target: "127.0.0.1:8100"
        host: "127.0.0.1"
        port: 8100
        weight: 100
        enabled: true
services:
  - name: "e2e-gitops-svc"
    protocol: "http"
    path: "/health"
    upstream_name: "e2e-gitops-upstream"
    enabled: true
consumers:
  - username: "e2e-gitops-consumer"
    group_name: "e2e-gitops-grp"
    status: "active"
endpoints:
  - path: "/api/e2e-gitops-endpoint"
    name: "E2E GitOps Endpoint"
    method: "GET"
    service_name: "e2e-gitops-svc"
    group_name: "e2e-gitops-grp"
    enabled: true
    is_public: true
`
		status, resMap, rawBody := adminRequestRaw(t, "POST", "/api/v1/metadata/import", "application/x-yaml", []byte(validYAML))
		if status != http.StatusOK {
			t.Fatalf("expected 200 OK for valid import, got %d: %s", status, string(rawBody))
		}
		if resMap["status"] != true {
			t.Errorf("expected status true, got: %+v", resMap)
		}

		// Step 3: Verify All Entities Queryable in Control Plane
		status, svcList := adminRequest(t, "GET", "/api/v1/services", nil)
		if status != http.StatusOK {
			t.Errorf("failed to list services: %d", status)
		}
		foundSvc := false
		if data, ok := svcList["data"].([]any); ok {
			for _, item := range data {
				if s := item.(map[string]any); s["name"] == "e2e-gitops-svc" {
					foundSvc = true
					break
				}
			}
		}
		if !foundSvc {
			t.Errorf("e2e-gitops-svc not found in services list")
		}

		status, epList := adminRequest(t, "GET", "/api/v1/endpoints", nil)
		if status != http.StatusOK {
			t.Errorf("failed to list endpoints: %d", status)
		}
		foundEP := false
		var epID string
		if data, ok := epList["data"].([]any); ok {
			for _, item := range data {
				if ep := item.(map[string]any); ep["path"] == "/api/e2e-gitops-endpoint" {
					foundEP = true
					epID = fmt.Sprintf("%v", ep["id"])
					break
				}
			}
		}
		if !foundEP {
			t.Errorf("/api/e2e-gitops-endpoint not found in endpoints list")
		}

		// Step 4: Idempotent Re-Import
		status2, resMap2, rawBody2 := adminRequestRaw(t, "POST", "/api/v1/metadata/import", "application/x-yaml", []byte(validYAML))
		if status2 != http.StatusOK {
			t.Fatalf("expected 200 OK on re-import, got %d: %s", status2, string(rawBody2))
		}
		if resMap2["status"] != true {
			t.Errorf("expected status true on re-import, got: %+v", resMap2)
		}

		// Cleanup
		t.Cleanup(func() {
			if epID != "" {
				_, _ = adminRequest(t, "DELETE", "/api/v1/endpoints/"+epID, nil)
			}
		})
	})
}

func TestE2E_RadixRouterAndMethodNotAllowed(t *testing.T) {
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	svcName := "e2e-radix-svc-" + suffix
	routePath := "/api/e2e-radix-" + suffix

	// Create service pointing to status health endpoint (127.0.0.1:8100)
	svcPayload := map[string]any{
		"name":     svcName,
		"protocol": "http",
		"host":     "127.0.0.1",
		"port":     8100,
		"path":     "/health",
		"enabled":  true,
	}
	code, svcRes := adminRequest(t, "POST", "/api/v1/services", svcPayload)
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to create service: %d %v", code, svcRes)
	}
	svcID := extractID(svcRes)
	t.Cleanup(func() {
		if svcID != "" {
			_, _ = adminRequest(t, "DELETE", "/api/v1/services/"+svcID, nil)
		}
	})

	// Create endpoint GET routePath
	epPayload := map[string]any{
		"service_id": svcID,
		"method":     "GET",
		"path":       routePath,
		"strip_path": true,
		"enabled":    true,
		"is_public":  true,
	}
	code, epRes := adminRequest(t, "POST", "/api/v1/endpoints", epPayload)
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to create endpoint: %d %v", code, epRes)
	}
	epID := extractID(epRes)
	t.Cleanup(func() {
		if epID != "" {
			_, _ = adminRequest(t, "DELETE", "/api/v1/endpoints/"+epID, nil)
		}
	})

	// Allow EventBus to synchronize in-memory router
	time.Sleep(200 * time.Millisecond)

	// Subtest 1: Exact Match GET succeeds through radix router
	t.Run("GET Route Resolution", func(t *testing.T) {
		statusCode, _, body := gatewayRequestWithHeaders(t, "GET", routePath, nil, nil)
		if statusCode != http.StatusOK {
			t.Fatalf("expected 200 OK from gateway, got %d: %s", statusCode, string(body))
		}
	})

	// Subtest 2: Method Not Allowed (405) with Allow header
	t.Run("HTTP 405 Method Not Allowed", func(t *testing.T) {
		for _, method := range []string{"POST", "DELETE", "PUT"} {
			statusCode, headers, body := gatewayRequestWithHeaders(t, method, routePath, nil, nil)
			if statusCode != http.StatusMethodNotAllowed {
				t.Errorf("expected 405 for %s %s, got %d", method, routePath, statusCode)
			}
			allowHeader := headers.Get("Allow")
			if !strings.Contains(allowHeader, "GET") {
				t.Errorf("expected Allow header containing GET, got: %s", allowHeader)
			}
			var errResp map[string]any
			if err := json.Unmarshal(body, &errResp); err == nil {
				if errResp["code"] != "METHOD_NOT_ALLOWED" {
					t.Errorf("expected code METHOD_NOT_ALLOWED, got %v", errResp["code"])
				}
				if errResp["request_id"] == nil || errResp["request_id"] == "" {
					t.Errorf("expected request_id in error body, got %v", errResp["request_id"])
				}
				if errResp["status"] != false {
					t.Errorf("expected status false in hybrid envelope, got %v", errResp["status"])
				}
			} else {
				t.Errorf("failed to unmarshal 405 response: %v", err)
			}
		}
	})

	// Subtest 3: HTTP 404 for Non-Existent Route
	t.Run("HTTP 404 Route Not Found", func(t *testing.T) {
		statusCode, _, body := gatewayRequestWithHeaders(t, "GET", "/api/ghost-route-not-here-99", nil, nil)
		if statusCode != http.StatusNotFound {
			t.Errorf("expected 404 for non-existent route, got %d", statusCode)
		}
		var errResp map[string]any
		if err := json.Unmarshal(body, &errResp); err == nil {
			if errResp["code"] != "ROUTE_NOT_FOUND" {
				t.Errorf("expected code ROUTE_NOT_FOUND, got %v", errResp["code"])
			}
			if errResp["request_id"] == nil || errResp["request_id"] == "" {
				t.Errorf("expected request_id in error body, got %v", errResp["request_id"])
			}
			if errResp["status"] != false {
				t.Errorf("expected status false in hybrid envelope, got %v", errResp["status"])
			}
		} else {
			t.Errorf("failed to unmarshal 404 response: %v", err)
		}
	})
}

func TestE2E_PassiveCircuitBreakerAndFailover(t *testing.T) {
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	upName := "e2e-cb-upstream-" + suffix
	svcName := "e2e-cb-svc-" + suffix
	routePath := "/api/e2e-cb-" + suffix

	// Create upstream with unreachable target (dead port 59199)
	upPayload := map[string]any{
		"name":      upName,
		"algorithm": "round-robin",
		"slots":     1000,
		"enabled":   true,
	}
	code, upRes := adminRequest(t, "POST", "/api/v1/upstreams", upPayload)
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to create upstream: %d %v", code, upRes)
	}
	upID := extractID(upRes)
	t.Cleanup(func() {
		if upID != "" {
			_, _ = adminRequest(t, "DELETE", "/api/v1/upstreams/"+upID, nil)
		}
	})

	// Add target pointing to dead port
	targetPayload := map[string]any{
		"target":  "127.0.0.1:59199",
		"host":    "127.0.0.1",
		"port":    59199,
		"weight":  100,
		"enabled": true,
	}
	code, _ = adminRequest(t, "POST", fmt.Sprintf("/api/v1/upstreams/%s/targets", upID), targetPayload)
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to create target: %d", code)
	}

	// Create service pointing to this upstream
	svcPayload := map[string]any{
		"name":        svcName,
		"protocol":    "http",
		"path":        "/test",
		"upstream_id": upID,
		"enabled":     true,
	}
	code, svcRes := adminRequest(t, "POST", "/api/v1/services", svcPayload)
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to create service: %d %v", code, svcRes)
	}
	svcID := extractID(svcRes)
	t.Cleanup(func() {
		if svcID != "" {
			_, _ = adminRequest(t, "DELETE", "/api/v1/services/"+svcID, nil)
		}
	})

	// Create endpoint
	epPayload := map[string]any{
		"service_id": svcID,
		"method":     "GET",
		"path":       routePath,
		"enabled":    true,
		"is_public":  true,
	}
	code, epRes := adminRequest(t, "POST", "/api/v1/endpoints", epPayload)
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

	// Send requests to trip the circuit breaker (threshold = 3 consecutive failures)
	for i := 0; i < 3; i++ {
		_, _, _ = gatewayRequestWithHeaders(t, "GET", routePath, nil, nil)
	}

	// Once tripped, subsequent requests immediately fast-fail with HTTP 503 Service Unavailable
	t.Run("Fast-Fail HTTP 503 When Tripped", func(t *testing.T) {
		statusCode, _, body := gatewayRequestWithHeaders(t, "GET", routePath, nil, nil)
		if statusCode != http.StatusServiceUnavailable {
			t.Fatalf("expected 503 Service Unavailable after circuit tripped, got %d: %s", statusCode, string(body))
		}
		if !strings.Contains(string(body), "no healthy upstream targets available") {
			t.Errorf("expected 'no healthy upstream targets available' in error body, got: %s", string(body))
		}
		var errResp map[string]any
		if err := json.Unmarshal(body, &errResp); err == nil {
			if errResp["code"] != "NO_HEALTHY_TARGETS" {
				t.Errorf("expected code NO_HEALTHY_TARGETS, got %v", errResp["code"])
			}
			if errResp["status"] != false {
				t.Errorf("expected status false in hybrid envelope, got %v", errResp["status"])
			}
		} else {
			t.Errorf("failed to unmarshal 503 response: %v", err)
		}
	})
}

func TestE2E_ConsumerAuthAndRateLimiting(t *testing.T) {
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	username := "e2e-user-" + suffix
	svcName := "e2e-auth-svc-" + suffix
	routePath := "/api/e2e-auth-" + suffix

	// 1. Create Consumer
	consumerPayload := map[string]any{
		"username": username,
		"status":   "active",
	}
	code, consRes := adminRequest(t, "POST", "/api/v1/consumers", consumerPayload)
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
	keyPayload := map[string]any{
		"name":     "E2E Rate Limit Key",
		"key_type": "api-key",
	}
	code, credRes := adminRequest(t, "POST", fmt.Sprintf("/api/v1/consumers/%s/credentials", consumerID), keyPayload)
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to create credential: %d %v", code, credRes)
	}
	apiKey := fmt.Sprintf("%v", credRes["key_secret"])
	if apiKey == "" || apiKey == "<nil>" {
		if d, ok := credRes["data"].(map[string]any); ok {
			apiKey = fmt.Sprintf("%v", d["key_secret"])
		}
	}

	// 3. Create service pointing to status health
	svcPayload := map[string]any{
		"name":     svcName,
		"protocol": "http",
		"host":     "127.0.0.1",
		"port":     8100,
		"path":     "/health",
		"enabled":  true,
	}
	code, svcRes := adminRequest(t, "POST", "/api/v1/services", svcPayload)
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to create service: %d %v", code, svcRes)
	}
	svcID := extractID(svcRes)
	t.Cleanup(func() {
		if svcID != "" {
			_, _ = adminRequest(t, "DELETE", "/api/v1/services/"+svcID, nil)
		}
	})

	// 4. Create protected endpoint with tight rate limit: RPM 10, Burst 2
	epPayload := map[string]any{
		"service_id":         svcID,
		"method":             "GET",
		"path":               routePath,
		"strip_path":         true,
		"is_public":          false, // Protected
		"rate_limit_enabled": true,
		"rate_limit_rpm":     10,
		"rate_limit_burst":   2,
		"enabled":            true,
	}
	code, epRes := adminRequest(t, "POST", "/api/v1/endpoints", epPayload)
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to create endpoint: %d %v", code, epRes)
	}
	epID := extractID(epRes)
	t.Cleanup(func() {
		if epID != "" {
			_, _ = adminRequest(t, "DELETE", "/api/v1/endpoints/"+epID, nil)
		}
	})

	// 5. Associate consumer with endpoint
	routesPayload := map[string]any{
		"route_ids": []string{epID},
	}
	code, _ = adminRequest(t, "PUT", fmt.Sprintf("/api/v1/consumers/%s/routes", consumerID), routesPayload)
	if code != http.StatusOK {
		t.Fatalf("failed to associate route with consumer: %d", code)
	}

	time.Sleep(200 * time.Millisecond)

	// Missing Authentication -> 401 Unauthorized
	t.Run("Missing Authentication Rejected", func(t *testing.T) {
		statusCode, _, body := gatewayRequestWithHeaders(t, "GET", routePath, nil, nil)
		if statusCode != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized without auth, got %d", statusCode)
		}
		var errResp map[string]any
		if err := json.Unmarshal(body, &errResp); err == nil {
			if errResp["code"] != "UNAUTHORIZED" {
				t.Errorf("expected code UNAUTHORIZED, got %v", errResp["code"])
			}
			if errResp["status"] != false {
				t.Errorf("expected status false in hybrid envelope, got %v", errResp["status"])
			}
		} else {
			t.Errorf("failed to unmarshal 401 response: %v", err)
		}
	})

	// Invalid API Key -> 401 Unauthorized
	t.Run("Invalid API Key Rejected", func(t *testing.T) {
		badHeaders := map[string]string{"X-API-Key": "ak_invalid_key_99999"}
		statusCode, _, body := gatewayRequestWithHeaders(t, "GET", routePath, badHeaders, nil)
		if statusCode != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized with invalid key, got %d", statusCode)
		}
		var errResp map[string]any
		if err := json.Unmarshal(body, &errResp); err == nil {
			if errResp["code"] != "INVALID_API_KEY" {
				t.Errorf("expected code INVALID_API_KEY, got %v", errResp["code"])
			}
			if errResp["status"] != false {
				t.Errorf("expected status false in hybrid envelope, got %v", errResp["status"])
			}
		} else {
			t.Errorf("failed to unmarshal 401 response: %v", err)
		}
	})

	// Valid API Key -> 200 OK
	t.Run("Valid API Key Accepted", func(t *testing.T) {
		authHeaders := map[string]string{"X-API-Key": apiKey}
		statusCode, _, body := gatewayRequestWithHeaders(t, "GET", routePath, authHeaders, nil)
		if statusCode != http.StatusOK {
			t.Errorf("expected 200 OK with valid API key, got %d: %s", statusCode, string(body))
		}
	})

	// Rate Limiting Burst Exceeded -> 429 Too Many Requests
	t.Run("Rate Limiting Burst Exceeded", func(t *testing.T) {
		authHeaders := map[string]string{"X-API-Key": apiKey}
		got429 := false
		for i := 0; i < 6; i++ {
			statusCode, headers, body := gatewayRequestWithHeaders(t, "GET", routePath, authHeaders, nil)
			if statusCode == http.StatusTooManyRequests {
				got429 = true
				retryAfter := headers.Get("Retry-After")
				if retryAfter == "" {
					t.Errorf("expected Retry-After header on 429 response")
				}
				var errResp map[string]any
				if err := json.Unmarshal(body, &errResp); err == nil {
					if errResp["code"] != "RATE_LIMIT_EXCEEDED" {
						t.Errorf("expected code RATE_LIMIT_EXCEEDED, got %v", errResp["code"])
					}
					if errResp["status"] != false {
						t.Errorf("expected status false in hybrid envelope, got %v", errResp["status"])
					}
				} else {
					t.Errorf("failed to unmarshal 429 response: %v", err)
				}
				break
			}
		}
		if !got429 {
			t.Errorf("expected at least one 429 Too Many Requests after burst exceeded")
		}
	})
}

func TestE2E_AuditLoggingTelemetry(t *testing.T) {
	// Send request through gateway
	_, _, _ = gatewayRequestWithHeaders(t, "GET", "/health", nil, nil)

	// Fetch audit logs from control plane
	time.Sleep(100 * time.Millisecond)
	status, res := adminRequest(t, "GET", "/api/v1/audit-logs", nil)
	if status != http.StatusOK {
		t.Fatalf("failed to fetch audit logs: %d", status)
	}
	logs, _ := res["data"].([]any)
	if len(logs) == 0 {
		t.Errorf("expected audit logs to contain records")
	}
}

func TestE2E_UnrestrictedConsumerDefaultAccessToProtectedRoutes(t *testing.T) {
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	username := "default-consumer-" + suffix
	svcName := "default-auth-svc-" + suffix
	routePath := "/api/default-auth-" + suffix

	// 1. Create Consumer with NO group and NO route restrictions
	consumerPayload := map[string]any{
		"username": username,
		"status":   "active",
	}
	code, consRes := adminRequest(t, "POST", "/api/v1/consumers", consumerPayload)
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
	keyPayload := map[string]any{
		"name":     "Default Auth Key",
		"key_type": "api-key",
	}
	code, credRes := adminRequest(t, "POST", fmt.Sprintf("/api/v1/consumers/%s/credentials", consumerID), keyPayload)
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
	svcPayload := map[string]any{
		"name":     svcName,
		"protocol": "http",
		"host":     "127.0.0.1",
		"port":     8100,
		"path":     "/health",
		"enabled":  true,
	}
	code, svcRes := adminRequest(t, "POST", "/api/v1/services", svcPayload)
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("failed to create service: %d %v", code, svcRes)
	}
	svcID := extractID(svcRes)
	t.Cleanup(func() {
		if svcID != "" {
			_, _ = adminRequest(t, "DELETE", "/api/v1/services/"+svcID, nil)
		}
	})

	// 4. Create Protected Route (is_public: false, NO group assigned)
	epPayload := map[string]any{
		"service_id": svcID,
		"method":     "GET",
		"path":       routePath,
		"strip_path": true,
		"is_public":  false, // Protected!
		"enabled":    true,
	}
	code, epRes := adminRequest(t, "POST", "/api/v1/endpoints", epPayload)
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

	// 5. Without API key -> 401 Unauthorized
	code, _, _ = gatewayRequestWithHeaders(t, "GET", routePath, nil, nil)
	if code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized without API key, got %d", code)
	}

	// 6. With API key -> MUST be 200 OK (NOT 403 Forbidden!)
	// This proves that unrestricted consumers can access protected routes without manual route linking
	headers := map[string]string{"X-API-Key": apiKey}
	code, respHeaders, body := gatewayRequestWithHeaders(t, "GET", routePath, headers, nil)
	if code != http.StatusOK {
		t.Fatalf("expected 200 OK with valid API key for unrestricted consumer, got %d: %s", code, string(body))
	}
	if respHeaders.Get("X-Request-ID") == "" {
		t.Errorf("expected non-empty X-Request-ID on authenticated response")
	}
}

