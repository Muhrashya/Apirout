package executor

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// Forbidden SQL keywords for security
	forbiddenKeywordsRegex = regexp.MustCompile(`(?i)\b(INSERT|UPDATE|DELETE|ALTER|DROP|TRUNCATE|CREATE|COPY|GRANT|REVOKE|EXECUTE|CALL|PG_READ_FILE|PG_LS_DIR|PG_SLEEP|DBLINK|XP_CMDSHELL)\b`)

	multiStatementPattern    = regexp.MustCompile(`;\s*\S+`)
	limitPattern             = regexp.MustCompile(`(?i)\bLIMIT\s+\d+`)
	removeLimitOffsetPattern = regexp.MustCompile(`(?i)\s+(LIMIT|OFFSET)\s+\d+.*$`)
)

// ValidateSQL ensures query is strictly SELECT-only and free of destructive statements
func ValidateSQL(sql string) error {
	sql = strings.TrimSpace(sql)
	if sql == "" {
		return fmt.Errorf("empty query")
	}

	upperSQL := strings.ToUpper(sql)
	if !strings.HasPrefix(upperSQL, "SELECT") && !strings.HasPrefix(upperSQL, "WITH") {
		return fmt.Errorf("only SELECT queries are allowed")
	}

	if match := forbiddenKeywordsRegex.FindString(sql); match != "" {
		return fmt.Errorf("forbidden keyword detected: %s", strings.ToUpper(match))
	}

	if multiStatementPattern.MatchString(sql) {
		return fmt.Errorf("multiple statements are not allowed")
	}

	return nil
}

// HasLimit checks if query string has explicit LIMIT
func HasLimit(sql string) bool {
	return limitPattern.MatchString(sql)
}

// RemoveLimitOffset strips LIMIT and OFFSET clauses for accurate count subqueries
func RemoveLimitOffset(sql string) string {
	return removeLimitOffsetPattern.ReplaceAllString(strings.TrimRight(strings.TrimSpace(sql), ";"), "")
}

// IsSafeIdentifier checks if a column or param name consists only of valid chars
func IsSafeIdentifier(name string) bool {
	if len(name) == 0 || len(name) > 64 {
		return false
	}
	for _, r := range name {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' {
			return false
		}
	}
	return true
}

func isReservedParam(name string) bool {
	switch strings.ToLower(name) {
	case "page", "limit", "offset", "token", "order_by", "__limit", "__offset":
		return true
	default:
		return strings.HasPrefix(name, "__")
	}
}
