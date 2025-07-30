package renderer

import (
	"errors"

	"github.com/epicoon/lxgo/jspp/internal/lxml/cvt"
	"github.com/epicoon/lxgo/jspp/internal/lxml/tree"
)

/** @interface cvt.INodeRenderer */
type widgetRenderer struct {
	parser cvt.IParser
	node   *tree.WidgetNode
}

var _ cvt.INodeRenderer = (*widgetRenderer)(nil)

/** @constructor */
func NewWidgetRenderer(parser cvt.IParser, node cvt.INode) (cvt.INodeRenderer, error) {
	n, ok := node.(*tree.WidgetNode)
	if !ok {
		return nil, errors.New("wrong node type: *tree.WidgetNode expected")
	}

	return &widgetRenderer{
		parser: parser,
		node:   n,
	}, nil
}

func (r *widgetRenderer) Run() string {

	//TODO

	return ""
}
