package gitops

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/apirout/apirout/internal/console/auth"
)

func TestHandler_ImportHTTPSeam(t *testing.T) {
	reg := setupTestRegistry(t)
	defer reg.Close()

	h := NewHandler(reg, nil)

	// Helper to attach SUPER_ADMIN claims
	withAdmin := func(req *http.Request) *http.Request {
		claims := &auth.Claims{
			UserID: "admin-1",
			Email:  "admin@apirout.io",
			Role:   "SUPER_ADMIN",
		}
		ctx := context.WithValue(req.Context(), auth.UserContextKey, claims)
		return req.WithContext(ctx)
	}

	// 1. Forbidden for non-superadmin
	reqNonAdmin := httptest.NewRequest("POST", "/api/v1/gitops/import", bytes.NewBufferString("{}"))
	recNonAdmin := httptest.NewRecorder()
	h.Import(recNonAdmin, reqNonAdmin)
	if recNonAdmin.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for unauthenticated request, got: %d", recNonAdmin.Code)
	}

	// 2. Unprocessable entity on missing dependency
	invalidYAML := `
version: "api.apirout.io/v1"
services:
  - name: "orphan-service"
    upstream_name: "phantom-upstream"
`
	reqInvalid := httptest.NewRequest("POST", "/api/v1/gitops/import", bytes.NewBufferString(invalidYAML))
	reqInvalid.Header.Set("Content-Type", "application/x-yaml")
	reqInvalid = withAdmin(reqInvalid)
	recInvalid := httptest.NewRecorder()

	h.Import(recInvalid, reqInvalid)
	if recInvalid.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity, got: %d (body: %s)", recInvalid.Code, recInvalid.Body.String())
	}

	var errBody map[string]interface{}
	if err := json.Unmarshal(recInvalid.Body.Bytes(), &errBody); err != nil {
		t.Fatalf("failed to parse JSON error: %v", err)
	}
	if errBody["entity"] != "Service" || errBody["identifier"] != "orphan-service" {
		t.Fatalf("unexpected error response body: %v", errBody)
	}

	// 3. Successful import with valid YAML
	validYAML := `
version: "api.apirout.io/v1"
groups:
  - name: "gitops-group"
upstreams:
  - name: "gitops-upstream"
services:
  - name: "gitops-service"
    upstream_name: "gitops-upstream"
endpoints:
  - path: "/api/gitops"
    service_name: "gitops-service"
    method: "GET"
`
	reqValid := httptest.NewRequest("POST", "/api/v1/gitops/import", bytes.NewBufferString(validYAML))
	reqValid.Header.Set("Content-Type", "application/x-yaml")
	reqValid = withAdmin(reqValid)
	recValid := httptest.NewRecorder()

	h.Import(recValid, reqValid)
	if recValid.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for valid import, got: %d (body: %s)", recValid.Code, recValid.Body.String())
	}

	var successBody map[string]interface{}
	if err := json.Unmarshal(recValid.Body.Bytes(), &successBody); err != nil {
		t.Fatalf("failed to decode success response: %v", err)
	}
	if successBody["status"] != true {
		t.Fatalf("expected status true, got: %v", successBody)
	}

	// 4. Oversized payload (SEC-13 Declarative Stream Shield) -> 400 Bad Request
	oversized := make([]byte, 6*1024*1024)
	copy(oversized, []byte(`{"version":"api.apirout.io/v1","groups":[`))
	for i := len(`{"version":"api.apirout.io/v1","groups":[`); i < len(oversized)-1; i++ {
		oversized[i] = ' '
	}
	oversized[len(oversized)-1] = ']'
	reqOversized := httptest.NewRequest("POST", "/api/v1/gitops/import", bytes.NewReader(oversized))
	reqOversized.Header.Set("Content-Type", "application/json")
	reqOversized = withAdmin(reqOversized)
	recOversized := httptest.NewRecorder()

	h.Import(recOversized, reqOversized)
	if recOversized.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for truncated oversized payload, got: %d", recOversized.Code)
	}
}

