package parser

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/epicoon/lxgo/jspp/internal/lxml/cvt"
	"github.com/epicoon/lxgo/jspp/internal/lxml/tree"
	"github.com/epicoon/lxgo/jspp/internal/utils"
)

/** @interface cvt.INodeParser */
type WidgetParser struct {
	parser cvt.IParser
	node   *tree.WidgetNode
}

var _ cvt.INodeParser = (*WidgetParser)(nil)

/** @constructor */
func NewWidgetParser(parser cvt.IParser, depth int) *WidgetParser {
	return &WidgetParser{
		parser: parser,
		node:   tree.NewWidgetNode(depth),
	}
}

func (p *WidgetParser) Run(head, line string) cvt.INode {
	l := len(line)
	n := p.node
	for l > 0 {
		switch line[0] {
		case '@':
			line = p.getKey(line, n)
		case '.':
			line = p.getCss(line, n)
		case '[':
			line = p.getSquare(line, n)
		case '(':
			line = p.getBracket(line, n)
		case '#':
			line = p.getOctothorpe(line, n)
		case '{':
			line = p.getBraces(line, n)
		}

		newL := len(line)
		if newL == l {
			p.parser.AddError("unexpected line format")
			return nil
		} else {
			l = newL
		}

		if p.parser.HasError() {
			return nil
		}
	}

	n.Widget = strings.Trim(head, "<>")
	return n
}

func (p *WidgetParser) getKey(line string, n *tree.WidgetNode) string {
	re := regexp.MustCompile(`^@(\b[\w\d_]+\b)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) != 2 {
		p.parser.AddError("wrong key format")
		return ""
	}

	n.Key = matches[1]

	re = regexp.MustCompile(`^@\b[\w\d_]+\b\s*`)
	line = re.ReplaceAllString(line, "")
	return line
}

func (p *WidgetParser) getCss(line string, n *tree.WidgetNode) string {
	re := regexp.MustCompile(`^\.(\b[\w\d_-]+\b)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) != 2 {
		p.parser.AddError("wrong css format")
		return ""
	}

	n.Css = append(n.Css, matches[1])

	re = regexp.MustCompile(`^\.\b[\w\d_-]+\b\s*`)
	line = re.ReplaceAllString(line, "")
	return line
}

func (p *WidgetParser) getSquare(line string, n *tree.WidgetNode) string {
	// [_]
	if strings.HasPrefix(line, "[_]") {
		n.IsVolume = true
		re := regexp.MustCompile(`^\[_\]\s*`)
		line = re.ReplaceAllString(line, "")
		return line
	}

	// Text
	re := regexp.MustCompile(`^\[\|(\d+)\|\]`)
	matches := re.FindStringSubmatch(line)
	if len(matches) == 2 {
		inx, _ := strconv.Atoi(matches[1])
		n.Text = p.parser.Texts()[inx]
		re = regexp.MustCompile(`^\[\|\d+\|\]\s*`)
		line = re.ReplaceAllString(line, "")
		return line
	}

	// [f:name] or [field:name]
	// [m:name] or [matrix:name]
	re = regexp.MustCompile(`^\[(f|m|field|matrix):([^\]]+)\]`)
	matches = re.FindStringSubmatch(line)
	if len(matches) == 3 {
		key := matches[1]
		val := matches[2]
		if key == "f" || key == "field" {
			n.Field = val
		} else if key == "m" || key == "matrix" {
			n.Field = val
			n.IsMatrix = true
		}
		re = regexp.MustCompile(`^\[(?:f|m|field|matrix):[^\]]+\]\s*`)
		line = re.ReplaceAllString(line, "")
		return line
	}

	// [10:10:10:10]
	// [10:10:10:10]px
	// [10%:10%:10px:10px]
	// [::10:10:10:10]
	re = regexp.MustCompile(`^\[(?:(\d+)([\w%]*))?:(?:(\d+)([\w%]*))?:(?:(\d+)([\w%]*))?:(?:(\d+)([\w%]*))?(?::(\d+)([\w%]*))?(?::(\d+)([\w%]*))?\]([\w%]+)?`)
	matches = re.FindStringSubmatch(line)
	if len(matches) != 14 {
		p.parser.AddError("wrong square brackets format")
		return ""
	}

	cUnit := matches[13]
	for i := 1; i < 13; i += 2 {
		val := matches[i]
		unit := matches[i+1]

		res := "null"
		if val != "" {
			res = val
			if unit != "" {
				res += unit
			} else if cUnit != "" {
				res += cUnit
			}
		}

		n.Geom = append(n.Geom, res)
	}
	re = regexp.MustCompile(`^\[(?:\d+[\w%]*)?:(?:\d+[\w%]*)?:(?:\d+[\w%]*)?:(?:\d+[\w%]*)?(?::\d+[\w%]*)?(?::\d+[\w%]*)?\](?:[\w%]+)?\s*`)
	line = re.ReplaceAllString(line, "")
	return line
}

func (p *WidgetParser) getBracket(line string, n *tree.WidgetNode) string {
	end := utils.FindMatchingBrace(line, 0, '(')
	if end == -1 {
		p.parser.AddError("wrong brackets format")
		return ""
	}

	n.Config = line[1:end]
	re := regexp.MustCompile(`^\(` + regexp.QuoteMeta(n.Config) + `\)\s*`)
	line = re.ReplaceAllString(line, "")
	return line
}

func (p *WidgetParser) getOctothorpe(line string, n *tree.WidgetNode) string {
	re := regexp.MustCompile(`^#(\b[\w\d_]+\b)\(`)
	matches := re.FindStringSubmatch(line)
	if len(matches) == 2 {
		end := utils.FindMatchingBrace(line, 0, '(')
		if end == -1 {
			p.parser.AddError("wrong function format")
			return ""
		}
		f := line[1:end]
		arr := strings.SplitN(f, "(", 2)
		n.Methods[matches[1]] = arr[1]
		n.MethodsSeq = append(n.MethodsSeq, matches[1])

		re := regexp.MustCompile(`^#` + regexp.QuoteMeta(f) + `\)\s*`)
		line = re.ReplaceAllString(line, "")
		return line
	}

	re = regexp.MustCompile(`^#(\b[\w\d_]+\b)`)
	matches = re.FindStringSubmatch(line)
	if len(matches) != 2 {
		p.parser.AddError("wrong octothorpe format")
		return ""
	}

	n.Methods[matches[1]] = ""
	n.MethodsSeq = append(n.MethodsSeq, matches[1])
	re = regexp.MustCompile(`^#\b[\w\d_]+\b\s*`)
	line = re.ReplaceAllString(line, "")
	return line
}

func (p *WidgetParser) getBraces(line string, n *tree.WidgetNode) string {
	end := utils.FindMatchingBrace(line, 0, '{')
	if end == -1 {
		p.parser.AddError("wrong brackets format")
		return ""
	}

	n.Data = line[1:end]
	re := regexp.MustCompile(`^\{` + regexp.QuoteMeta(n.Data) + `\}\s*`)
	line = re.ReplaceAllString(line, "")
	return line
}
