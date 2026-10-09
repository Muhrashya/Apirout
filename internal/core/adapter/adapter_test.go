package adapter

import (
	"context"
	"strings"
	"testing"
)

func TestGetAdapter_Resolution(t *testing.T) {
	tests := []struct {
		dbType      string
		wantType    string
		wantDriver  string
		expectError bool
	}{
		{dbType: "postgres", wantType: "postgres", wantDriver: "postgres"},
		{dbType: "postgresql", wantType: "postgres", wantDriver: "postgres"},
		{dbType: "POSTGRES", wantType: "postgres", wantDriver: "postgres"},
		{dbType: "mysql", wantType: "mysql", wantDriver: "mysql"},
		{dbType: "MYSQL", wantType: "mysql", wantDriver: "mysql"},
		{dbType: "mssql", wantType: "mssql", wantDriver: "sqlserver"},
		{dbType: "sqlserver", wantType: "mssql", wantDriver: "sqlserver"},
		{dbType: "MSSQL", wantType: "mssql", wantDriver: "sqlserver"},
		{dbType: "sqlite", expectError: true},
		{dbType: "oracle", expectError: true},
		{dbType: "invalid", expectError: true},
	}

	for _, tt := range tests {
		t.Run(tt.dbType, func(t *testing.T) {
			ad, err := GetAdapter(tt.dbType)
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error for %s, got nil", tt.dbType)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.dbType, err)
			}
			if ad.DBType() != tt.wantType {
				t.Errorf("got DBType %s, want %s", ad.DBType(), tt.wantType)
			}
			if ad.DriverName() != tt.wantDriver {
				t.Errorf("got DriverName %s, want %s", ad.DriverName(), tt.wantDriver)
			}
		})
	}
}

func TestDatabaseAdapter_BuildConnectionString(t *testing.T) {
	pg := NewPostgresAdapter()
	pgConn := pg.BuildConnectionString("db.internal", 5432, "app_db", "admin", "secret", "require")
	if !strings.Contains(pgConn, "postgres://admin:secret@db.internal:5432/app_db") || !strings.Contains(pgConn, "sslmode=require") {
		t.Errorf("unexpected Postgres connection string: %s", pgConn)
	}

	my := NewMySQLAdapter()
	myConn := my.BuildConnectionString("db.internal", 3306, "app_db", "root", "secret", "")
	if !strings.Contains(myConn, "root:secret@tcp(db.internal:3306)/app_db") {
		t.Errorf("unexpected MySQL connection string: %s", myConn)
	}

	ms := NewMSSQLAdapter()
	msConn := ms.BuildConnectionString("db.internal", 1433, "app_db", "sa", "P@ssw0rd!", "")
	if !strings.Contains(msConn, "sqlserver://sa:") || !strings.Contains(msConn, "database=app_db") {
		t.Errorf("unexpected MSSQL connection string: %s", msConn)
	}
}

func TestDatabaseAdapter_QuoteIdentifier(t *testing.T) {
	pg := NewPostgresAdapter()
	if pg.QuoteIdentifier("users") != `"users"` {
		t.Errorf("got %s, want %s", pg.QuoteIdentifier("users"), `"users"`)
	}

	my := NewMySQLAdapter()
	if my.QuoteIdentifier("users") != "`users`" {
		t.Errorf("got %s, want %s", my.QuoteIdentifier("users"), "`users`")
	}

	ms := NewMSSQLAdapter()
	if ms.QuoteIdentifier("users") != "[users]" {
		t.Errorf("got %s, want %s", ms.QuoteIdentifier("users"), "[users]")
	}
}

func TestDatabaseAdapter_Introspect_UnsupportedError(t *testing.T) {
	ctx := context.Background()
	_, err := GetAdapter("sqlite")
	if err == nil {
		t.Fatalf("expected sqlite adapter acquisition to fail")
	}

	// MSSQL adapter should accept connection string (connection failure expected for bogus host)
	ms := NewMSSQLAdapter()
	_, err = ms.Introspect(ctx, "sqlserver://sa:bogus@127.0.0.1:99999?database=test&connection+timeout=1")
	if err == nil {
		t.Fatalf("expected connection error for bogus mssql host, got nil")
	}
	// Error should NOT be "unsupported database type"
	if strings.Contains(err.Error(), "unsupported database type") {
		t.Errorf("mssql should be supported for introspection, but got: %v", err)
	}
}
