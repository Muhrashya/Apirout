package executor

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// Forbidden SQL keywords (case-insensitive)
	forbiddenKeywordsRegex = regexp.MustCompile(`(?i)\b(INSERT|UPDATE|DELETE|ALTER|DROP|TRUNCATE|CREATE|COPY|GRANT|REVOKE|EXECUTE|CALL|PG_READ_FILE|PG_LS_DIR|PG_SLEEP|DBLINK|XML)\b`)

	// Pattern to detect multiple statements (semicolon followed by more SQL)
	multiStatementPattern = regexp.MustCompile(`;\s*\S+`)

	// Pattern to check if query already has LIMIT
	limitPattern = regexp.MustCompile(`(?i)\bLIMIT\s+\d+`)
)

// ValidateSQL validates that a SQL query is SELECT-only
func ValidateSQL(sql string) error {
	sql = strings.TrimSpace(sql)

	if sql == "" {
		return fmt.Errorf("empty query")
	}

	// Convert to uppercase for keyword checking
	upperSQL := strings.ToUpper(sql)

	// Must start with SELECT or WITH (for CTEs)
	if !strings.HasPrefix(upperSQL, "SELECT") && !strings.HasPrefix(upperSQL, "WITH") {
		return fmt.Errorf("only SELECT queries are allowed")
	}

	// Check for forbidden keywords
	if match := forbiddenKeywordsRegex.FindString(sql); match != "" {
		return fmt.Errorf("forbidden keyword detected: %s", strings.ToUpper(match))
	}

	// Check for multiple statements
	if multiStatementPattern.MatchString(sql) {
		return fmt.Errorf("multiple statements are not allowed")
	}

	return nil
}

// WrapWithLimit wraps a query with a LIMIT clause if it doesn't have one
func WrapWithLimit(sql string, limit int) string {
	sql = strings.TrimSpace(sql)

	// Remove trailing semicolon if present
	sql = strings.TrimSuffix(sql, ";")

	// If query already has a LIMIT, don't add another
	if limitPattern.MatchString(sql) {
		return sql
	}

	// Wrap in subquery and add limit
	return fmt.Sprintf("SELECT * FROM (%s) AS _limited_query LIMIT %d", sql, limit)
}

// WrapWithPagination wraps a query with LIMIT and OFFSET
func WrapWithPagination(sql string, limit, offset int) string {
	sql = strings.TrimSpace(sql)
	sql = strings.TrimSuffix(sql, ";")

	// Always wrap to safely apply offset and limit preference
	return fmt.Sprintf("SELECT * FROM (%s) AS _paginated_query LIMIT %d OFFSET %d", sql, limit, offset)
}

// SanitizeSQL performs basic SQL sanitization
func SanitizeSQL(sql string) string {
	sql = strings.TrimSpace(sql)
	sql = strings.TrimSuffix(sql, ";")
	return sql
}
