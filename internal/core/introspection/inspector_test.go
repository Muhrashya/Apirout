package introspection

import (
	"strings"
	"testing"
)

func TestBuildTrinoIntrospectQueries(t *testing.T) {
	// Case 1: Default / unspecified schema
	connStrDefault := "https://user@trino.example.com:443?catalog=system&schema=default"
	tblQ, colQ := buildTrinoIntrospectQueries(connStrDefault)

	if strings.Contains(tblQ, ";") {
		t.Errorf("tableQuery for Trino must not contain trailing semicolon, got: %s", tblQ)
	}
	if strings.Contains(colQ, ";") {
		t.Errorf("colQuery for Trino must not contain trailing semicolon, got: %s", colQ)
	}
	if !strings.Contains(tblQ, "NOT IN ('information_schema', 'sys')") {
		t.Errorf("expected exclusion of system schemas, got: %s", tblQ)
	}

	// Case 2: Custom user schema
	connStrCustom := "https://user@trino.example.com:443?catalog=hive&schema=analytics_prod"
	tblQ2, colQ2 := buildTrinoIntrospectQueries(connStrCustom)

	if strings.Contains(tblQ2, ";") {
		t.Errorf("tableQuery for Trino must not contain trailing semicolon, got: %s", tblQ2)
	}
	if strings.Contains(colQ2, ";") {
		t.Errorf("colQuery for Trino must not contain trailing semicolon, got: %s", colQ2)
	}
	if !strings.Contains(tblQ2, "WHERE table_schema = 'analytics_prod'") {
		t.Errorf("expected exact schema filtering for custom schema, got: %s", tblQ2)
	}
	if !strings.Contains(colQ2, "WHERE table_schema = 'analytics_prod'") {
		t.Errorf("expected exact column filtering for custom schema, got: %s", colQ2)
	}
}
