package query

// Node is a WHERE/HAVING condition tree element - a single Condition, or a
// Group combining several Nodes with AND/OR. Pass one to
// IQueryBuilder.Where/AndWhere/Or/Having.
type Node interface {
	compile(*compiler) (string, []any)
}

// And groups nodes with SQL AND: (n1 AND n2 AND ...).
func And(nodes ...Node) *Group {
	return &Group{Operator: "AND", Nodes: nodes}
}

// Or groups nodes with SQL OR: (n1 OR n2 OR ...).
func Or(nodes ...Node) *Group {
	return &Group{Operator: "OR", Nodes: nodes}
}

// Eq builds "field = value". field may reference a related model
// ("Relation.Field", auto-joined) or a JSONB path ("column->key", kept as-is).
func Eq(field string, value any) Node { return &Condition{field, "=", value} }

// Gt builds "field > value".
func Gt(field string, value any) Node { return &Condition{field, ">", value} }

// Lt builds "field < value".
func Lt(field string, value any) Node { return &Condition{field, "<", value} }

// Gte builds "field >= value".
func Gte(field string, value any) Node { return &Condition{field, ">=", value} }

// Lte builds "field <= value".
func Lte(field string, value any) Node { return &Condition{field, "<=", value} }

// Like builds "field LIKE value" - value is used as-is, wildcards ("%") are
// not added automatically.
func Like(field string, value any) Node {
	return &Condition{field, "LIKE", value}
}

// IsNull builds "field IS NULL".
func IsNull(field string) Node { return &Condition{field, "IS NULL", nil} }

// NotNull builds "field IS NOT NULL".
func NotNull(field string) Node { return &Condition{field, "IS NOT NULL", nil} }

// In builds "field IN value" - value is a slice, or a *SubQuery.
func In(field string, value any) Node {
	return &Condition{field, "IN", value}
}

// Exists builds "EXISTS (sub)".
func Exists(sub *SubQuery) Node {
	return &Condition{"", "EXISTS", sub}
}
