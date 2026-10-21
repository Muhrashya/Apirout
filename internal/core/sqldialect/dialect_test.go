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

func TestToAnySlice(t *testing.T) {
	// 1. []string
	strSlice, ok := ToAnySlice([]string{"a", "b", "c"})
	if !ok || len(strSlice) != 3 || strSlice[0] != "a" {
		t.Errorf("expected []string conversion to succeed, got %v, ok=%v", strSlice, ok)
	}

	// 2. []int
	intSlice, ok := ToAnySlice([]int{1, 2, 3})
	if !ok || len(intSlice) != 3 || intSlice[1] != 2 {
		t.Errorf("expected []int conversion to succeed, got %v, ok=%v", intSlice, ok)
	}

	// 3. []int64
	i64Slice, ok := ToAnySlice([]int64{10, 20})
	if !ok || len(i64Slice) != 2 || i64Slice[0] != int64(10) {
		t.Errorf("expected []int64 conversion to succeed, got %v, ok=%v", i64Slice, ok)
	}

	// 4. []float64
	f64Slice, ok := ToAnySlice([]float64{1.5, 2.5})
	if !ok || len(f64Slice) != 2 || f64Slice[1] != 2.5 {
		t.Errorf("expected []float64 conversion to succeed, got %v, ok=%v", f64Slice, ok)
	}

	// 5. []any
	anySlice, ok := ToAnySlice([]any{"x", 42})
	if !ok || len(anySlice) != 2 {
		t.Errorf("expected []any conversion to succeed, got %v, ok=%v", anySlice, ok)
	}

	// 6. Non-slice
	_, ok = ToAnySlice("not a slice")
	if ok {
		t.Errorf("expected non-slice conversion to fail")
	}
}

func TestMSSQLPaginationWithOrderBy(t *testing.T) {
	gen, err := GetGenerator("sqlserver")
	if err != nil {
		t.Fatalf("unexpected error getting generator: %v", err)
	}

	// Case 1: Query with trailing ORDER BY -> ORDER BY must be lifted outside derived table
	rawWithOrder := "SELECT id, name, created_at FROM users WHERE status = 'active' ORDER BY created_at DESC"
	pagedSQL, err := gen.WrapPagination(rawWithOrder, 10, 20)
	if err != nil {
		t.Fatalf("WrapPagination failed: %v", err)
	}

	// Inner subquery must not contain ORDER BY
	expectedSub := "SELECT * FROM (SELECT id, name, created_at FROM users WHERE status = 'active') AS _paginated_sub"
	if !strings.HasPrefix(pagedSQL, expectedSub) {
		t.Errorf("expected derived subquery without inner ORDER BY, got:\n%s", pagedSQL)
	}
	// Outer query must contain the original ORDER BY
	if !strings.Contains(pagedSQL, "ORDER BY created_at DESC OFFSET 20 ROWS FETCH NEXT 10 ROWS ONLY") {
		t.Errorf("expected outer ORDER BY and OFFSET/FETCH, got:\n%s", pagedSQL)
	}

	// Case 2: Query without ORDER BY -> outer gets ORDER BY (SELECT NULL)
	rawWithoutOrder := "SELECT id, name FROM users WHERE status = 'active'"
	pagedSQL2, err := gen.WrapPagination(rawWithoutOrder, 15, 0)
	if err != nil {
		t.Fatalf("WrapPagination failed: %v", err)
	}
	if !strings.Contains(pagedSQL2, "ORDER BY (SELECT NULL) OFFSET 0 ROWS FETCH NEXT 15 ROWS ONLY") {
		t.Errorf("expected ORDER BY (SELECT NULL), got:\n%s", pagedSQL2)
	}
}

func TestTrinoDialectGenerator(t *testing.T) {
	gen, err := GetGenerator("trino")
	if err != nil {
		t.Fatalf("unexpected error getting trino generator: %v", err)
	}

	if gen.Dialect() != "trino" {
		t.Errorf("expected dialect trino, got %s", gen.Dialect())
	}

	if gen.QuoteIdentifier("orders") != `"orders"` {
		t.Errorf("expected \"orders\", got %s", gen.QuoteIdentifier("orders"))
	}

	// 1. Select with filters and pagination
	spec := QuerySpec{
		Table:   "orders",
		Columns: []string{"id", "amount", "status"},
		Filters: []FilterCondition{
			{Column: "status", Operator: OpEq, Value: "completed"},
			{Column: "amount", Operator: OpGte, Value: 100.50},
		},
		OrderBy: []OrderBy{
			{Column: "id", Direction: OrderAsc},
		},
		Limit:  10,
		Offset: 20,
	}

	sqlStr, args, err := gen.BuildSelect(spec)
	if err != nil {
		t.Fatalf("BuildSelect failed: %v", err)
	}
	if !strings.Contains(sqlStr, `FROM "orders"`) {
		t.Errorf("expected quoted table \"orders\", got: %s", sqlStr)
	}
	if !strings.Contains(sqlStr, `LIMIT 10 OFFSET 20`) {
		t.Errorf("expected LIMIT 10 OFFSET 20, got: %s", sqlStr)
	}
	if len(args) != 2 {
		t.Errorf("expected 2 args, got %d", len(args))
	}

	// 2. Insert
	ins := InsertSpec{
		Table: "orders",
		Values: map[string]any{
			"id":     1,
			"amount": 50,
		},
	}
	insSQL, insArgs, err := gen.BuildInsert(ins)
	if err != nil {
		t.Fatalf("BuildInsert failed: %v", err)
	}
	if !strings.Contains(insSQL, `INSERT INTO "orders"`) {
		t.Errorf("expected INSERT INTO \"orders\", got: %s", insSQL)
	}
	if len(insArgs) != 2 {
		t.Errorf("expected 2 insert args, got %d", len(insArgs))
	}

	// 3. Update (Must return error for analytical engine)
	up := UpdateSpec{
		Table: "orders",
		Values: map[string]any{
			"amount": 60,
		},
		Filters: []FilterCondition{
			{Column: "id", Operator: OpEq, Value: 1},
		},
	}
	_, _, err = gen.BuildUpdate(up)
	if err == nil || !strings.Contains(err.Error(), "does not support row-level UPDATE") {
		t.Errorf("expected row-level update error, got: %v", err)
	}

	// 4. Delete
	del := DeleteSpec{
		Table: "orders",
		Filters: []FilterCondition{
			{Column: "id", Operator: OpEq, Value: 1},
		},
	}
	delSQL, delArgs, err := gen.BuildDelete(del)
	if err != nil {
		t.Fatalf("BuildDelete failed: %v", err)
	}
	if !strings.Contains(delSQL, `DELETE FROM "orders" WHERE "id" = ?`) {
		t.Errorf("expected DELETE FROM \"orders\", got: %s", delSQL)
	}
	if len(delArgs) != 1 {
		t.Errorf("expected 1 delete arg, got %d", len(delArgs))
	}

	// 5. WrapPagination
	paginated, err := gen.WrapPagination("SELECT * FROM orders", 50, 100)
	if err != nil {
		t.Fatalf("WrapPagination failed: %v", err)
	}
	if !strings.Contains(paginated, "LIMIT 50 OFFSET 100") {
		t.Errorf("expected LIMIT 50 OFFSET 100, got: %s", paginated)
	}
}
