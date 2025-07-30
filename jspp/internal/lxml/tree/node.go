package tree

import "github.com/epicoon/lxgo/jspp/internal/lxml/cvt"

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * Abstract node
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

/** @interface cvt.INode */
type node struct {
	tp     int
	depth  int
	nested []cvt.INode

	html string
}

var _ cvt.INode = (*node)(nil)

func newNode(tp, depth int) *node {
	return &node{
		tp:     tp,
		depth:  depth,
		nested: make([]cvt.INode, 0),
	}
}

func (n *node) Type() int {
	return n.tp
}

func (n *node) Is(tp int) bool {
	return n.tp == tp
}

func (n *node) Depth() int {
	return n.depth
}

func (n *node) AddNode(nested cvt.INode) {
	n.nested = append(n.nested, nested)
}

func (n *node) AddHTML(html string) {
	if n.html != "" {
		n.html += "\n"
	}
	n.html += html
}

func (n *node) HTML() string {
	return n.html
}

func (n *node) IsEmpty() bool {
	return len(n.nested) == 0
}

func (n *node) Nested() []cvt.INode {
	return n.nested
}

func (n *node) EachNested(f func(n cvt.INode)) {
	for _, n := range n.nested {
		f(n)
	}
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * WidgetNode
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

/** @interface cvt.INode */
type WidgetNode struct {
	*node
	Widget   string
	Key      string
	Field    string
	IsMatrix bool
	Geom     []string
	IsVolume bool
	Css      []string
	Text     string

	//TODO do we need to parse this into map[string]any ?
	Config     string
	Data       string
	Methods    map[string]string
	MethodsSeq []string
}

/** @constructor */
func NewWidgetNode(depth int) *WidgetNode {
	return &WidgetNode{
		node:    newNode(cvt.NodeTypeWidget, depth),
		Css:     make([]string, 0),
		Methods: make(map[string]string, 0),
	}
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * SyntaxNode
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

/** @interface cvt.INode */
type SyntaxNode struct {
	*node
	// if, elseif, else, for
	Op   string
	Code string
}

/** @constructor */
func NewSyntaxNode(depth int) *SyntaxNode {
	return &SyntaxNode{
		node: newNode(cvt.NodeTypeSyntax, depth),
	}
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * BlockNode
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

/** @interface cvt.INode */
type BlockNode struct {
	*node
	IsLink bool
	Name   string
	Code   string
}

/** @constructor */
func NewBlockNode(depth int) *BlockNode {
	return &BlockNode{
		node: newNode(cvt.NodeTypeBlock, depth),
	}
}
