package sqldialect

import (
	"fmt"
	"sort"
	"strings"
)

// MySQLGenerator generates MySQL / MariaDB dialect SQL
type MySQLGenerator struct{}

// NewMySQLGenerator creates a new MySQL generator
func NewMySQLGenerator() *MySQLGenerator {
	return &MySQLGenerator{}
}

func (g *MySQLGenerator) Dialect() string {
	return "mysql"
}

func (g *MySQLGenerator) QuoteIdentifier(ident string) string {
	clean := strings.ReplaceAll(ident, "`", "``")
	return "`" + clean + "`"
}

func (g *MySQLGenerator) Placeholder(index int) string {
	return "?"
}

func (g *MySQLGenerator) BuildSelect(spec QuerySpec) (string, []any, error) {
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

func (g *MySQLGenerator) BuildInsert(spec InsertSpec) (string, []any, error) {
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

	sb.WriteString("INSERT INTO ")
	sb.WriteString(g.QuoteIdentifier(spec.Table))
	sb.WriteString(" (")
	sb.WriteString(strings.Join(quotedCols, ", "))
	sb.WriteString(") VALUES (")
	sb.WriteString(strings.Join(placeholders, ", "))
	sb.WriteString(")")

	return sb.String(), args, nil
}

func (g *MySQLGenerator) BuildUpdate(spec UpdateSpec) (string, []any, error) {
	if spec.Table == "" {
		return "", nil, fmt.Errorf("table name is required")
	}
	if len(spec.Values) == 0 {
		return "", nil, fmt.Errorf("no update values provided")
	}

	var sb strings.Builder
	var args []any

	keys := make([]string, 0, len(spec.Values))
	for k := range spec.Values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	setParts := make([]string, len(keys))
	for i, k := range keys {
		setParts[i] = fmt.Sprintf("%s = ?", g.QuoteIdentifier(k))
		args = append(args, spec.Values[k])
	}

	sb.WriteString("UPDATE ")
	sb.WriteString(g.QuoteIdentifier(spec.Table))
	sb.WriteString(" SET ")
	sb.WriteString(strings.Join(setParts, ", "))

	if len(spec.Filters) > 0 {
		sb.WriteString(" WHERE ")
		whereParts, whereArgs := g.buildFilters(spec.Filters)
		sb.WriteString(strings.Join(whereParts, " AND "))
		args = append(args, whereArgs...)
	}

	return sb.String(), args, nil
}

func (g *MySQLGenerator) BuildDelete(spec DeleteSpec) (string, []any, error) {
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

func (g *MySQLGenerator) WrapPagination(rawSQL string, limit, offset int) (string, error) {
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

func (g *MySQLGenerator) buildFilters(filters []FilterCondition) ([]string, []any) {
	var parts []string
	var args []any

	for _, f := range filters {
		col := g.QuoteIdentifier(f.Column)
		switch f.Operator {
		case OpIsNull:
			parts = append(parts, fmt.Sprintf("%s IS NULL", col))
		case OpNotNull:
			parts = append(parts, fmt.Sprintf("%s IS NOT NULL", col))
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
			if slice, ok := toAnySlice(f.Value); ok && len(slice) > 0 {
				inPlaceholders := make([]string, len(slice))
				for i, val := range slice {
					inPlaceholders[i] = "?"
					args = append(args, val)
				}
				parts = append(parts, fmt.Sprintf("%s IN (%s)", col, strings.Join(inPlaceholders, ", ")))
			} else {
				parts = append(parts, fmt.Sprintf("%s = ?", col))
				args = append(args, f.Value)
			}
		default:
			parts = append(parts, fmt.Sprintf("%s = ?", col))
			args = append(args, f.Value)
		}
	}

	return parts, args
}
