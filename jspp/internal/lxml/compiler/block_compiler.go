package compiler

import (
	"errors"
	"fmt"

	"github.com/epicoon/lxgo/jspp/internal/lxml/cvt"
	"github.com/epicoon/lxgo/jspp/internal/lxml/tree"
)

/** @interface cvt.INodeCompiler */
type blockCompiler struct {
	tc    *treeCompiler
	node  *tree.BlockNode
	fName string
}

var _ iNodeCompiler = (*blockCompiler)(nil)

/** @constructor */
func newBlockCompiler(tc *treeCompiler, node cvt.INode) (*blockCompiler, error) {
	n, ok := node.(*tree.BlockNode)
	if !ok {
		return nil, errors.New("wrong node type: *tree.WidgetNode expected")
	}

	return &blockCompiler{
		tc:   tc,
		node: n,
	}, nil
}

func (c *blockCompiler) run() string {
	if !c.node.IsLink {
		return fmt.Sprintf("function %s%s{%s}", c.fName, c.node.Code, c.tc.compileContent(c.node))
	}

	// tree.EachBlock (definitions) always runs before EachRoot (which is
	// where a link node like this one gets compiled), so c.tc.fMap already
	// holds every real definition by this point - a name not there means
	// this link points at a block that was never defined.
	if _, exists := c.tc.fMap[c.node.Name]; !exists {
		c.tc.parser.AddError(fmt.Sprintf("undefined block '%s'", c.node.Name))
		return ""
	}

	return fmt.Sprintf("[|%s|]%s;", c.node.Name, c.node.Code)
}
