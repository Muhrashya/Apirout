package introspection

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

// ColumnInfo holds metadata for a single database column
type ColumnInfo struct {
	Name         string `json:"name"`
	DataType     string `json:"data_type"`
	IsNullable   bool   `json:"is_nullable"`
	DefaultValue string `json:"default_value,omitempty"`
	IsPrimaryKey bool   `json:"is_primary_key"`
}

// TableInfo holds metadata for an introspected table or view
type TableInfo struct {
	Name       string       `json:"name"`
	Type       string       `json:"type"` // "BASE TABLE" or "VIEW"
	PrimaryKey string       `json:"primary_key,omitempty"`
	Columns    []ColumnInfo `json:"columns"`
	RowCount   int64        `json:"row_count,omitempty"`
}

// SchemaInfo represents the complete schema of a target database
type SchemaInfo struct {
	Tables    []TableInfo `json:"tables"`
	Timestamp int64       `json:"timestamp"`
}

// Introspect connects to target database and extracts complete schema metadata
func Introspect(ctx context.Context, dbType, connString string) (*SchemaInfo, error) {
	switch dbType {
	case "postgres", "postgresql":
		return introspectPostgres(ctx, connString)
	case "mysql":
		return introspectMySQL(ctx, connString)
	default:
		return nil, fmt.Errorf("unsupported database type for introspection: %s", dbType)
	}
}

func introspectPostgres(ctx context.Context, connString string) (*SchemaInfo, error) {
	db, err := sql.Open("postgres", connString)
	if err != nil {
		return nil, fmt.Errorf("open postgres connection: %w", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// 1. Fetch tables
	tableQuery := `
		SELECT table_name, table_type
		FROM information_schema.tables
		WHERE table_schema = 'public'
		  AND table_type IN ('BASE TABLE', 'VIEW')
		ORDER BY table_name;
	`
	rows, err := db.QueryContext(ctx, tableQuery)
	if err != nil {
		return nil, fmt.Errorf("query tables: %w", err)
	}
	defer rows.Close()

	var tables []TableInfo
	for rows.Next() {
		var t TableInfo
		if err := rows.Scan(&t.Name, &t.Type); err != nil {
			return nil, err
		}
		tables = append(tables, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// 2. Fetch primary keys
	pkQuery := `
		SELECT kcu.table_name, kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
		  ON tc.constraint_name = kcu.constraint_name
		  AND tc.table_schema = kcu.table_schema
		WHERE tc.constraint_type = 'PRIMARY KEY'
		  AND tc.table_schema = 'public';
	`
	pkRows, err := db.QueryContext(ctx, pkQuery)
	pkMap := make(map[string]string) // table -> pk_col
	if err == nil {
		defer pkRows.Close()
		for pkRows.Next() {
			var tbl, col string
			if err := pkRows.Scan(&tbl, &col); err == nil {
				pkMap[tbl] = col
			}
		}
	}

	// 3. Fetch columns for each table
	colQuery := `
		SELECT table_name, column_name, data_type, is_nullable, COALESCE(column_default, '')
		FROM information_schema.columns
		WHERE table_schema = 'public'
		ORDER BY table_name, ordinal_position;
	`
	colRows, err := db.QueryContext(ctx, colQuery)
	if err != nil {
		return nil, fmt.Errorf("query columns: %w", err)
	}
	defer colRows.Close()

	colsByTable := make(map[string][]ColumnInfo)
	for colRows.Next() {
		var tbl, col, dt, nullable, defVal string
		if err := colRows.Scan(&tbl, &col, &dt, &nullable, &defVal); err != nil {
			return nil, err
		}
		isPK := pkMap[tbl] == col
		colsByTable[tbl] = append(colsByTable[tbl], ColumnInfo{
			Name:         col,
			DataType:     dt,
			IsNullable:   nullable == "YES",
			DefaultValue: defVal,
			IsPrimaryKey: isPK,
		})
	}

	for i := range tables {
		tables[i].PrimaryKey = pkMap[tables[i].Name]
		if tables[i].PrimaryKey == "" && len(colsByTable[tables[i].Name]) > 0 {
			// Fallback: look for common ID column
			for _, col := range colsByTable[tables[i].Name] {
				if col.Name == "id" {
					tables[i].PrimaryKey = "id"
					break
				}
			}
		}
		tables[i].Columns = colsByTable[tables[i].Name]
	}

	return &SchemaInfo{
		Tables:    tables,
		Timestamp: time.Now().Unix(),
	}, nil
}

func introspectMySQL(ctx context.Context, connString string) (*SchemaInfo, error) {
	db, err := sql.Open("mysql", connString)
	if err != nil {
		return nil, fmt.Errorf("open mysql connection: %w", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	tableQuery := `
		SELECT TABLE_NAME, TABLE_TYPE
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = DATABASE()
		ORDER BY TABLE_NAME;
	`
	rows, err := db.QueryContext(ctx, tableQuery)
	if err != nil {
		return nil, fmt.Errorf("query mysql tables: %w", err)
	}
	defer rows.Close()

	var tables []TableInfo
	for rows.Next() {
		var t TableInfo
		if err := rows.Scan(&t.Name, &t.Type); err != nil {
			return nil, err
		}
		tables = append(tables, t)
	}

	colQuery := `
		SELECT TABLE_NAME, COLUMN_NAME, DATA_TYPE, IS_NULLABLE, COALESCE(COLUMN_DEFAULT, ''), COLUMN_KEY
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE()
		ORDER BY TABLE_NAME, ORDINAL_POSITION;
	`
	colRows, err := db.QueryContext(ctx, colQuery)
	if err != nil {
		return nil, fmt.Errorf("query mysql columns: %w", err)
	}
	defer colRows.Close()

	colsByTable := make(map[string][]ColumnInfo)
	pkMap := make(map[string]string)
	for colRows.Next() {
		var tbl, col, dt, nullable, defVal, colKey string
		if err := colRows.Scan(&tbl, &col, &dt, &nullable, &defVal, &colKey); err != nil {
			return nil, err
		}
		isPK := colKey == "PRI"
		if isPK {
			pkMap[tbl] = col
		}
		colsByTable[tbl] = append(colsByTable[tbl], ColumnInfo{
			Name:         col,
			DataType:     dt,
			IsNullable:   nullable == "YES",
			DefaultValue: defVal,
			IsPrimaryKey: isPK,
		})
	}

	for i := range tables {
		tables[i].PrimaryKey = pkMap[tables[i].Name]
		tables[i].Columns = colsByTable[tables[i].Name]
	}

	return &SchemaInfo{
		Tables:    tables,
		Timestamp: time.Now().Unix(),
	}, nil
}
