package sqldialect

// Operator represents a filter comparison operator
type Operator string

const (
	OpEq      Operator = "eq"
	OpNeq     Operator = "neq"
	OpGt      Operator = "gt"
	OpGte     Operator = "gte"
	OpLt      Operator = "lt"
	OpLte     Operator = "lte"
	OpLike    Operator = "like"
	OpIlike   Operator = "ilike"
	OpIn      Operator = "in"
	OpIsNull  Operator = "is_null"
	OpNotNull Operator = "not_null"
)

// OrderDirection represents sorting direction
type OrderDirection string

const (
	OrderAsc  OrderDirection = "ASC"
	OrderDesc OrderDirection = "DESC"
)

// OrderBy represents a column ordering specification
type OrderBy struct {
	Column    string
	Direction OrderDirection
}

// FilterCondition represents a single WHERE predicate
type FilterCondition struct {
	Column   string
	Operator Operator
	Value    any
}

// QuerySpec represents a SELECT query specification
type QuerySpec struct {
	Table   string
	Columns []string
	Filters []FilterCondition
	OrderBy []OrderBy
	Limit   int
	Offset  int
}

// InsertSpec represents an INSERT query specification
type InsertSpec struct {
	Table     string
	Values    map[string]any
	Returning []string
}

// UpdateSpec represents an UPDATE query specification
type UpdateSpec struct {
	Table     string
	Values    map[string]any
	Filters   []FilterCondition
	Returning []string
}

// DeleteSpec represents a DELETE query specification
type DeleteSpec struct {
	Table     string
	Filters   []FilterCondition
	Returning []string
}
