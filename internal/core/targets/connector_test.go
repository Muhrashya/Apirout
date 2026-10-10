package targets

import (
	"strings"
	"testing"
)

func TestBuildConnectionString_EscapesSpecialCharacters(t *testing.T) {
	specialUser := "user@corp:domain"
	specialPass := "P@ssw0rd!#%&"

	// Postgres
	pgConn, err := BuildConnectionString("postgres", "localhost", 5432, "testdb", specialUser, specialPass, "disable")
	if err != nil {
		t.Fatalf("postgres BuildConnectionString failed: %v", err)
	}
	if strings.Contains(pgConn, "user@corp:domain") || strings.Contains(pgConn, "P@ssw0rd!#%&") {
		t.Errorf("postgres connection string contains unescaped credentials: %s", pgConn)
	}
	if !strings.Contains(pgConn, "user%40corp%3Adomain") || !strings.Contains(pgConn, "P%40ssw0rd%21%23%25%26") {
		t.Errorf("expected URL-encoded credentials in postgres connection string, got: %s", pgConn)
	}

	// MySQL
	myConn, err := BuildConnectionString("mysql", "localhost", 3306, "testdb", specialUser, specialPass, "")
	if err != nil {
		t.Fatalf("mysql BuildConnectionString failed: %v", err)
	}
	if !strings.Contains(myConn, "user%40corp%3Adomain") || !strings.Contains(myConn, "P%40ssw0rd%21%23%25%26") {
		t.Errorf("expected URL-encoded credentials in mysql connection string, got: %s", myConn)
	}

	// MSSQL
	msConn, err := BuildConnectionString("mssql", "localhost", 1433, "testdb", specialUser, specialPass, "")
	if err != nil {
		t.Fatalf("mssql BuildConnectionString failed: %v", err)
	}
	if !strings.Contains(msConn, "user%40corp%3Adomain") || !strings.Contains(msConn, "P%40ssw0rd%21%23%25%26") {
		t.Errorf("expected URL-encoded credentials in mssql connection string, got: %s", msConn)
	}

	// Redis
	redisConn, err := BuildConnectionString("redis", "localhost", 6379, "0", specialUser, specialPass, "")
	if err != nil {
		t.Fatalf("redis BuildConnectionString failed: %v", err)
	}
	if !strings.Contains(redisConn, "user%40corp%3Adomain") || !strings.Contains(redisConn, "P%40ssw0rd%21%23%25%26") {
		t.Errorf("expected URL-encoded credentials in redis connection string, got: %s", redisConn)
	}
}
