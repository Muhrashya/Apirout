package sqldialect

import (
	"fmt"
	"sort"
	"strings"
)

// PostgresGenerator generates PostgreSQL dialect SQL
type PostgresGenerator struct{}

// NewPostgresGenerator creates a new Postgres generator
func NewPostgresGenerator() *PostgresGenerator {
	return &PostgresGenerator{}
}

func (g *PostgresGenerator) Dialect() string {
	return "postgres"
}

func (g *PostgresGenerator) QuoteIdentifier(ident string) string {
	clean := strings.ReplaceAll(ident, `"`, `""`)
	return `"` + clean + `"`
}

func (g *PostgresGenerator) Placeholder(index int) string {
	return fmt.Sprintf("$%d", index)
}

func (g *PostgresGenerator) BuildSelect(spec QuerySpec) (string, []any, error) {
	if spec.Table == "" {
		return "", nil, fmt.Errorf("table name is required")
	}

	var sb strings.Builder
	var args []any
	argIdx := 1

	// SELECT
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
	}

	// LIMIT & OFFSET
	if spec.Limit > 0 {
		sb.WriteString(fmt.Sprintf(" LIMIT %d", spec.Limit))
	}
	if spec.Offset > 0 {
		sb.WriteString(fmt.Sprintf(" OFFSET %d", spec.Offset))
	}

	return sb.String(), args, nil
}

func (g *PostgresGenerator) BuildInsert(spec InsertSpec) (string, []any, error) {
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
	sb.WriteString(") VALUES (")
	sb.WriteString(strings.Join(placeholders, ", "))
	sb.WriteString(")")

	if len(spec.Returning) > 0 {
		sb.WriteString(" RETURNING ")
		if len(spec.Returning) == 1 && spec.Returning[0] == "*" {
			sb.WriteString("*")
		} else {
			ret := make([]string, len(spec.Returning))
			for i, r := range spec.Returning {
				ret[i] = g.QuoteIdentifier(r)
			}
			sb.WriteString(strings.Join(ret, ", "))
		}
	}

	return sb.String(), args, nil
}

func (g *PostgresGenerator) BuildUpdate(spec UpdateSpec) (string, []any, error) {
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

	if len(spec.Filters) > 0 {
		sb.WriteString(" WHERE ")
		whereParts, whereArgs, _ := g.buildFilters(spec.Filters, argIdx)
		sb.WriteString(strings.Join(whereParts, " AND "))
		args = append(args, whereArgs...)
	}

	if len(spec.Returning) > 0 {
		sb.WriteString(" RETURNING ")
		if len(spec.Returning) == 1 && spec.Returning[0] == "*" {
			sb.WriteString("*")
		} else {
			ret := make([]string, len(spec.Returning))
			for i, r := range spec.Returning {
				ret[i] = g.QuoteIdentifier(r)
			}
			sb.WriteString(strings.Join(ret, ", "))
		}
	}

	return sb.String(), args, nil
}

func (g *PostgresGenerator) BuildDelete(spec DeleteSpec) (string, []any, error) {
	if spec.Table == "" {
		return "", nil, fmt.Errorf("table name is required")
	}

	var sb strings.Builder
	var args []any

	sb.WriteString("DELETE FROM ")
	sb.WriteString(g.QuoteIdentifier(spec.Table))

	if len(spec.Filters) > 0 {
		sb.WriteString(" WHERE ")
		whereParts, whereArgs, _ := g.buildFilters(spec.Filters, 1)
		sb.WriteString(strings.Join(whereParts, " AND "))
		args = append(args, whereArgs...)
	}

	return sb.String(), args, nil
}

func (g *PostgresGenerator) WrapPagination(rawSQL string, limit, offset int) (string, error) {
	trimmed := strings.TrimRight(strings.TrimSpace(rawSQL), ";")
	if limit <= 0 && offset <= 0 {
		return trimmed, nil
	}
	var sb strings.Builder
	sb.WriteString("SELECT * FROM (")
	sb.WriteString(trimmed)
	sb.WriteString(") AS _paginated_sub")
	if limit > 0 {
		sb.WriteString(fmt.Sprintf(" LIMIT %d", limit))
	}
	if offset > 0 {
		sb.WriteString(fmt.Sprintf(" OFFSET %d", offset))
	}
	return sb.String(), nil
}

func (g *PostgresGenerator) buildFilters(filters []FilterCondition, startIdx int) ([]string, []any, int) {
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
			parts = append(parts, fmt.Sprintf("%s ILIKE %s", col, g.Placeholder(idx)))
			args = append(args, f.Value)
			idx++
		case OpIn:
			if slice, ok := toAnySlice(f.Value); ok && len(slice) > 0 {
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

func toAnySlice(val any) ([]any, bool) {
	switch v := val.(type) {
	case []any:
		return v, true
	case []string:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = item
		}
		return out, true
	case []int:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = item
		}
		return out, true
	default:
		return nil, false
	}
}
