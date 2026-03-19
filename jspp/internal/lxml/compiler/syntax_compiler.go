package compiler

import (
	"errors"
	"fmt"

	"github.com/epicoon/lxgo/jspp/internal/lxml/cvt"
	"github.com/epicoon/lxgo/jspp/internal/lxml/tree"
)

/** @interface cvt.INodeCompiler */
type syntaxCompiler struct {
	tc   *treeCompiler
	node *tree.SyntaxNode
}

var _ iNodeCompiler = (*syntaxCompiler)(nil)

/** @constructor */
func newSyntaxCompiler(tc *treeCompiler, node cvt.INode) (*syntaxCompiler, error) {
	n, ok := node.(*tree.SyntaxNode)
	if !ok {
		return nil, errors.New("wrong node type: *tree.WidgetNode expected")
	}

	return &syntaxCompiler{
		tc:   tc,
		node: n,
	}, nil
}

func (c *syntaxCompiler) run() string {
	op := c.node.Op
	if op == "call" {
		return c.node.Code
	}

	code := ""
	if c.node.Code != "" {
		code = "(" + c.node.Code + ")"
	}

	if op == "elseif" {
		op = "else if"
	}

	return fmt.Sprintf("%s %s{%s}", op, code, c.tc.compileContent(c.node))
}
