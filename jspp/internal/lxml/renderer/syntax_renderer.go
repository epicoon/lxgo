package renderer

import (
	"errors"

	"github.com/epicoon/lxgo/jspp/internal/lxml/cvt"
	"github.com/epicoon/lxgo/jspp/internal/lxml/tree"
)

/** @interface cvt.INodeRenderer */
type syntaxRenderer struct {
	parser cvt.IParser
	node   *tree.SyntaxNode
}

var _ cvt.INodeRenderer = (*syntaxRenderer)(nil)

/** @constructor */
func NewSyntaxRenderer(parser cvt.IParser, node cvt.INode) (cvt.INodeRenderer, error) {
	n, ok := node.(*tree.SyntaxNode)
	if !ok {
		return nil, errors.New("wrong node type: *tree.WidgetNode expected")
	}

	return &syntaxRenderer{
		parser: parser,
		node:   n,
	}, nil
}

func (r *syntaxRenderer) Run() string {

	// TODO

	return ""
}
