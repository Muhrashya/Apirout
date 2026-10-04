package monitoring

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/apirout/apirout/internal/console/registry"
)

func TestDashboardHandlerPagination(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_handlers_pagination.db")

	reg, err := registry.NewWithSQLite(dbPath)
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

	// Insert test data
	for i := 1; i <= 30; i++ {
		dur := i * 3
		details, _ := json.Marshal(map[string]interface{}{
			"status_code": 200,
			"method":      "GET",
			"path":        fmt.Sprintf("/api/v1/item/%d", i),
			"client_ip":   "127.0.0.1",
		})
		ip := "127.0.0.1"

		_ = reg.CreateAuditLog(&registry.AuditLog{
			ID:         fmt.Sprintf("req-%d", i),
			EventType:  "api_request",
			IPAddress:  &ip,
			DurationMs: &dur,
			Success:    true,
			Details:    string(details),
			Timestamp:  time.Now().Add(time.Duration(i) * time.Minute),
		})
	}

	for i := 1; i <= 10; i++ {
		_ = reg.CreateAuditLog(&registry.AuditLog{
			ID:        fmt.Sprintf("act-%d", i),
			EventType: "admin_login_success",
			Details:   `{"email":"admin@apirout.io"}`,
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
		})
	}

	svc := NewService(reg)
	handler := NewDashboardHandler(svc)

	// 1. Test Requests pagination
	t.Run("Requests pagination page 1 limit 10", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/monitoring/requests?page=1&limit=10", nil)
		rec := httptest.NewRecorder()

		handler.Requests(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			Data       []registry.RecentRequest `json:"data"`
			Pagination struct {
				Total      int `json:"total"`
				Page       int `json:"page"`
				Limit      int `json:"limit"`
				TotalPages int `json:"total_pages"`
			} `json:"pagination"`
		}

		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if resp.Pagination.Total != 30 {
			t.Errorf("Expected total 30, got %d", resp.Pagination.Total)
		}
		if resp.Pagination.Page != 1 {
			t.Errorf("Expected page 1, got %d", resp.Pagination.Page)
		}
		if resp.Pagination.Limit != 10 {
			t.Errorf("Expected limit 10, got %d", resp.Pagination.Limit)
		}
		if resp.Pagination.TotalPages != 3 {
			t.Errorf("Expected total_pages 3, got %d", resp.Pagination.TotalPages)
		}
		if len(resp.Data) != 10 {
			t.Errorf("Expected 10 items in data slice, got %d", len(resp.Data))
		}
	})

	// 2. Test AuditLogs pagination
	t.Run("AuditLogs pagination page 1 limit 5", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/audit-logs?page=1&limit=5&event_type=admin_login_success", nil)
		rec := httptest.NewRecorder()

		handler.AuditLogs(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			Data       []registry.AuditLog `json:"data"`
			Pagination struct {
				Total      int `json:"total"`
				Page       int `json:"page"`
				Limit      int `json:"limit"`
				TotalPages int `json:"total_pages"`
			} `json:"pagination"`
		}

		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if resp.Pagination.Total != 10 {
			t.Errorf("Expected total 10, got %d", resp.Pagination.Total)
		}
		if resp.Pagination.TotalPages != 2 {
			t.Errorf("Expected total_pages 2, got %d", resp.Pagination.TotalPages)
		}
		if len(resp.Data) != 5 {
			t.Errorf("Expected 5 items in data slice, got %d", len(resp.Data))
		}
	})
}
