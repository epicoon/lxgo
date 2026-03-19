package query

import (
	"fmt"
	"strings"
)

type compiler struct {
	tableAlias string
	joins      map[string]string
	counter    int
}

func newCompiler(alias string) *compiler {
	return &compiler{
		tableAlias: alias,
		joins:      make(map[string]string),
	}
}

func (c *compiler) relationAlias(relation string) string {
	if alias, ok := c.joins[relation]; ok {
		return alias
	}

	c.counter++
	alias := fmt.Sprintf("j%d", c.counter)

	c.joins[relation] = alias
	return alias
}

func (c *compiler) column(field string) string {
	// JSONB support: AuthData.role
	if strings.Contains(field, "->") {
		return field
	}

	parts := strings.Split(field, ".")

	if len(parts) == 1 {
		return fmt.Sprintf("%s.%s", c.tableAlias, toSnake(parts[0]))
	}

	// relation.field
	relation := parts[0]
	column := parts[1]

	alias := c.relationAlias(relation)
	c.joins[relation] = alias

	return fmt.Sprintf("%s.%s", alias, toSnake(column))
}

func (c *compiler) compileNode(n Node) (string, []any) {
	return n.compile(c)
}
