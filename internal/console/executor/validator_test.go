package executor

import (
	"testing"
)

func TestValidateSQL(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		wantErr bool
	}{
		// Allowed cases
		{"Simple Select", "SELECT * FROM users", false},
		{"Select with Where", "SELECT id FROM users WHERE id = 1", false},
		{"CTE", "WITH t AS (SELECT 1) SELECT * FROM t", false},
		{"Case Insensitive", "select * from users", false},
		{"Whitespace", "  SELECT   *   FROM   users  ", false},

		// Forbidden Keywords
		{"Insert", "INSERT INTO users VALUES (1)", true},
		{"Update", "UPDATE users SET name = 'foo'", true},
		{"Delete", "DELETE FROM users", true},
		{"Drop", "DROP TABLE users", true},
		{"Truncate", "TRUNCATE TABLE users", true},
		{"Alter", "ALTER TABLE users ADD COLUMN foo int", true},
		{"Grant", "GRANT ALL ON users TO public", true},
		{"Copy", "COPY users TO stdout", true},

		// Multi-statement
		{"Semicolon Chain", "SELECT 1; SELECT 2", true},
		{"Semicolon Drop", "SELECT 1; DROP TABLE users", true},
		{"Trailing Semicolon Allowed", "SELECT 1;", false},

		// Dangerous Functions (Hardened)
		{"PG Read File", "SELECT pg_read_file('/etc/passwd')", true},
		{"PG Read Binary File", "SELECT pg_read_binary_file('/etc/shadow')", true},
		{"PG Sleep", "SELECT pg_sleep(10)", true},
		{"MySQL Sleep", "SELECT * FROM users WHERE id = 1 AND SLEEP(5)", true},
		{"MySQL Benchmark", "SELECT BENCHMARK(1000000, MD5(1))", true},
		{"MySQL Into Outfile", "SELECT * FROM users INTO OUTFILE '/tmp/dump.txt'", true},
		{"MySQL Into Dumpfile", "SELECT * FROM users INTO DUMPFILE '/tmp/dump.bin'", true},
		{"MySQL Load File", "SELECT LOAD_FILE('/etc/passwd')", true},
		{"Postgres Lo Export", "SELECT lo_export(1337, '/tmp/evil')", true},
		{"Postgres Lo Import", "SELECT lo_import('/tmp/evil')", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSQL(tt.sql)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSQL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
