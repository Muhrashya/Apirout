package targets

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/microsoft/go-mssqldb"
)

// MSSQLConnector handles Microsoft SQL Server target database operations
type MSSQLConnector struct{}

// NewMSSQLConnector creates a new MSSQL connector
func NewMSSQLConnector() *MSSQLConnector {
	return &MSSQLConnector{}
}

// BuildConnectionString builds a Microsoft SQL Server connection string
func (c *MSSQLConnector) BuildConnectionString(host string, port int, dbname, username, password string) string {
	encodedPassword := url.QueryEscape(password)
	if port <= 0 {
		port = 1433
	}

	return fmt.Sprintf(
		"sqlserver://%s:%s@%s:%d?database=%s&connection+timeout=5&TrustServerCertificate=true",
		username, encodedPassword, host, port, dbname,
	)
}

// TestConnection tests an MSSQL connection and returns latency
func (c *MSSQLConnector) TestConnection(connString string) (time.Duration, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()

	db, err := sql.Open("sqlserver", connString)
	if err != nil {
		return 0, fmt.Errorf("failed to open connection: %w", err)
	}
	defer db.Close()

	db.SetConnMaxLifetime(5 * time.Second)
	db.SetMaxOpenConns(1)

	var result int
	err = db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		return 0, fmt.Errorf("connection test failed: %w", err)
	}

	latency := time.Since(start)
	return latency, nil
}

// Execute runs a query on an MSSQL connection
func (c *MSSQLConnector) Execute(ctx context.Context, connString, query string, params map[string]interface{}, limit int) ([]Row, error) {
	db, err := sqlx.Open("sqlserver", connString)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}
	defer db.Close()

	db.SetConnMaxLifetime(10 * time.Second)
	db.SetMaxOpenConns(1)

	rows, err := db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	var results []Row
	count := 0

	for rows.Next() && count < limit {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		row := make(Row, 0, len(columns))
		for i, col := range columns {
			val := values[i]
			var finalVal interface{}
			if b, ok := val.([]byte); ok {
				finalVal = string(b)
			} else {
				finalVal = val
			}
			row = append(row, Column{Name: col, Value: finalVal})
		}
		results = append(results, row)
		count++
	}

	return results, rows.Err()
}
