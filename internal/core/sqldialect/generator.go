package sqldialect

import (
	"fmt"
	"strings"
)

// Generator builds dialect-specific SQL and binds arguments
type Generator interface {
	Dialect() string
	QuoteIdentifier(ident string) string
	Placeholder(index int) string
	BuildSelect(spec QuerySpec) (string, []any, error)
	BuildInsert(spec InsertSpec) (string, []any, error)
	BuildUpdate(spec UpdateSpec) (string, []any, error)
	BuildDelete(spec DeleteSpec) (string, []any, error)
	WrapPagination(rawSQL string, limit, offset int) (string, error)
}

// GetGenerator returns the appropriate SQL generator for a given dialect
func GetGenerator(dialect string) (Generator, error) {
	norm := strings.ToLower(strings.TrimSpace(dialect))
	switch norm {
	case "postgres", "postgresql", "pg":
		return NewPostgresGenerator(), nil
	case "mysql", "mariadb":
		return NewMySQLGenerator(), nil
	case "mssql", "sqlserver":
		return NewMSSQLGenerator(), nil
	default:
		return nil, fmt.Errorf("unsupported sql dialect: %s", dialect)
	}
}

// WrapPagination wraps any query with limit and offset using the requested dialect
func WrapPagination(rawSQL, dialect string, limit, offset int) (string, error) {
	gen, err := GetGenerator(dialect)
	if err != nil {
		return "", err
	}
	return gen.WrapPagination(rawSQL, limit, offset)
}
