package registry

import (
	"fmt"
	"strings"
)

// fixPostgresBooleanColumns fixes columns that should be BOOLEAN but might be INTEGER
func (r *Registry) fixPostgresBooleanColumns() error {
	// List of tables and columns that should be boolean
	columnsToCheck := []struct {
		table  string
		column string
	}{
		{"endpoints", "rate_limit_enabled"},
		{"endpoints", "return_total_count"},
	}

	for _, c := range columnsToCheck {
		// Check current data type
		var dataType string
		err := r.db.Get(&dataType, fmt.Sprintf("SELECT data_type FROM information_schema.columns WHERE table_name='%s' AND column_name='%s'", c.table, c.column))
		
		// If column doesn't exist, ignore (will be added by ensureColumnExists)
		if err != nil {
			continue
		}

		// If it's integer, convert it
		if strings.ToLower(dataType) == "integer" {
			fmt.Printf("Migrating column %s.%s from INTEGER to BOOLEAN...\n", c.table, c.column)
			
			// Use USING clause to convert 0/1 to false/true
			query := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE BOOLEAN USING (%s::boolean)", c.table, c.column, c.column)
			if _, err := r.db.Exec(query); err != nil {
				return fmt.Errorf("failed to convert column %s.%s to boolean: %w", c.table, c.column, err)
			}
			
			// Restore default value
			defaultVal := "TRUE"
			if c.column == "return_total_count" {
				defaultVal = "FALSE"
			}
			query = fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET DEFAULT %s", c.table, c.column, defaultVal)
			if _, err := r.db.Exec(query); err != nil {
				return fmt.Errorf("failed to set default for column %s.%s: %w", c.table, c.column, err)
			}
		}
	}

	return nil
}
