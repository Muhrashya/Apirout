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

// Helper to make gateway requests
func gatewayRequest(t *testing.T, method, path string, headers map[string]string, body any) (int, []byte) {
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
	return resp.StatusCode, respBody
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
		resp, err := http.Get(gatewayURL + "/health")
		if err != nil {
			t.Fatalf("gateway healthcheck failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
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
		code, _ = gatewayRequest(t, "GET", fmt.Sprintf("/api/v1/instant/%s/system_settings/%s", connID, testSettingKey), authHeaders, nil)
		if code != http.StatusNotFound {
			t.Errorf("expected 404 after delete, got %d", code)
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
}
