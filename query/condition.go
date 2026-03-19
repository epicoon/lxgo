package query

import "fmt"

type Condition struct {
	Field    string
	Operator string
	Value    any
}

func (cnd *Condition) compile(c *compiler) (string, []any) {

	if cnd.Operator == "EXISTS" {
		sub := cnd.Value.(*SubQuery)
		sql, args := sub.compile(c)
		return "EXISTS " + sql, args
	}

	col := c.column(cnd.Field)

	switch cnd.Operator {

	case "IS NULL", "IS NOT NULL":
		return fmt.Sprintf("%s %s", col, cnd.Operator), nil

	case "IN":
		if sub, ok := cnd.Value.(*SubQuery); ok {
			sql, args := sub.compile(c)
			return fmt.Sprintf("%s IN %s", col, sql), args
		}
		return fmt.Sprintf("%s IN ?", col), []any{cnd.Value}

	default:
		return fmt.Sprintf("%s %s ?", col, cnd.Operator), []any{cnd.Value}
	}
}
