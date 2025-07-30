package parser

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/epicoon/lxgo/jspp/internal/lxml/cvt"
	"github.com/epicoon/lxgo/jspp/internal/lxml/tree"
)

type SyntaxParser struct {
	parser cvt.IParser
	node   *tree.SyntaxNode
}

var _ cvt.INodeParser = (*SyntaxParser)(nil)

/** @constructor */
func NewSyntaxParser(parser cvt.IParser, depth int) *SyntaxParser {
	return &SyntaxParser{
		parser: parser,
		node:   tree.NewSyntaxNode(depth),
	}
}

func (p *SyntaxParser) Run(head, line string) cvt.INode {
	n := p.node
	n.Op = head

	if head == "else" {
		return n
	}

	if head == "if" || head == "elseif" {
		n.Code, _ = strings.CutSuffix(line, ":")
		return n
	}

	if head == "for" {
		// for i = 1; i < count; i++:
		re := regexp.MustCompile(`(let\s+)?[\w\d_]+\s*=\s*.+?\s*;\s*[\w\d_]+\s*(?:>=|<=|>|<)\s*.+?\s*;\s*[\w\d_](?:\+\+|--|\+=.+?|-=.+?)\s*:`)
		match := re.FindStringSubmatch(line)
		if len(match) == 2 {
			code, _ := strings.CutSuffix(match[0], ":")
			if match[1] == "" {
				n.Code = "len " + code
			} else {
				n.Code = code
			}
			return n
		}

		var iter, op, from, to string
		ok := false

		// for i < lim:
		// for i <= lim:
		// for < lim:
		// for <= lim:
		re = regexp.MustCompile(`(?:(.+?)\s*)?(<=|<)\s*(.+):`)
		match = re.FindStringSubmatch(line)
		if len(match) == 4 {
			ok = true
			if match[1] == "" {
				iter = "_iter"
			} else {
				iter = match[1]
			}
			op = match[2]
			from = "0"
			to = match[3]
		}

		// for i = 1 to lim:
		// for 1 to lim:
		// for lim:
		if !ok {
			re = regexp.MustCompile(`(?:(.+?)\s*=\s*)?(?:(.+?)\s+to\s+)?(.+)\s*:`)
			match = re.FindStringSubmatch(line)
			if len(match) == 4 {
				ok = true
				op = "<="
				if match[1] == "" {
					iter = "_iter"
				} else {
					iter = match[1]
				}
				if match[2] == "" {
					from = "0"
				} else {
					from = match[2]
				}
				to = match[3]
			}
		}

		if ok {
			n.Code = fmt.Sprintf("let %s=%s;%s%s%s;%s++", iter, from, iter, op, to, iter)
			return n
		}
	}

	p.parser.AddError("unknown syntax: " + head)
	return nil
}
