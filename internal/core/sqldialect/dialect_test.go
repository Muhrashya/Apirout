package sqldialect

import (
	"strings"
	"testing"
)

func TestDialectGenerators(t *testing.T) {
	dialects := []string{"postgres", "mysql", "sqlserver"}

	for _, d := range dialects {
		t.Run(d, func(t *testing.T) {
			gen, err := GetGenerator(d)
			if err != nil {
				t.Fatalf("unexpected error getting generator: %v", err)
			}

			// 1. Test Select
			spec := QuerySpec{
				Table:   "users",
				Columns: []string{"id", "email", "status"},
				Filters: []FilterCondition{
					{Column: "status", Operator: OpEq, Value: "active"},
					{Column: "age", Operator: OpGte, Value: 18},
					{Column: "role", Operator: OpIn, Value: []string{"admin", "member"}},
				},
				OrderBy: []OrderBy{
					{Column: "created_at", Direction: OrderDesc},
				},
				Limit:  25,
				Offset: 50,
			}

			sqlStr, args, err := gen.BuildSelect(spec)
			if err != nil {
				t.Fatalf("BuildSelect failed: %v", err)
			}

			if !strings.Contains(sqlStr, "users") {
				t.Errorf("expected table 'users' in sql: %s", sqlStr)
			}
			if len(args) != 4 {
				t.Errorf("expected 4 arguments, got %d: %v", len(args), args)
			}

			switch d {
			case "postgres":
				if !strings.Contains(sqlStr, `"users"`) {
					t.Errorf("expected quoted table \"users\": %s", sqlStr)
				}
				if !strings.Contains(sqlStr, "$1") || !strings.Contains(sqlStr, "$2") {
					t.Errorf("expected $1, $2 placeholders: %s", sqlStr)
				}
				if !strings.Contains(sqlStr, "LIMIT 25 OFFSET 50") {
					t.Errorf("expected LIMIT/OFFSET: %s", sqlStr)
				}
			case "mysql":
				if !strings.Contains(sqlStr, "`users`") {
					t.Errorf("expected quoted table `users`: %s", sqlStr)
				}
				if !strings.Contains(sqlStr, "?") {
					t.Errorf("expected ? placeholders: %s", sqlStr)
				}
				if !strings.Contains(sqlStr, "LIMIT 25 OFFSET 50") {
					t.Errorf("expected LIMIT/OFFSET: %s", sqlStr)
				}
			case "sqlserver":
				if !strings.Contains(sqlStr, "[users]") {
					t.Errorf("expected quoted table [users]: %s", sqlStr)
				}
				if !strings.Contains(sqlStr, "@p1") || !strings.Contains(sqlStr, "@p2") {
					t.Errorf("expected @p1, @p2 placeholders: %s", sqlStr)
				}
				if !strings.Contains(sqlStr, "OFFSET 50 ROWS FETCH NEXT 25 ROWS ONLY") {
					t.Errorf("expected MSSQL offset fetch: %s", sqlStr)
				}
			}

			// 2. Test Insert
			ins := InsertSpec{
				Table: "customers",
				Values: map[string]any{
					"name":  "Alice",
					"email": "alice@example.com",
				},
				Returning: []string{"id"},
			}
			insSQL, insArgs, err := gen.BuildInsert(ins)
			if err != nil {
				t.Fatalf("BuildInsert failed: %v", err)
			}
			if len(insArgs) != 2 {
				t.Errorf("expected 2 insert args, got %d", len(insArgs))
			}
			if d == "postgres" && !strings.Contains(insSQL, `RETURNING "id"`) {
				t.Errorf("expected RETURNING \"id\" for postgres: %s", insSQL)
			}
			if d == "sqlserver" && !strings.Contains(insSQL, `OUTPUT INSERTED.[id]`) {
				t.Errorf("expected OUTPUT INSERTED.[id] for mssql: %s", insSQL)
			}

			// 3. Test Update
			up := UpdateSpec{
				Table: "customers",
				Values: map[string]any{
					"name": "Alice Bob",
				},
				Filters: []FilterCondition{
					{Column: "id", Operator: OpEq, Value: 101},
				},
			}
			upSQL, upArgs, err := gen.BuildUpdate(up)
			if err != nil {
				t.Fatalf("BuildUpdate failed: %v", err)
			}
			if len(upArgs) != 2 {
				t.Errorf("expected 2 update args, got %d", len(upArgs))
			}
			if !strings.Contains(upSQL, "WHERE") {
				t.Errorf("expected WHERE clause in update: %s", upSQL)
			}

			// 4. Test Delete
			del := DeleteSpec{
				Table: "customers",
				Filters: []FilterCondition{
					{Column: "id", Operator: OpEq, Value: 101},
				},
			}
			delSQL, delArgs, err := gen.BuildDelete(del)
			if err != nil {
				t.Fatalf("BuildDelete failed: %v", err)
			}
			if len(delArgs) != 1 {
				t.Errorf("expected 1 delete arg, got %d", len(delArgs))
			}
			if !strings.Contains(delSQL, "DELETE FROM") {
				t.Errorf("expected DELETE FROM in delete: %s", delSQL)
			}

			// 5. Test WrapPagination
			pagedSQL, err := gen.WrapPagination("SELECT * FROM orders WHERE total > 100", 10, 20)
			if err != nil {
				t.Fatalf("WrapPagination failed: %v", err)
			}
			if !strings.Contains(pagedSQL, "_paginated_sub") {
				t.Errorf("expected _paginated_sub alias: %s", pagedSQL)
			}
		})
	}
}

func TestFilterOperators(t *testing.T) {
	gen, _ := GetGenerator("postgres")
	spec := QuerySpec{
		Table: "items",
		Filters: []FilterCondition{
			{Column: "deleted_at", Operator: OpIsNull},
			{Column: "activated_at", Operator: OpNotNull},
			{Column: "title", Operator: OpIlike, Value: "%test%"},
		},
	}
	sqlStr, args, err := gen.BuildSelect(spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sqlStr, `"deleted_at" IS NULL`) {
		t.Errorf("expected IS NULL: %s", sqlStr)
	}
	if !strings.Contains(sqlStr, `"activated_at" IS NOT NULL`) {
		t.Errorf("expected IS NOT NULL: %s", sqlStr)
	}
	if !strings.Contains(sqlStr, `"title" ILIKE $1`) {
		t.Errorf("expected ILIKE $1: %s", sqlStr)
	}
	if len(args) != 1 || args[0] != "%test%" {
		t.Errorf("expected single arg '%%test%%', got: %v", args)
	}
}
