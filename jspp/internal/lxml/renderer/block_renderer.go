package renderer

import (
	"errors"

	"github.com/epicoon/lxgo/jspp/internal/lxml/cvt"
	"github.com/epicoon/lxgo/jspp/internal/lxml/tree"
)

/** @interface cvt.INodeRenderer */
type blockRenderer struct {
	parser cvt.IParser
	node   *tree.BlockNode
}

var _ cvt.INodeRenderer = (*blockRenderer)(nil)

/** @constructor */
func NewBlockRenderer(parser cvt.IParser, node cvt.INode) (cvt.INodeRenderer, error) {
	n, ok := node.(*tree.BlockNode)
	if !ok {
		return nil, errors.New("wrong node type: *tree.WidgetNode expected")
	}

	return &blockRenderer{
		parser: parser,
		node:   n,
	}, nil
}

func (r *blockRenderer) Run() string {

	// TODO

	return ""
}
