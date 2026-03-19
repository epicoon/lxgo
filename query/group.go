package query

import "strings"

type Group struct {
	Operator string // AND / OR
	Nodes    []Node
}

func (g *Group) compile(c *compiler) (string, []any) {
	var parts []string
	var args []any

	for _, n := range g.Nodes {
		sql, a := n.compile(c)
		parts = append(parts, sql)
		args = append(args, a...)
	}

	return "(" + strings.Join(parts, " "+g.Operator+" ") + ")", args
}
