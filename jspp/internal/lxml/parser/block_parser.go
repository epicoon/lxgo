package parser

import (
	"strings"

	"github.com/epicoon/lxgo/jspp/internal/lxml/cvt"
	"github.com/epicoon/lxgo/jspp/internal/lxml/tree"
)

type BlockParser struct {
	parser cvt.IParser
	node   *tree.BlockNode
}

var _ cvt.INodeParser = (*BlockParser)(nil)

/** @constructor */
func NewBlockParser(parser cvt.IParser, depth int) *BlockParser {
	return &BlockParser{
		parser: parser,
		node:   tree.NewBlockNode(depth),
	}
}

func (p *BlockParser) Run(head, line string) cvt.INode {
	n := p.node

	head = strings.Trim(head, "<>")
	if head[0] == '*' {
		n.IsLink = false
		n.Name = strings.TrimLeft(head, "*")
	} else if head[0] == '&' {
		n.IsLink = true
		n.Name = strings.TrimLeft(head, "&")
	} else {
		p.parser.AddError("wrong block syntax: * or & expected")
		return nil
	}

	code, _ := strings.CutSuffix(line, ">")
	if code == "" {
		n.Code = "()"
	} else {
		n.Code = code
	}

	return n
}
