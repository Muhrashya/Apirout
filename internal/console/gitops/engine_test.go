package gitops

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/apirout/apirout/internal/console/registry"
)

func setupTestRegistry(t *testing.T) *registry.Registry {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_gitops.db")

	reg, err := registry.NewWithSQLite(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test registry: %v", err)
	}

	if err := reg.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}
	if err := reg.MigrateSchema(); err != nil {
		t.Fatalf("Failed to migrate schema: %v", err)
	}

	return reg
}

func TestDeclarativeEngine_IngestValidSnapshotAndIdempotency(t *testing.T) {
	reg := setupTestRegistry(t)
	defer reg.Close()

	engine := NewDeclarativeEngine(reg.DB(), reg.DBType(), nil)

	descGroup := "Core API Group"
	descService := "Users Microservice"
	descConsumer := "Mobile Application Client"
	customID := "cust-99"

	snapshot := &MetadataSnapshot{
		Version: "api.apirout.io/v1",
		Groups: []GroupSpec{
			{Name: "core-group", Description: &descGroup},
		},
		Connections: []ConnectionSpec{
			{
				Name:      "main-postgres",
				DBType:    "postgres",
				Host:      "localhost",
				Port:      5432,
				DBName:    "app_db",
				Username:  "admin",
				SSLMode:   "disable",
				GroupName: "core-group",
			},
		},
		Upstreams: []UpstreamSpec{
			{
				Name:      "user-upstream",
				Algorithm: "round-robin",
				Slots:     10000,
				Enabled:   true,
				Targets: []TargetSpec{
					{Target: "10.0.0.1:8080", Host: "10.0.0.1", Port: 8080, Weight: 100, Enabled: true},
					{Target: "10.0.0.2:8080", Host: "10.0.0.2", Port: 8080, Weight: 100, Enabled: true},
				},
			},
		},
		Services: []ServiceSpec{
			{
				Name:         "user-service",
				Description:  &descService,
				Protocol:     "http",
				Path:         "/v1",
				Enabled:      true,
				UpstreamName: "user-upstream",
			},
		},
		Consumers: []ConsumerSpec{
			{
				Username:    "mobile-app",
				CustomID:    &customID,
				Description: &descConsumer,
				GroupName:   "core-group",
				Status:      "active",
			},
		},
		Queries: []QuerySpec{
			{
				Name:         "get-active-users",
				GroupName:    "core-group",
				Connection:   "main-postgres",
				SQLText:      "SELECT id, name FROM users WHERE active = true",
				DefaultLimit: 50,
			},
		},
		Endpoints: []EndpointSpec{
			{
				Path:        "/api/users",
				Name:        "Get Users",
				Method:      "GET",
				ServiceName: "user-service",
				GroupName:   "core-group",
				Enabled:     true,
				IsPublic:    true,
			},
			{
				Path:      "/api/queries/users",
				Name:      "Query Users",
				Method:    "GET",
				QueryName: "get-active-users",
				GroupName: "core-group",
				Enabled:   true,
				IsPublic:  false,
			},
		},
	}

	// 1. Initial ingestion
	res, err := engine.Ingest(context.Background(), snapshot)
	if err != nil {
		t.Fatalf("Ingest failed unexpectedly: %v", err)
	}

	if res.GroupsCount != 1 || res.ConnectionsCount != 1 || res.UpstreamsCount != 1 ||
		res.TargetsCount != 2 || res.ServicesCount != 1 || res.ConsumersCount != 1 ||
		res.QueriesCount != 1 || res.EndpointsCount != 2 {
		t.Fatalf("unexpected ingestion result counts: %+v", res)
	}

	// Verify database persistence via registry read APIs
	groups, err := reg.ListGroups()
	if err != nil || len(groups) != 1 || groups[0].Name != "core-group" {
		t.Fatalf("expected 1 group persisted, got: %v (err: %v)", groups, err)
	}

	services, err := reg.ListServices()
	if err != nil || len(services) != 1 || services[0].Name != "user-service" {
		t.Fatalf("expected 1 service persisted, got: %v (err: %v)", services, err)
	}
	if services[0].UpstreamName == nil || *services[0].UpstreamName != "user-upstream" {
		t.Fatalf("expected service upstream linked to user-upstream, got: %v", services[0].UpstreamName)
	}

	endpoints, err := reg.ListEndpoints()
	if err != nil || len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints persisted, got: %d", len(endpoints))
	}

	// 2. Idempotency test: Re-ingest the exact same snapshot
	res2, err := engine.Ingest(context.Background(), snapshot)
	if err != nil {
		t.Fatalf("Re-ingest failed on idempotency test: %v", err)
	}
	if res2.EndpointsCount != 2 {
		t.Fatalf("expected 2 endpoints updated idempotently, got: %d", res2.EndpointsCount)
	}

	// Verify total count in DB didn't duplicate
	endpointsAfter, err := reg.ListEndpoints()
	if err != nil || len(endpointsAfter) != 2 {
		t.Fatalf("expected endpoints count to remain 2 after re-ingest, got: %d", len(endpointsAfter))
	}
}

func TestDeclarativeEngine_AtomicRollbackOnMissingDependency(t *testing.T) {
	reg := setupTestRegistry(t)
	defer reg.Close()

	engine := NewDeclarativeEngine(reg.DB(), reg.DBType(), nil)

	// Snapshot where Service references a nonexistent Upstream
	snapshot := &MetadataSnapshot{
		Version: "api.apirout.io/v1",
		Groups: []GroupSpec{
			{Name: "valid-group"},
		},
		Connections: []ConnectionSpec{
			{
				Name:      "valid-conn",
				DBType:    "postgres",
				Host:      "localhost",
				Port:      5432,
				DBName:    "db",
				Username:  "usr",
				GroupName: "valid-group",
			},
		},
		Services: []ServiceSpec{
			{
				Name:         "broken-service",
				UpstreamName: "nonexistent-upstream", // DOES NOT EXIST
			},
		},
	}

	res, err := engine.Ingest(context.Background(), snapshot)
	if err == nil {
		t.Fatalf("expected ingestion to fail with ValidationError, got success: %+v", res)
	}

	valErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected error of type *ValidationError, got: %T (%v)", err, err)
	}

	if valErr.Entity != "Service" || valErr.Identifier != "broken-service" {
		t.Fatalf("expected ValidationError on Service broken-service, got: %+v", valErr)
	}

	// Verify ATOMIC ROLLBACK: valid-group and valid-conn must NOT be persisted in the DB!
	groups, err := reg.ListGroups()
	if err != nil || len(groups) != 0 {
		t.Fatalf("expected groups to be rolled back to 0, got: %d", len(groups))
	}

	conns, err := reg.ListConnections()
	if err != nil || len(conns) != 0 {
		t.Fatalf("expected connections to be rolled back to 0, got: %d", len(conns))
	}
}

func TestDeclarativeEngine_AtomicRollbackOnMissingConnectionForQuery(t *testing.T) {
	reg := setupTestRegistry(t)
	defer reg.Close()

	engine := NewDeclarativeEngine(reg.DB(), reg.DBType(), nil)

	snapshot := &MetadataSnapshot{
		Version: "api.apirout.io/v1",
		Groups: []GroupSpec{
			{Name: "group-1"},
		},
		Queries: []QuerySpec{
			{
				Name:       "broken-query",
				GroupName:  "group-1",
				Connection: "ghost-conn", // DOES NOT EXIST
				SQLText:    "SELECT 1",
			},
		},
	}

	_, err := engine.Ingest(context.Background(), snapshot)
	if err == nil {
		t.Fatal("expected ingestion to fail due to missing connection")
	}

	valErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected *ValidationError, got: %T (%v)", err, err)
	}
	if valErr.Entity != "Query" || valErr.Identifier != "broken-query" {
		t.Fatalf("expected failure on Query broken-query, got: %+v", valErr)
	}

	// Verify rollback
	groups, _ := reg.ListGroups()
	if len(groups) != 0 {
		t.Fatalf("expected groups rolled back, got: %d", len(groups))
	}
}
