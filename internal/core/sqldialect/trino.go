package sqldialect

import (
	"fmt"
	"sort"
	"strings"
)

// TrinoGenerator generates ANSI SQL dialect for the Trino query engine
type TrinoGenerator struct{}

// NewTrinoGenerator creates a new Trino generator
func NewTrinoGenerator() *TrinoGenerator {
	return &TrinoGenerator{}
}

func (g *TrinoGenerator) Dialect() string {
	return "trino"
}

func (g *TrinoGenerator) QuoteIdentifier(ident string) string {
	// Trino uses standard ANSI double quotes for identifiers
	clean := strings.ReplaceAll(ident, `"`, `""`)
	return `"` + clean + `"`
}

func (g *TrinoGenerator) Placeholder(index int) string {
	return "?"
}

func (g *TrinoGenerator) BuildSelect(spec QuerySpec) (string, []any, error) {
	if spec.Table == "" {
		return "", nil, fmt.Errorf("table name is required")
	}

	var sb strings.Builder
	var args []any

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
		whereParts, whereArgs := g.buildFilters(spec.Filters)
		sb.WriteString(strings.Join(whereParts, " AND "))
		args = append(args, whereArgs...)
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

func (g *TrinoGenerator) BuildInsert(spec InsertSpec) (string, []any, error) {
	if spec.Table == "" {
		return "", nil, fmt.Errorf("table name is required")
	}
	if len(spec.Values) == 0 {
		return "", nil, fmt.Errorf("no insert values provided")
	}

	var sb strings.Builder
	var args []any

	keys := make([]string, 0, len(spec.Values))
	for k := range spec.Values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	quotedCols := make([]string, len(keys))
	placeholders := make([]string, len(keys))
	for i, k := range keys {
		quotedCols[i] = g.QuoteIdentifier(k)
		placeholders[i] = "?"
		args = append(args, spec.Values[k])
	}

	sb.WriteString(fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		g.QuoteIdentifier(spec.Table),
		strings.Join(quotedCols, ", "),
		strings.Join(placeholders, ", "),
	))

	return sb.String(), args, nil
}

func (g *TrinoGenerator) BuildUpdate(spec UpdateSpec) (string, []any, error) {
	// Trino is an analytical OLAP query engine; row-level updates are not supported across most connectors
	return "", nil, fmt.Errorf("trino does not support row-level UPDATE operations (analytical lakehouse target)")
}

func (g *TrinoGenerator) BuildDelete(spec DeleteSpec) (string, []any, error) {
	if spec.Table == "" {
		return "", nil, fmt.Errorf("table name is required")
	}

	var sb strings.Builder
	var args []any

	sb.WriteString("DELETE FROM ")
	sb.WriteString(g.QuoteIdentifier(spec.Table))

	if len(spec.Filters) > 0 {
		sb.WriteString(" WHERE ")
		whereParts, whereArgs := g.buildFilters(spec.Filters)
		sb.WriteString(strings.Join(whereParts, " AND "))
		args = append(args, whereArgs...)
	}

	return sb.String(), args, nil
}

func (g *TrinoGenerator) WrapPagination(rawSQL string, limit, offset int) (string, error) {
	cleanSQL := strings.TrimRight(strings.TrimSpace(rawSQL), ";")
	if limit > 0 && offset > 0 {
		return fmt.Sprintf("%s LIMIT %d OFFSET %d", cleanSQL, limit, offset), nil
	}
	if limit > 0 {
		return fmt.Sprintf("%s LIMIT %d", cleanSQL, limit), nil
	}
	if offset > 0 {
		return fmt.Sprintf("%s OFFSET %d", cleanSQL, offset), nil
	}
	return cleanSQL, nil
}

func (g *TrinoGenerator) buildFilters(filters []FilterCondition) ([]string, []any) {
	var parts []string
	var args []any

	for _, f := range filters {
		col := g.QuoteIdentifier(f.Column)
		switch f.Operator {
		case OpEq:
			parts = append(parts, fmt.Sprintf("%s = ?", col))
			args = append(args, f.Value)
		case OpNeq:
			parts = append(parts, fmt.Sprintf("%s != ?", col))
			args = append(args, f.Value)
		case OpGt:
			parts = append(parts, fmt.Sprintf("%s > ?", col))
			args = append(args, f.Value)
		case OpGte:
			parts = append(parts, fmt.Sprintf("%s >= ?", col))
			args = append(args, f.Value)
		case OpLt:
			parts = append(parts, fmt.Sprintf("%s < ?", col))
			args = append(args, f.Value)
		case OpLte:
			parts = append(parts, fmt.Sprintf("%s <= ?", col))
			args = append(args, f.Value)
		case OpLike:
			parts = append(parts, fmt.Sprintf("%s LIKE ?", col))
			args = append(args, f.Value)
		case OpIlike:
			parts = append(parts, fmt.Sprintf("LOWER(%s) LIKE LOWER(?)", col))
			args = append(args, f.Value)
		case OpIn:
			slice, ok := ToAnySlice(f.Value)
			if ok && len(slice) > 0 {
				ph := make([]string, len(slice))
				for i := range slice {
					ph[i] = "?"
					args = append(args, slice[i])
				}
				parts = append(parts, fmt.Sprintf("%s IN (%s)", col, strings.Join(ph, ", ")))
			}
		case OpIsNull:
			parts = append(parts, fmt.Sprintf("%s IS NULL", col))
		case OpNotNull:
			parts = append(parts, fmt.Sprintf("%s IS NOT NULL", col))
		}
	}

	return parts, args
}
