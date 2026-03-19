package query

type Node interface {
	compile(*compiler) (string, []any)
}

func And(nodes ...Node) *Group {
	return &Group{Operator: "AND", Nodes: nodes}
}

func Or(nodes ...Node) *Group {
	return &Group{Operator: "OR", Nodes: nodes}
}

func Eq(field string, value any) Node  { return &Condition{field, "=", value} }
func Gt(field string, value any) Node  { return &Condition{field, ">", value} }
func Lt(field string, value any) Node  { return &Condition{field, "<", value} }
func Gte(field string, value any) Node { return &Condition{field, ">=", value} }
func Lte(field string, value any) Node { return &Condition{field, "<=", value} }
func Like(field string, value any) Node {
	return &Condition{field, "LIKE", value}
}
func IsNull(field string) Node  { return &Condition{field, "IS NULL", nil} }
func NotNull(field string) Node { return &Condition{field, "IS NOT NULL", nil} }
func In(field string, value any) Node {
	return &Condition{field, "IN", value}
}
func Exists(sub *SubQuery) Node {
	return &Condition{"", "EXISTS", sub}
}
