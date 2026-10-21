package targets

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/trinodb/trino-go-client/trino"
)

// TrinoConnector handles Trino distributed SQL query engine operations
type TrinoConnector struct{}

// NewTrinoConnector creates a new Trino connector
func NewTrinoConnector() *TrinoConnector {
	return &TrinoConnector{}
}

// BuildConnectionString builds a Trino connection string URL (DSN)
// Format: http[s]://[user[:password]@]host:port?catalog=...&schema=...
func (c *TrinoConnector) BuildConnectionString(host string, port int, dbname, username, password, sslmode string) string {
	scheme := "http"
	if sslmode == "require" || sslmode == "true" || sslmode == "verify-full" || sslmode == "verify-ca" {
		scheme = "https"
	}

	if port <= 0 {
		if scheme == "https" {
			port = 8443
		} else {
			port = 8080
		}
	}

	if username == "" {
		username = "trino"
	}

	catalog := "system"
	schema := "default"
	if dbname != "" {
		parts := strings.SplitN(dbname, "/", 2)
		catalog = strings.TrimSpace(parts[0])
		if len(parts) > 1 && strings.TrimSpace(parts[1]) != "" {
			schema = strings.TrimSpace(parts[1])
		}
	}

	var userAuth string
	if password != "" {
		userAuth = fmt.Sprintf("%s:%s@", url.QueryEscape(username), url.QueryEscape(password))
	} else {
		userAuth = fmt.Sprintf("%s@", url.QueryEscape(username))
	}

	params := url.Values{}
	params.Set("catalog", catalog)
	params.Set("schema", schema)

	return fmt.Sprintf("%s://%s%s:%d?%s", scheme, userAuth, host, port, params.Encode())
}

// TestConnection tests a Trino connection with a lightweight SELECT 1 query
func (c *TrinoConnector) TestConnection(connString string) (time.Duration, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()

	db, err := sql.Open("trino", connString)
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

// Execute runs a query on a Trino connection
func (c *TrinoConnector) Execute(ctx context.Context, connString, query string, params map[string]interface{}, limit int) ([]Row, error) {
	db, err := sqlx.Open("trino", connString)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}
	defer db.Close()

	db.SetConnMaxLifetime(10 * time.Second)
	db.SetMaxOpenConns(1)

	cleanQuery := strings.TrimRight(strings.TrimSpace(query), ";")
	rows, err := db.NamedQueryContext(ctx, cleanQuery, params)
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
			row = append(row, Column{
				Name:  col,
				Value: values[i],
			})
		}
		results = append(results, row)
		count++
	}

	return results, nil
}
