package executor

import (
	"regexp"
	"strings"
)

// RemoveLimitOffset removes LIMIT and OFFSET clauses from a SQL query
// This is used for generating accurate count queries when wrapping a query that has a hardcoded LIMIT
func RemoveLimitOffset(sql string) string {
	// Need to be careful with subqueries, but a regex for trailing LIMIT/OFFSET is a good heuristic for now.
	// Matches case-insensitive LIMIT N [OFFSET M] at the end of string.
	re := regexp.MustCompile(`(?i)\s+LIMIT\s+\d+(\s+OFFSET\s+\d+)?\s*$`)
	return re.ReplaceAllString(sql, "")
}

// PrepareCustomCountSQL appends dynamic filters to a custom count query
func PrepareCustomCountSQL(countSQL string, dynamicWhere []string) string {
	if len(dynamicWhere) == 0 {
		return countSQL
	}

	filterClause := strings.Join(dynamicWhere, " AND ")
	upperSQL := strings.ToUpper(countSQL)

	// Simple heuristic: if WHERE exists, append with AND. If not, append with WHERE.
	// This is not perfect (e.g. subqueries, GROUP BY) but covers 90% use cases.
	// A better approach would be to let users use a variable like {{filters}} in their custom SQL,
	// but for now we'll try to be smart.
	
	// If it has GROUP BY, ORDER BY, or LIMIT at the end, handling becomes tricky.
	// Assuming Count SQL is simple like "SELECT COUNT(*) FROM table WHERE ..."
	
	if strings.Contains(upperSQL, " WHERE ") {
		return countSQL + " AND " + filterClause
	}
	
	return countSQL + " WHERE " + filterClause
}
