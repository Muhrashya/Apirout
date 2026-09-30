package executor

import (
	"encoding/json"
	"fmt"
	"strings"
)

type filterOp struct {
	suffix string
	op     string
}

var supportedOperators = []filterOp{
	{suffix: "_gte", op: ">="},
	{suffix: "_lte", op: "<="},
	{suffix: "_neq", op: "<>"},
	{suffix: "_ne", op: "<>"},
	{suffix: "_gt", op: ">"},
	{suffix: "_lt", op: "<"},
	{suffix: "_ilike", op: "ILIKE"},
	{suffix: "_like", op: "LIKE"},
	{suffix: "_in", op: "IN"},
}

// BuildDynamicWhere constructs safe parameterized WHERE clauses supporting comparison operators
func BuildDynamicWhere(dbType string, params map[string]interface{}, paramsSchemaJSON string) []string {
	var whereClauses []string
	if len(params) == 0 {
		return whereClauses
	}

	// 1. Build whitelist map from ParamsSchema (lowercase -> original column name)
	var whitelist map[string]string
	if paramsSchemaJSON != "" && paramsSchemaJSON != "{}" {
		whitelist = make(map[string]string)
		var schema []struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal([]byte(paramsSchemaJSON), &schema); err == nil {
			for _, item := range schema {
				whitelist[strings.ToLower(item.Name)] = item.Name
			}
		}
	}

	// 2. Determine quoting character based on database type
	quoteChar := `"`
	if dbType == "mysql" {
		quoteChar = "`"
	}

	for key, val := range params {
		if isReservedParam(key) {
			continue
		}

		// Detect operator suffix
		colKey := key
		op := "="
		for _, o := range supportedOperators {
			if strings.HasSuffix(key, o.suffix) {
				colKey = strings.TrimSuffix(key, o.suffix)
				op = o.op
				break
			}
		}

		if !IsSafeIdentifier(colKey) || !IsSafeIdentifier(key) {
			continue
		}

		colName := colKey
		if whitelist != nil {
			if originalName, ok := whitelist[strings.ToLower(colKey)]; ok {
				colName = originalName
			} else {
				// Not in whitelist, skip for security
				continue
			}
		}

		quotedCol := fmt.Sprintf("%s%s%s", quoteChar, colName, quoteChar)

		switch op {
		case "IN":
			// Handle comma-separated IN values (e.g. ?status_in=AKTIF,PENDING)
			if strVal, ok := val.(string); ok {
				parts := strings.Split(strVal, ",")
				placeholders := make([]string, len(parts))
				for i, p := range parts {
					paramName := fmt.Sprintf("%s_%d", key, i)
					params[paramName] = strings.TrimSpace(p)
					placeholders[i] = ":" + paramName
				}
				delete(params, key) // replace raw comma string
				whereClauses = append(whereClauses, fmt.Sprintf("%s IN (%s)", quotedCol, strings.Join(placeholders, ", ")))
			}
		case "ILIKE":
			if dbType == "mysql" || dbType == "mssql" {
				// MySQL & MSSQL are usually case-insensitive by default with LIKE
				whereClauses = append(whereClauses, fmt.Sprintf("LOWER(%s) LIKE LOWER(:%s)", quotedCol, key))
			} else {
				whereClauses = append(whereClauses, fmt.Sprintf("%s ILIKE :%s", quotedCol, key))
			}
		default:
			whereClauses = append(whereClauses, fmt.Sprintf("%s %s :%s", quotedCol, op, key))
		}
	}

	return whereClauses
}
