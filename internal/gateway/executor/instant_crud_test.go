package executor

import (
	"strings"
	"testing"

	"github.com/apirout/apirout/internal/core/adapter"
	"github.com/apirout/apirout/internal/core/sqldialect"
)

func TestInstantCRUD_DbTypeDriver(t *testing.T) {
	tests := []struct {
		dbType     string
		wantDriver string
		wantErr    bool
	}{
		{dbType: "postgres", wantDriver: "postgres"},
		{dbType: "postgresql", wantDriver: "postgres"},
		{dbType: "mysql", wantDriver: "mysql"},
		{dbType: "mssql", wantDriver: "sqlserver"},
		{dbType: "sqlserver", wantDriver: "sqlserver"},
		{dbType: "sqlite", wantErr: true},
		{dbType: "oracle", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.dbType, func(t *testing.T) {
			driver, err := dbTypeDriver(tt.dbType)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for dbType %s, got nil", tt.dbType)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.dbType, err)
			}
			if driver != tt.wantDriver {
				t.Errorf("got driver %s, want %s", driver, tt.wantDriver)
			}
		})
	}
}

func TestInstantCRUD_QuoteIdentifier(t *testing.T) {
	if quoteIdentifier("postgres", "users") != `"users"` {
		t.Errorf("postgres quote: got %s, want %s", quoteIdentifier("postgres", "users"), `"users"`)
	}
	if quoteIdentifier("mysql", "users") != "`users`" {
		t.Errorf("mysql quote: got %s, want %s", quoteIdentifier("mysql", "users"), "`users`")
	}
	if quoteIdentifier("mssql", "users") != "[users]" {
		t.Errorf("mssql quote: got %s, want %s", quoteIdentifier("mssql", "users"), "[users]")
	}
	if quoteIdentifier("sqlserver", "orders") != "[orders]" {
		t.Errorf("sqlserver quote: got %s, want %s", quoteIdentifier("sqlserver", "orders"), "[orders]")
	}
}

func TestInstantCRUD_DialectGenerators_AllEngines(t *testing.T) {
	engines := []string{"postgres", "mysql", "mssql"}

	for _, eng := range engines {
		t.Run(eng, func(t *testing.T) {
			ad, err := adapter.GetAdapter(eng)
			if err != nil {
				t.Fatalf("failed to get adapter for %s: %v", eng, err)
			}
			gen := ad.GetQueryGenerator()

			// 1. SELECT query
			selQuery, selArgs, err := gen.BuildSelect(sqldialect.QuerySpec{
				Table:   "customers",
				Columns: []string{"id", "name", "email"},
				Filters: []sqldialect.FilterCondition{
					{Column: "status", Operator: sqldialect.OpEq, Value: "active"},
				},
				Limit:  10,
				Offset: 0,
			})
			if err != nil {
				t.Fatalf("BuildSelect failed: %v", err)
			}
			if !strings.Contains(selQuery, "customers") || len(selArgs) != 1 {
				t.Errorf("invalid select query %s with args %+v", selQuery, selArgs)
			}

			// 2. INSERT query
			insQuery, insArgs, err := gen.BuildInsert(sqldialect.InsertSpec{
				Table: "customers",
				Values: map[string]any{
					"name":  "Alice",
					"email": "alice@example.com",
				},
			})
			if err != nil {
				t.Fatalf("BuildInsert failed: %v", err)
			}
			if !strings.Contains(insQuery, "INSERT INTO") || len(insArgs) != 2 {
				t.Errorf("invalid insert query %s with args %+v", insQuery, insArgs)
			}

			// 3. UPDATE query
			updQuery, updArgs, err := gen.BuildUpdate(sqldialect.UpdateSpec{
				Table: "customers",
				Values: map[string]any{
					"name": "Alice Cooper",
				},
				Filters: []sqldialect.FilterCondition{
					{Column: "id", Operator: sqldialect.OpEq, Value: "42"},
				},
			})
			if err != nil {
				t.Fatalf("BuildUpdate failed: %v", err)
			}
			if !strings.Contains(updQuery, "UPDATE") || len(updArgs) != 2 {
				t.Errorf("invalid update query %s with args %+v", updQuery, updArgs)
			}

			// 4. DELETE query
			delQuery, delArgs, err := gen.BuildDelete(sqldialect.DeleteSpec{
				Table: "customers",
				Filters: []sqldialect.FilterCondition{
					{Column: "id", Operator: sqldialect.OpEq, Value: "42"},
				},
			})
			if err != nil {
				t.Fatalf("BuildDelete failed: %v", err)
			}
			if !strings.Contains(delQuery, "DELETE FROM") || len(delArgs) != 1 {
				t.Errorf("invalid delete query %s with args %+v", delQuery, delArgs)
			}
		})
	}
}
