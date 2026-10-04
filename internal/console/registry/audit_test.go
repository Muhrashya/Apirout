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
