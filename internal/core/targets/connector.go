package targets

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Column represents a single column in a result row
type Column struct {
	Name  string
	Value interface{}
}

// Row represents a single row in a result set, preserving column order in JSON
type Row []Column

// MarshalJSON implements json.Marshaler to ensure column order is strictly preserved
func (r Row) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, col := range r {
		if i > 0 {
			b.WriteByte(',')
		}
		// Marshal column name
		name, err := json.Marshal(col.Name)
		if err != nil {
			return nil, err
		}
		b.Write(name)
		b.WriteByte(':')
		// Marshal column value
		val, err := json.Marshal(col.Value)
		if err != nil {
			return nil, err
		}
		b.Write(val)
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

// Connector is the standard interface for target database execution
type Connector interface {
	TestConnection(connString string) (time.Duration, error)
	Execute(ctx context.Context, connString, query string, params map[string]interface{}, limit int) ([]Row, error)
}

// GetConnector returns the appropriate connector for the target database type
func GetConnector(dbType string) (Connector, error) {
	switch dbType {
	case "postgres", "postgresql":
		return NewPostgresConnector(), nil
	case "mysql":
		return NewMySQLConnector(), nil
	case "mssql", "sqlserver":
		return NewMSSQLConnector(), nil
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}
}

// BuildConnectionString builds a connection string for the given database profile
func BuildConnectionString(dbType, host string, port int, dbname, username, password, sslmode string) (string, error) {
	switch dbType {
	case "postgres", "postgresql":
		conn := NewPostgresConnector()
		return conn.BuildConnectionString(host, port, dbname, username, password, sslmode), nil
	case "mysql":
		conn := NewMySQLConnector()
		return conn.BuildConnectionString(host, port, dbname, username, password), nil
	case "mssql", "sqlserver":
		conn := NewMSSQLConnector()
		return conn.BuildConnectionString(host, port, dbname, username, password), nil
	default:
		return "", fmt.Errorf("unsupported database type: %s", dbType)
	}
}
