package registry

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/apirout/apirout/internal/core/models"
)

func TestMetricsSummaryFromAuditLogs(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_metrics.db")

	reg, err := NewWithSQLite(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test registry: %v", err)
	}
	defer reg.db.Close()

	if err := reg.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}
	if err := reg.MigrateSchema(); err != nil {
		t.Fatalf("Failed to migrate schema: %v", err)
	}

	since := time.Now().Add(-1 * time.Hour)

	// Create test endpoint
	ep := &models.Endpoint{
		ID:     "ep-123",
		Name:   "Get Users",
		Path:   "/api/v1/users",
		Method: "GET",
	}
	if err := reg.CreateEndpoint(ep); err != nil {
		t.Fatalf("Failed to create test endpoint: %v", err)
	}

	// Insert test requests into audit_logs:
	// - 5x 200 OK: latencies [10, 20, 30, 40, 50]
	// - 3x 404 Not Found: latencies [4, 5, 6]
	// - 2x 500 Server Error: latencies [100, 200]
	testCases := []struct {
		statusCode int
		durationMs int
		success    bool
	}{
		{200, 10, true},
		{200, 20, true},
		{200, 30, true},
		{200, 40, true},
		{200, 50, true},
		{404, 4, false},
		{404, 5, false},
		{404, 6, false},
		{500, 100, false},
		{500, 200, false},
	}

	for i, tc := range testCases {
		dur := tc.durationMs
		epID := ep.ID
		detailsJSON, _ := json.Marshal(map[string]interface{}{
			"status_code": tc.statusCode,
			"path":        ep.Path,
			"method":      ep.Method,
		})

		log := &AuditLog{
			ID:          fmt.Sprintf("log-%d", i+1),
			EventType:   "api_request",
			EndpointID:  &epID,
			DurationMs:  &dur,
			Success:     tc.success,
			Details:     string(detailsJSON),
			Timestamp:   time.Now(),
		}
		if err := reg.CreateAuditLog(log); err != nil {
			t.Fatalf("Failed to insert audit log: %v", err)
		}
	}

	// Call GetMetricsSummary
	summary, err := reg.GetMetricsSummary(since)
	if err != nil {
		t.Fatalf("GetMetricsSummary returned error: %v", err)
	}

	// Total requests must be 10
	totalRequests, ok := summary["total_requests"].(int)
	if !ok || totalRequests != 10 {
		t.Errorf("Expected total_requests == 10, got %v", summary["total_requests"])
	}

	// Status breakdown
	if s2xx, ok := summary["status_2xx"].(int); !ok || s2xx != 5 {
		t.Errorf("Expected status_2xx == 5, got %v", summary["status_2xx"])
	}
	if s4xx, ok := summary["status_4xx"].(int); !ok || s4xx != 3 {
		t.Errorf("Expected status_4xx == 3, got %v", summary["status_4xx"])
	}
	if s5xx, ok := summary["status_5xx"].(int); !ok || s5xx != 2 {
		t.Errorf("Expected status_5xx == 2, got %v", summary["status_5xx"])
	}

	// Latency percentiles across sorted durations [4, 5, 6, 10, 20, 30, 40, 50, 100, 200]:
	// Median / P50 should be between 20-30ms
	p50, ok := summary["latency_p50_ms"].(int)
	if !ok || p50 <= 0 {
		t.Errorf("Expected positive latency_p50_ms, got %v", summary["latency_p50_ms"])
	}

	// P90 should be around 100ms
	p90, ok := summary["latency_p90_ms"].(int)
	if !ok || p90 < 50 {
		t.Errorf("Expected latency_p90_ms >= 50ms, got %v", summary["latency_p90_ms"])
	}

	// P99 should be 200ms
	p99, ok := summary["latency_p99_ms"].(int)
	if !ok || p99 < 100 {
		t.Errorf("Expected latency_p99_ms >= 100ms, got %v", summary["latency_p99_ms"])
	}

	// Test GetTopEndpoints
	top, err := reg.GetTopEndpoints(since, 5)
	if err != nil {
		t.Fatalf("GetTopEndpoints returned error: %v", err)
	}
	if len(top) == 0 {
		t.Fatalf("Expected at least 1 top endpoint, got 0")
	}
	if top[0].RequestCount != 10 {
		t.Errorf("Expected top endpoint request_count == 10, got %d", top[0].RequestCount)
	}
}

func TestGetRecentRequestsPaginated(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_paginated_requests.db")

	reg, err := NewWithSQLite(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test registry: %v", err)
	}
	defer reg.db.Close()

	if err := reg.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}
	if err := reg.MigrateSchema(); err != nil {
		t.Fatalf("Failed to migrate schema: %v", err)
	}

	// Insert 15 test requests:
	// - 10x 200 GET /api/v1/users
	// - 3x 404 POST /api/v1/orders
	// - 2x 500 GET /api/v1/checkout
	for i := 1; i <= 15; i++ {
		status := 200
		method := "GET"
		path := "/api/v1/users"
		if i > 10 && i <= 13 {
			status = 404
			method = "POST"
			path = "/api/v1/orders"
		} else if i > 13 {
			status = 500
			method = "GET"
			path = "/api/v1/checkout"
		}

		dur := i * 5
		detailsJSON, _ := json.Marshal(map[string]interface{}{
			"status_code": status,
			"method":      method,
			"path":        path,
			"client_ip":   fmt.Sprintf("192.168.1.%d", i),
		})
		ip := fmt.Sprintf("192.168.1.%d", i)

		log := &AuditLog{
			ID:         fmt.Sprintf("req-%d", i),
			EventType:  "api_request",
			IPAddress:  &ip,
			DurationMs: &dur,
			Success:    status < 400,
			Details:    string(detailsJSON),
			Timestamp:  time.Now().Add(time.Duration(i) * time.Minute),
		}
		if err := reg.CreateAuditLog(log); err != nil {
			t.Fatalf("Failed to insert log %d: %v", i, err)
		}
	}

	// 1. Test Default Page 1, Limit 10 (Total 15, should return 10)
	reqs, total, err := reg.GetRecentRequestsPaginated(RequestFilter{}, 1, 10)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if total != 15 {
		t.Errorf("Expected total 15, got %d", total)
	}
	if len(reqs) != 10 {
		t.Errorf("Expected 10 items on page 1, got %d", len(reqs))
	}

	// 2. Test Page 2, Limit 10 (Total 15, should return 5)
	reqsPage2, total2, err := reg.GetRecentRequestsPaginated(RequestFilter{}, 2, 10)
	if err != nil {
		t.Fatalf("Unexpected error on page 2: %v", err)
	}
	if total2 != 15 {
		t.Errorf("Expected total 15 on page 2, got %d", total2)
	}
	if len(reqsPage2) != 5 {
		t.Errorf("Expected 5 items on page 2, got %d", len(reqsPage2))
	}

	// 3. Test Filter by Method "POST" (should return 3)
	reqsPost, totalPost, err := reg.GetRecentRequestsPaginated(RequestFilter{Method: "POST"}, 1, 10)
	if err != nil {
		t.Fatalf("Unexpected error on filter POST: %v", err)
	}
	if totalPost != 3 {
		t.Errorf("Expected total 3 for POST, got %d", totalPost)
	}
	if len(reqsPost) != 3 {
		t.Errorf("Expected 3 POST items, got %d", len(reqsPost))
	}

	// 4. Test Filter by Status "5xx" (should return 2)
	reqs5xx, total5xx, err := reg.GetRecentRequestsPaginated(RequestFilter{StatusGroup: "5xx"}, 1, 10)
	if err != nil {
		t.Fatalf("Unexpected error on filter 5xx: %v", err)
	}
	if total5xx != 2 {
		t.Errorf("Expected total 2 for 5xx, got %d", total5xx)
	}
	if len(reqs5xx) != 2 {
		t.Errorf("Expected 2 5xx items, got %d", len(reqs5xx))
	}

	// 5. Test Filter by Search "orders" (should return 3)
	reqsSearch, totalSearch, err := reg.GetRecentRequestsPaginated(RequestFilter{Search: "orders"}, 1, 10)
	if err != nil {
		t.Fatalf("Unexpected error on search orders: %v", err)
	}
	if totalSearch != 3 {
		t.Errorf("Expected total 3 for search orders, got %d", totalSearch)
	}
	if len(reqsSearch) != 3 {
		t.Errorf("Expected 3 items for search orders, got %d", len(reqsSearch))
	}
}

func TestListAuditLogsPaginated(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_paginated_audit.db")

	reg, err := NewWithSQLite(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test registry: %v", err)
	}
	defer reg.db.Close()

	if err := reg.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}
	if err := reg.MigrateSchema(); err != nil {
		t.Fatalf("Failed to migrate schema: %v", err)
	}

	// Insert 12 administrative audit logs (and 3 api_request logs that should be excluded)
	// - 5x admin_login_success
	// - 4x route_create
	// - 3x service_update
	// - 3x api_request (must be excluded)
	for i := 1; i <= 5; i++ {
		_ = reg.CreateAuditLog(&AuditLog{
			ID:        fmt.Sprintf("log-admin-%d", i),
			EventType: "admin_login_success",
			Details:   fmt.Sprintf(`{"email":"admin%d@example.com"}`, i),
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
		})
	}
	for i := 1; i <= 4; i++ {
		_ = reg.CreateAuditLog(&AuditLog{
			ID:        fmt.Sprintf("log-route-%d", i),
			EventType: "route_create",
			Details:   fmt.Sprintf(`{"route":"route-%d"}`, i),
			Timestamp: time.Now().Add(time.Duration(i+10) * time.Minute),
		})
	}
	for i := 1; i <= 3; i++ {
		_ = reg.CreateAuditLog(&AuditLog{
			ID:        fmt.Sprintf("log-service-%d", i),
			EventType: "service_update",
			Details:   fmt.Sprintf(`{"service":"service-%d"}`, i),
			Timestamp: time.Now().Add(time.Duration(i+20) * time.Minute),
		})
	}
	// api_request logs:
	for i := 1; i <= 3; i++ {
		_ = reg.CreateAuditLog(&AuditLog{
			ID:        fmt.Sprintf("log-api-%d", i),
			EventType: "api_request",
			Details:   `{"path":"/api/test"}`,
			Timestamp: time.Now().Add(time.Duration(i+30) * time.Minute),
		})
	}

	// Total admin audit logs = 5 + 4 + 3 = 12
	logs, total, err := reg.ListAuditLogsPaginated("", "", 1, 5)
	if err != nil {
		t.Fatalf("ListAuditLogsPaginated failed: %v", err)
	}
	if total != 12 {
		t.Errorf("Expected total 12, got %d", total)
	}
	if len(logs) != 5 {
		t.Errorf("Expected 5 logs on page 1, got %d", len(logs))
	}

	// Filter by event_type = "route_create" (should return 4)
	routeLogs, totalRoutes, err := reg.ListAuditLogsPaginated("route_create", "", 1, 10)
	if err != nil {
		t.Fatalf("ListAuditLogsPaginated failed on filter: %v", err)
	}
	if totalRoutes != 4 {
		t.Errorf("Expected total 4 for route_create, got %d", totalRoutes)
	}
	if len(routeLogs) != 4 {
		t.Errorf("Expected 4 logs for route_create, got %d", len(routeLogs))
	}

	// Filter by search = "service-2" (should return 1)
	searchLogs, totalSearch, err := reg.ListAuditLogsPaginated("", "service-2", 1, 10)
	if err != nil {
		t.Fatalf("ListAuditLogsPaginated failed on search: %v", err)
	}
	if totalSearch != 1 {
		t.Errorf("Expected total 1 for search, got %d", totalSearch)
	}
	if len(searchLogs) != 1 {
		t.Errorf("Expected 1 log for search, got %d", len(searchLogs))
	}
}

func TestPruneAuditLogs(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_prune_audit.db")

	reg, err := NewWithSQLite(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test registry: %v", err)
	}
	defer reg.Close()

	if err := reg.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}
	if err := reg.MigrateSchema(); err != nil {
		t.Fatalf("Failed to migrate schema: %v", err)
	}

	now := time.Now()

	// Insert 5 stale logs (20 days old, older than 14 days)
	for i := 1; i <= 5; i++ {
		err := reg.CreateAuditLog(&AuditLog{
			ID:        fmt.Sprintf("stale-%d", i),
			EventType: "api_request",
			Details:   `{"status":200}`,
			Timestamp: now.Add(-20 * 24 * time.Hour),
		})
		if err != nil {
			t.Fatalf("Failed to create stale log: %v", err)
		}
	}

	// Insert 3 fresh logs (2 days old, within 14 days)
	for i := 1; i <= 3; i++ {
		err := reg.CreateAuditLog(&AuditLog{
			ID:        fmt.Sprintf("fresh-%d", i),
			EventType: "api_request",
			Details:   `{"status":200}`,
			Timestamp: now.Add(-2 * 24 * time.Hour),
		})
		if err != nil {
			t.Fatalf("Failed to create fresh log: %v", err)
		}
	}

	// 1. Calling Prune with <= 0 should do nothing
	pruned, err := reg.PruneAuditLogs(0)
	if err != nil {
		t.Fatalf("PruneAuditLogs(0) failed: %v", err)
	}
	if pruned != 0 {
		t.Errorf("Expected 0 pruned rows when retention is 0, got %d", pruned)
	}

	// Total logs should still be 8
	allLogs, err := reg.ListAuditLogs("", 100)
	if err != nil {
		t.Fatalf("Failed to list logs: %v", err)
	}
	if len(allLogs) != 8 {
		t.Fatalf("Expected 8 total logs before pruning, got %d", len(allLogs))
	}

	// 2. Prune with 14 days retention: should prune 5 stale logs
	pruned, err = reg.PruneAuditLogs(14)
	if err != nil {
		t.Fatalf("PruneAuditLogs(14) failed: %v", err)
	}
	if pruned != 5 {
		t.Errorf("Expected 5 rows pruned, got %d", pruned)
	}

	// Verify only 3 fresh logs remain
	remainingLogs, err := reg.ListAuditLogs("", 100)
	if err != nil {
		t.Fatalf("Failed to list remaining logs: %v", err)
	}
	if len(remainingLogs) != 3 {
		t.Errorf("Expected 3 logs remaining after pruning, got %d", len(remainingLogs))
	}

	for _, l := range remainingLogs {
		if !l.Timestamp.After(now.Add(-14 * 24 * time.Hour)) {
			t.Errorf("Expected log %s to be newer than 14 days cutoff, but timestamp was %v", l.ID, l.Timestamp)
		}
	}
}



