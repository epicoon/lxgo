package tree

import (
	"github.com/epicoon/lxgo/jspp/internal/lxml/cvt"
)

/** @interface cvt.ITree */
type Tree struct {
	roots  []cvt.INode
	blocks []cvt.INode
}

var _ cvt.ITree = (*Tree)(nil)

/** @constructor */
func NewTree() *Tree {
	return &Tree{
		roots: make([]cvt.INode, 0),
	}
}

func (t *Tree) AddNode(n cvt.INode) {
	if n.Is(cvt.NodeTypeBlock) {
		bn := n.(*BlockNode)
		if !bn.IsLink {
			t.blocks = append(t.blocks, bn)
			return
		}
	}

	t.roots = append(t.roots, n)
}

func (t *Tree) EachBlock(f func(n cvt.INode)) {
	for _, node := range t.blocks {
		f(node)
	}
}

func (t *Tree) EachRoot(f func(n cvt.INode)) {
	for _, node := range t.roots {
		f(node)
	}
}
