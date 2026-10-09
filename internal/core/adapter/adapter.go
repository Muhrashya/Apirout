package adapter

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/apirout/apirout/internal/core/introspection"
	"github.com/apirout/apirout/internal/core/sqldialect"
	"github.com/apirout/apirout/internal/core/targets"
)

// DatabaseAdapter defines the unified seam across connection pooling, connection strings,
// schema introspection, query dialect generation, and identifier escaping.
type DatabaseAdapter interface {
	DBType() string
	DriverName() string
	BuildConnectionString(host string, port int, dbname, username, password, sslmode string) string
	Introspect(ctx context.Context, connString string) (*introspection.SchemaInfo, error)
	GetQueryGenerator() sqldialect.Generator
	QuoteIdentifier(ident string) string
	Open(connString string) (*sql.DB, error)
}

// GetAdapter resolves the registered DatabaseAdapter for a given database type
func GetAdapter(dbType string) (DatabaseAdapter, error) {
	switch strings.ToLower(strings.TrimSpace(dbType)) {
	case "postgres", "postgresql":
		return NewPostgresAdapter(), nil
	case "mysql":
		return NewMySQLAdapter(), nil
	case "mssql", "sqlserver":
		return NewMSSQLAdapter(), nil
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}
}

// SupportedDBTypes returns the list of database types supported by the unified adapter seam
func SupportedDBTypes() []string {
	return []string{"postgres", "mysql", "mssql"}
}

// --- PostgresAdapter ---

type PostgresAdapter struct {
	connector *targets.PostgresConnector
	generator *sqldialect.PostgresGenerator
}

func NewPostgresAdapter() *PostgresAdapter {
	return &PostgresAdapter{
		connector: targets.NewPostgresConnector(),
		generator: sqldialect.NewPostgresGenerator(),
	}
}

func (a *PostgresAdapter) DBType() string {
	return "postgres"
}

func (a *PostgresAdapter) DriverName() string {
	return "postgres"
}

func (a *PostgresAdapter) BuildConnectionString(host string, port int, dbname, username, password, sslmode string) string {
	if sslmode == "" {
		sslmode = "disable"
	}
	return a.connector.BuildConnectionString(host, port, dbname, username, password, sslmode)
}

func (a *PostgresAdapter) Introspect(ctx context.Context, connString string) (*introspection.SchemaInfo, error) {
	return introspection.Introspect(ctx, "postgres", connString)
}

func (a *PostgresAdapter) GetQueryGenerator() sqldialect.Generator {
	return a.generator
}

func (a *PostgresAdapter) QuoteIdentifier(ident string) string {
	return a.generator.QuoteIdentifier(ident)
}

func (a *PostgresAdapter) Open(connString string) (*sql.DB, error) {
	return sql.Open(a.DriverName(), connString)
}

// --- MySQLAdapter ---

type MySQLAdapter struct {
	connector *targets.MySQLConnector
	generator *sqldialect.MySQLGenerator
}

func NewMySQLAdapter() *MySQLAdapter {
	return &MySQLAdapter{
		connector: targets.NewMySQLConnector(),
		generator: sqldialect.NewMySQLGenerator(),
	}
}

func (a *MySQLAdapter) DBType() string {
	return "mysql"
}

func (a *MySQLAdapter) DriverName() string {
	return "mysql"
}

func (a *MySQLAdapter) BuildConnectionString(host string, port int, dbname, username, password, _ string) string {
	return a.connector.BuildConnectionString(host, port, dbname, username, password)
}

func (a *MySQLAdapter) Introspect(ctx context.Context, connString string) (*introspection.SchemaInfo, error) {
	return introspection.Introspect(ctx, "mysql", connString)
}

func (a *MySQLAdapter) GetQueryGenerator() sqldialect.Generator {
	return a.generator
}

func (a *MySQLAdapter) QuoteIdentifier(ident string) string {
	return a.generator.QuoteIdentifier(ident)
}

func (a *MySQLAdapter) Open(connString string) (*sql.DB, error) {
	return sql.Open(a.DriverName(), connString)
}

// --- MSSQLAdapter ---

type MSSQLAdapter struct {
	connector *targets.MSSQLConnector
	generator *sqldialect.MSSQLGenerator
}

func NewMSSQLAdapter() *MSSQLAdapter {
	return &MSSQLAdapter{
		connector: targets.NewMSSQLConnector(),
		generator: sqldialect.NewMSSQLGenerator(),
	}
}

func (a *MSSQLAdapter) DBType() string {
	return "mssql"
}

func (a *MSSQLAdapter) DriverName() string {
	return "sqlserver"
}

func (a *MSSQLAdapter) BuildConnectionString(host string, port int, dbname, username, password, _ string) string {
	return a.connector.BuildConnectionString(host, port, dbname, username, password)
}

func (a *MSSQLAdapter) Introspect(ctx context.Context, connString string) (*introspection.SchemaInfo, error) {
	return introspection.Introspect(ctx, "mssql", connString)
}

func (a *MSSQLAdapter) GetQueryGenerator() sqldialect.Generator {
	return a.generator
}

func (a *MSSQLAdapter) QuoteIdentifier(ident string) string {
	return a.generator.QuoteIdentifier(ident)
}

func (a *MSSQLAdapter) Open(connString string) (*sql.DB, error) {
	return sql.Open(a.DriverName(), connString)
}
