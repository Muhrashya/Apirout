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

	// Trino
	trinoConn, err := BuildConnectionString("trino", "trino-host", 8080, "iceberg/analytics", specialUser, specialPass, "disable")
	if err != nil {
		t.Fatalf("trino BuildConnectionString failed: %v", err)
	}
	if !strings.Contains(trinoConn, "user%40corp%3Adomain") || !strings.Contains(trinoConn, "P%40ssw0rd%21%23%25%26") {
		t.Errorf("expected URL-encoded credentials in trino connection string, got: %s", trinoConn)
	}
	if !strings.Contains(trinoConn, "catalog=iceberg") || !strings.Contains(trinoConn, "schema=analytics") {
		t.Errorf("expected catalog=iceberg and schema=analytics in trino connection string, got: %s", trinoConn)
	}
	if !strings.HasPrefix(trinoConn, "http://") {
		t.Errorf("expected http:// prefix for disable sslmode, got: %s", trinoConn)
	}

	// Trino HTTPS & defaults
	trinoHttpsConn, err := BuildConnectionString("trino", "trino-host", 0, "tpch", "trino", "", "require")
	if err != nil {
		t.Fatalf("trino HTTPS BuildConnectionString failed: %v", err)
	}
	if !strings.HasPrefix(trinoHttpsConn, "https://") || !strings.Contains(trinoHttpsConn, ":8443") {
		t.Errorf("expected https:// and port 8443 for Trino SSL, got: %s", trinoHttpsConn)
	}
	if !strings.Contains(trinoHttpsConn, "catalog=tpch") || !strings.Contains(trinoHttpsConn, "schema=default") {
		t.Errorf("expected catalog=tpch and schema=default, got: %s", trinoHttpsConn)
	}
}
