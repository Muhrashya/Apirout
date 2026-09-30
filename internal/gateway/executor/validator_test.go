package executor

import (
	"testing"
)

func TestValidateSQL_ValidQueries(t *testing.T) {
	validQueries := []string{
		"SELECT * FROM users",
		"SELECT id, name, email FROM accounts WHERE status = 'active'",
		"WITH sales_summary AS (SELECT department, SUM(amount) AS total FROM sales GROUP BY department) SELECT * FROM sales_summary",
		"  select count(1) from logs  ",
	}

	for _, q := range validQueries {
		if err := ValidateSQL(q); err != nil {
			t.Errorf("expected valid query %q, got error: %v", q, err)
		}
	}
}

func TestValidateSQL_ForbiddenKeywords(t *testing.T) {
	forbiddenQueries := []string{
		"INSERT INTO users (name) VALUES ('alice')",
		"UPDATE users SET name = 'bob'",
		"DELETE FROM users WHERE id = 1",
		"DROP TABLE users",
		"TRUNCATE TABLE users",
		"ALTER TABLE users ADD COLUMN age int",
		"SELECT * FROM users; SELECT pg_sleep(5)",
		"SELECT xp_cmdshell('dir')",
	}

	for _, q := range forbiddenQueries {
		if err := ValidateSQL(q); err == nil {
			t.Errorf("expected query %q to be rejected, but it was allowed", q)
		}
	}
}

func TestValidateSQL_MultipleStatements(t *testing.T) {
	multiStatementQueries := []string{
		"SELECT * FROM users; SELECT * FROM products",
		"SELECT 1; DROP DATABASE test",
	}

	for _, q := range multiStatementQueries {
		if err := ValidateSQL(q); err == nil {
			t.Errorf("expected multi-statement query %q to be rejected", q)
		}
	}
}

func TestHasLimit(t *testing.T) {
	if !HasLimit("SELECT * FROM users LIMIT 10") {
		t.Errorf("expected HasLimit to return true")
	}
	if !HasLimit("SELECT * FROM users limit 50") {
		t.Errorf("expected HasLimit to return true (case-insensitive)")
	}
	if HasLimit("SELECT * FROM users") {
		t.Errorf("expected HasLimit to return false")
	}
}

func TestRemoveLimitOffset(t *testing.T) {
	query := "SELECT * FROM users LIMIT 20 OFFSET 40"
	expected := "SELECT * FROM users"
	result := RemoveLimitOffset(query)
	if result != expected {
		t.Errorf("RemoveLimitOffset(%q) = %q, expected %q", query, result, expected)
	}
}

func TestIsSafeIdentifier(t *testing.T) {
	valid := []string{"user_id", "created_at", "Status1", "col_name_test"}
	for _, id := range valid {
		if !IsSafeIdentifier(id) {
			t.Errorf("expected %q to be safe identifier", id)
		}
	}

	invalid := []string{"", "table.col", "user;id", "user name", "col-name", "col$"}
	for _, id := range invalid {
		if IsSafeIdentifier(id) {
			t.Errorf("expected %q to be unsafe identifier", id)
		}
	}
}
