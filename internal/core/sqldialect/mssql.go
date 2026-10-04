package sqldialect

import (
	"fmt"
	"sort"
	"strings"
)

// MSSQLGenerator generates Microsoft SQL Server dialect SQL
type MSSQLGenerator struct{}

// NewMSSQLGenerator creates a new MSSQL generator
func NewMSSQLGenerator() *MSSQLGenerator {
	return &MSSQLGenerator{}
}

func (g *MSSQLGenerator) Dialect() string {
	return "sqlserver"
}

func (g *MSSQLGenerator) QuoteIdentifier(ident string) string {
	clean := strings.ReplaceAll(ident, "]", "]]")
	return "[" + clean + "]"
}

func (g *MSSQLGenerator) Placeholder(index int) string {
	return fmt.Sprintf("@p%d", index)
}

func (g *MSSQLGenerator) BuildSelect(spec QuerySpec) (string, []any, error) {
	if spec.Table == "" {
		return "", nil, fmt.Errorf("table name is required")
	}

	var sb strings.Builder
	var args []any
	argIdx := 1

	// If no offset is specified and we have a limit, TOP can be used if no OrderBy, but OFFSET/FETCH is cleaner and uniform in modern MSSQL
	sb.WriteString("SELECT ")
	if len(spec.Columns) == 0 {
		sb.WriteString("*")
	} else {
		cols := make([]string, len(spec.Columns))
		for i, c := range spec.Columns {
			cols[i] = g.QuoteIdentifier(c)
		}
		sb.WriteString(strings.Join(cols, ", "))
	}

	// FROM
	sb.WriteString(" FROM ")
	sb.WriteString(g.QuoteIdentifier(spec.Table))

	// WHERE
	if len(spec.Filters) > 0 {
		sb.WriteString(" WHERE ")
		whereParts, whereArgs, nextIdx := g.buildFilters(spec.Filters, argIdx)
		sb.WriteString(strings.Join(whereParts, " AND "))
		args = append(args, whereArgs...)
		argIdx = nextIdx
	}

	// ORDER BY
	if len(spec.OrderBy) > 0 {
		sb.WriteString(" ORDER BY ")
		orderParts := make([]string, len(spec.OrderBy))
		for i, o := range spec.OrderBy {
			dir := "ASC"
			if strings.ToUpper(string(o.Direction)) == "DESC" {
				dir = "DESC"
			}
			orderParts[i] = fmt.Sprintf("%s %s", g.QuoteIdentifier(o.Column), dir)
		}
		sb.WriteString(strings.Join(orderParts, ", "))
	} else if spec.Limit > 0 || spec.Offset > 0 {
		// MSSQL requires ORDER BY for OFFSET/FETCH
		sb.WriteString(" ORDER BY (SELECT NULL)")
	}

	// OFFSET & FETCH
	if spec.Limit > 0 || spec.Offset > 0 {
		offset := spec.Offset
		if offset < 0 {
			offset = 0
		}
		sb.WriteString(fmt.Sprintf(" OFFSET %d ROWS", offset))
		if spec.Limit > 0 {
			sb.WriteString(fmt.Sprintf(" FETCH NEXT %d ROWS ONLY", spec.Limit))
		}
	}

	return sb.String(), args, nil
}

func (g *MSSQLGenerator) BuildInsert(spec InsertSpec) (string, []any, error) {
	if spec.Table == "" {
		return "", nil, fmt.Errorf("table name is required")
	}
	if len(spec.Values) == 0 {
		return "", nil, fmt.Errorf("no insert values provided")
	}

	var sb strings.Builder
	var args []any
	argIdx := 1

	keys := make([]string, 0, len(spec.Values))
	for k := range spec.Values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	quotedCols := make([]string, len(keys))
	placeholders := make([]string, len(keys))
	for i, k := range keys {
		quotedCols[i] = g.QuoteIdentifier(k)
		placeholders[i] = g.Placeholder(argIdx)
		args = append(args, spec.Values[k])
		argIdx++
	}

	sb.WriteString("INSERT INTO ")
	sb.WriteString(g.QuoteIdentifier(spec.Table))
	sb.WriteString(" (")
	sb.WriteString(strings.Join(quotedCols, ", "))
	sb.WriteString(")")

	if len(spec.Returning) > 0 {
		sb.WriteString(" OUTPUT ")
		if len(spec.Returning) == 1 && spec.Returning[0] == "*" {
			sb.WriteString("INSERTED.*")
		} else {
			outCols := make([]string, len(spec.Returning))
			for i, r := range spec.Returning {
				outCols[i] = "INSERTED." + g.QuoteIdentifier(r)
			}
			sb.WriteString(strings.Join(outCols, ", "))
		}
	}

	sb.WriteString(" VALUES (")
	sb.WriteString(strings.Join(placeholders, ", "))
	sb.WriteString(")")

	return sb.String(), args, nil
}

func (g *MSSQLGenerator) BuildUpdate(spec UpdateSpec) (string, []any, error) {
	if spec.Table == "" {
		return "", nil, fmt.Errorf("table name is required")
	}
	if len(spec.Values) == 0 {
		return "", nil, fmt.Errorf("no update values provided")
	}

	var sb strings.Builder
	var args []any
	argIdx := 1

	keys := make([]string, 0, len(spec.Values))
	for k := range spec.Values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	setParts := make([]string, len(keys))
	for i, k := range keys {
		setParts[i] = fmt.Sprintf("%s = %s", g.QuoteIdentifier(k), g.Placeholder(argIdx))
		args = append(args, spec.Values[k])
		argIdx++
	}

	sb.WriteString("UPDATE ")
	sb.WriteString(g.QuoteIdentifier(spec.Table))
	sb.WriteString(" SET ")
	sb.WriteString(strings.Join(setParts, ", "))

	if len(spec.Returning) > 0 {
		sb.WriteString(" OUTPUT ")
		if len(spec.Returning) == 1 && spec.Returning[0] == "*" {
			sb.WriteString("INSERTED.*")
		} else {
			outCols := make([]string, len(spec.Returning))
			for i, r := range spec.Returning {
				outCols[i] = "INSERTED." + g.QuoteIdentifier(r)
			}
			sb.WriteString(strings.Join(outCols, ", "))
		}
	}

	if len(spec.Filters) > 0 {
		sb.WriteString(" WHERE ")
		whereParts, whereArgs, _ := g.buildFilters(spec.Filters, argIdx)
		sb.WriteString(strings.Join(whereParts, " AND "))
		args = append(args, whereArgs...)
	}

	return sb.String(), args, nil
}

func (g *MSSQLGenerator) BuildDelete(spec DeleteSpec) (string, []any, error) {
	if spec.Table == "" {
		return "", nil, fmt.Errorf("table name is required")
	}

	var sb strings.Builder
	var args []any

	sb.WriteString("DELETE FROM ")
	sb.WriteString(g.QuoteIdentifier(spec.Table))

	if len(spec.Returning) > 0 {
		sb.WriteString(" OUTPUT ")
		if len(spec.Returning) == 1 && spec.Returning[0] == "*" {
			sb.WriteString("DELETED.*")
		} else {
			outCols := make([]string, len(spec.Returning))
			for i, r := range spec.Returning {
				outCols[i] = "DELETED." + g.QuoteIdentifier(r)
			}
			sb.WriteString(strings.Join(outCols, ", "))
		}
	}

	if len(spec.Filters) > 0 {
		sb.WriteString(" WHERE ")
		whereParts, whereArgs, _ := g.buildFilters(spec.Filters, 1)
		sb.WriteString(strings.Join(whereParts, " AND "))
		args = append(args, whereArgs...)
	}

	return sb.String(), args, nil
}

func (g *MSSQLGenerator) WrapPagination(rawSQL string, limit, offset int) (string, error) {
	trimmed := strings.TrimRight(strings.TrimSpace(rawSQL), ";")
	if limit <= 0 && offset <= 0 {
		return trimmed, nil
	}

	baseQuery, orderClause := splitOuterOrderBy(trimmed)

	var sb strings.Builder
	sb.WriteString("SELECT * FROM (")
	sb.WriteString(baseQuery)
	sb.WriteString(") AS _paginated_sub")

	if orderClause != "" {
		sb.WriteString(" ORDER BY ")
		sb.WriteString(orderClause)
	} else {
		sb.WriteString(" ORDER BY (SELECT NULL)")
	}

	if offset < 0 {
		offset = 0
	}
	sb.WriteString(fmt.Sprintf(" OFFSET %d ROWS", offset))
	if limit > 0 {
		sb.WriteString(fmt.Sprintf(" FETCH NEXT %d ROWS ONLY", limit))
	}
	return sb.String(), nil
}

// splitOuterOrderBy separates an outer trailing ORDER BY clause from the query body to prevent T-SQL Msg 1033
func splitOuterOrderBy(sql string) (string, string) {
	trimmed := strings.TrimRight(strings.TrimSpace(sql), ";")

	depth := 0
	for i := len(trimmed) - 1; i >= 0; i-- {
		switch trimmed[i] {
		case ')':
			depth++
		case '(':
			depth--
		}
		if depth == 0 && i+8 <= len(trimmed) {
			if strings.EqualFold(trimmed[i:i+8], "ORDER BY") {
				beforeValid := i == 0 || trimmed[i-1] == ' ' || trimmed[i-1] == '\t' || trimmed[i-1] == '\n'
				afterValid := i+8 == len(trimmed) || trimmed[i+8] == ' ' || trimmed[i+8] == '\t' || trimmed[i+8] == '\n'
				if beforeValid && afterValid {
					prefix := strings.TrimSpace(trimmed[:i])
					orderClause := strings.TrimSpace(trimmed[i+8:])
					prefixUpper := strings.ToUpper(prefix)
					if !strings.Contains(prefixUpper, " TOP ") && !strings.Contains(prefixUpper, " OFFSET ") {
						return prefix, orderClause
					}
				}
			}
		}
	}
	return trimmed, ""
}

func (g *MSSQLGenerator) buildFilters(filters []FilterCondition, startIdx int) ([]string, []any, int) {
	var parts []string
	var args []any
	idx := startIdx

	for _, f := range filters {
		col := g.QuoteIdentifier(f.Column)
		switch f.Operator {
		case OpIsNull:
			parts = append(parts, fmt.Sprintf("%s IS NULL", col))
		case OpNotNull:
			parts = append(parts, fmt.Sprintf("%s IS NOT NULL", col))
		case OpEq:
			parts = append(parts, fmt.Sprintf("%s = %s", col, g.Placeholder(idx)))
			args = append(args, f.Value)
			idx++
		case OpNeq:
			parts = append(parts, fmt.Sprintf("%s != %s", col, g.Placeholder(idx)))
			args = append(args, f.Value)
			idx++
		case OpGt:
			parts = append(parts, fmt.Sprintf("%s > %s", col, g.Placeholder(idx)))
			args = append(args, f.Value)
			idx++
		case OpGte:
			parts = append(parts, fmt.Sprintf("%s >= %s", col, g.Placeholder(idx)))
			args = append(args, f.Value)
			idx++
		case OpLt:
			parts = append(parts, fmt.Sprintf("%s < %s", col, g.Placeholder(idx)))
			args = append(args, f.Value)
			idx++
		case OpLte:
			parts = append(parts, fmt.Sprintf("%s <= %s", col, g.Placeholder(idx)))
			args = append(args, f.Value)
			idx++
		case OpLike:
			parts = append(parts, fmt.Sprintf("%s LIKE %s", col, g.Placeholder(idx)))
			args = append(args, f.Value)
			idx++
		case OpIlike:
			parts = append(parts, fmt.Sprintf("LOWER(%s) LIKE LOWER(%s)", col, g.Placeholder(idx)))
			args = append(args, f.Value)
			idx++
		case OpIn:
			if slice, ok := ToAnySlice(f.Value); ok && len(slice) > 0 {
				inPlaceholders := make([]string, len(slice))
				for i, val := range slice {
					inPlaceholders[i] = g.Placeholder(idx)
					args = append(args, val)
					idx++
				}
				parts = append(parts, fmt.Sprintf("%s IN (%s)", col, strings.Join(inPlaceholders, ", ")))
			} else {
				parts = append(parts, fmt.Sprintf("%s = %s", col, g.Placeholder(idx)))
				args = append(args, f.Value)
				idx++
			}
		default:
			parts = append(parts, fmt.Sprintf("%s = %s", col, g.Placeholder(idx)))
			args = append(args, f.Value)
			idx++
		}
	}

	return parts, args, idx
}
