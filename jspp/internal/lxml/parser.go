package lxml

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/epicoon/lxgo/jspp"
	"github.com/epicoon/lxgo/jspp/internal/lxml/compiler"
	"github.com/epicoon/lxgo/jspp/internal/lxml/cvt"
	"github.com/epicoon/lxgo/jspp/internal/lxml/parser"
	"github.com/epicoon/lxgo/jspp/internal/lxml/tree"
)

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * cvt.IParser
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

/** @interface cvt.IParser */
type lxmlParser struct {
	pp jspp.IPreprocessor

	output       string
	tree         cvt.ITree
	nStack       nodeStack
	widgets      []string
	lines        []string
	texts        []string
	currentLine  int
	currentDepth int
	stdShift     int
	inHTML       bool
	err          string
}

var _ cvt.IParser = (*lxmlParser)(nil)

/** @constructor */
func NewParser(pp jspp.IPreprocessor) *lxmlParser {
	return &lxmlParser{
		pp:     pp,
		tree:   tree.NewTree(),
		nStack: *newNodeStack(),
	}
}

func (p *lxmlParser) SetOutput(out string) cvt.IParser {
	p.output = out
	return p
}

func (p *lxmlParser) ParseFile(path string) (string, error) {
	path = p.pp.App().Pathfinder().GetAbsPath(path)

	d, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file '%s' not found", path)
		}
	}

	text := string(d)
	if text == "" {
		return "", nil
	}

	return p.ParseText(text)
}

func (p *lxmlParser) ParseText(text string) (string, error) {
	p.splitLines(text)
	if len(p.lines) == 0 {
		return "", nil
	}

	// Create tree
	l := len(p.lines)
	for i := 0; i < l; i++ {
		p.parseLine()
		if p.HasError() {
			return "", errors.New(p.err)
		}
		p.currentLine++
	}

	// Compile code
	c := compiler.NewTreeCompiler(p)
	c.SetOutput(p.output)
	code := c.Run()

	if p.HasError() {
		return "", errors.New(p.err)
	}

	return code, nil
}

func (p *lxmlParser) AddError(msg string) {
	p.err = fmt.Sprintf("error while parsing line[%d]: \"%s\" - %s", p.currentLine, p.lines[p.currentLine], msg)
}

func (p *lxmlParser) HasError() bool {
	return p.err != ""
}

func (p *lxmlParser) Texts() []string {
	return p.texts
}

func (p *lxmlParser) Tree() cvt.ITree {
	return p.tree
}

func (p *lxmlParser) Widgets() []string {
	if p.widgets == nil {
		p.widgets = []string{"lx.Rect", "lx.Box"}
		p.pp.ModulesMap().Each(func(data jspp.IJSModuleData) {
			if !data.HasData() {
				return
			}
			d := data.Data()
			val, exists := d["widget"]
			if !exists {
				return
			}
			p.widgets = append(p.widgets, val.(string))
		})
	}
	return p.widgets
}

func (p *lxmlParser) parseLine() {
	line := p.lines[p.currentLine]
	if line == "" {
		return
	}
	re := regexp.MustCompile(`^\s*$`)
	if re.MatchString(line) {
		return
	}

	re = regexp.MustCompile(`^((?: |\t)*)(<?[/*&]?\b[\w\d\._]+\b>?)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) != 3 {
		p.AddError("wrong line format")
		return
	}

	// Define line type
	head := matches[2]
	lineType := p.defineLineType(head)
	if lineType == cvt.NodeTypeHtml {
		n := p.nStack.Peek()
		if n == nil {
			p.AddError("HTML code should be wrapped by widget")
			return
		}
		n.AddHTML(line)
		if !p.inHTML {
			p.inHTML = true
		}
		return
	}

	// Define depth
	shift := matches[1]
	depth := p.getDepth(shift)
	if p.HasError() {
		return
	}

	// Return from HTML
	if p.inHTML {
		if depth != p.currentDepth {
			p.AddError(fmt.Sprintf("wrong depth: %d expected", p.currentDepth))
			return
		}
		p.inHTML = false
	}

	// Actualize stack
	change := p.currentDepth - depth
	if change >= 0 {
		p.nStack.Drop(change + 1)
	}
	p.currentDepth = depth

	// Cut head
	re = regexp.MustCompile(`^(?: |\t)*<?[/*&]?\b[\w\d\._]+\b>?\s*`)
	line = re.ReplaceAllString(line, "")

	// Make node
	nodeParser := p.newNodeParser(lineType, depth)
	if nodeParser == nil {
		p.AddError("unknown line type")
		return
	}
	n := nodeParser.Run(head, line)
	if p.HasError() {
		return
	}

	// Check block definition
	if n.Is(cvt.NodeTypeBlock) {
		bn := n.(*tree.BlockNode)
		if !bn.IsLink && p.currentDepth != 0 {
			p.AddError("block should be defined on 0 depth level")
			return
		}
	}

	// Collect tree
	if p.nStack.IsEmpty() {
		p.tree.AddNode(n)
	} else {
		p.nStack.Peek().AddNode(n)
	}
	p.nStack.Push(n)
}

func (p *lxmlParser) getDepth(shift string) int {
	if shift == "" {
		return 0
	}

	if p.stdShift == 0 {
		if p.currentLine == 0 {
			p.AddError("wrong depth: 0 expected")
			return 0
		}
		p.stdShift = len(shift)
	}

	shiftLen := len(shift)

	rez := shiftLen % p.stdShift
	if rez != 0 {
		p.AddError(fmt.Sprintf("wrong depth: should be multiple of %d", p.stdShift))
		return 0
	}

	depth := shiftLen / p.stdShift

	if depth > p.currentDepth+1 {
		p.AddError(fmt.Sprintf("wrong depth: no more then %d expected", p.currentDepth+1))
		return 0
	}

	return depth
}

func (p *lxmlParser) newNodeParser(lineType, depth int) cvt.INodeParser {
	switch lineType {
	case cvt.NodeTypeWidget:
		return parser.NewWidgetParser(p, depth)
	case cvt.NodeTypeSyntax:
		return parser.NewSyntaxParser(p, depth)
	case cvt.NodeTypeBlock:
		return parser.NewBlockParser(p, depth)
	}

	return nil
}

func (p *lxmlParser) defineLineType(head string) int {
	synt := []string{"if", "elseif", "else", "for"}
	if slices.Contains(synt, head) {
		return cvt.NodeTypeSyntax
	}

	if head[0] == '<' {
		if head[1] == '*' || head[1] == '&' {
			return cvt.NodeTypeBlock
		}

		tag := strings.Trim(head, "<>")
		widgets := p.Widgets()
		if slices.Contains(widgets, tag) {
			return cvt.NodeTypeWidget
		}
	}

	return cvt.NodeTypeHtml
}

func (p *lxmlParser) splitLines(text string) {
	p.texts = make([]string, 0)

	// Remove first empty lines
	re := regexp.MustCompile(`^(\s*(\r\n|\r|\n))*`)
	text = re.ReplaceAllString(text, "")

	// Remove common shift
	re = regexp.MustCompile(`^(?: |\t)*`)
	shift := re.FindString(text)
	if shift != "" {
		re = regexp.MustCompile(`(^|\r\n|\r|\n)` + shift)
		text = re.ReplaceAllString(text, "$1")
	}

	// Escape \`
	text = strings.ReplaceAll(text, "\\`", "[|||]")

	// Escape `...`
	re = regexp.MustCompile("`[^`]*?`")
	text = re.ReplaceAllStringFunc(text, func(s string) string {
		inx := len(p.texts)
		s = strings.Trim(s, "`")
		s = strings.ReplaceAll(s, "[|||]", "`")
		p.texts = append(p.texts, s)
		return fmt.Sprintf("[|%d|]", inx)
	})

	// Remove custom line breaks
	re = regexp.MustCompile(`\\(?:\r\n|\r|\n)\s*`)
	text = re.ReplaceAllString(text, " ")

	// Get lines
	re = regexp.MustCompile(`(\r\n|\r|\n)`)
	text = re.ReplaceAllString(text, "\n")
	re = regexp.MustCompile(`\n\n+`)
	text = re.ReplaceAllString(text, "\n")
	p.lines = strings.Split(text, "\n")
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * nodeStack
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

type nodeStack struct {
	stack []cvt.INode
}

func newNodeStack() *nodeStack {
	return &nodeStack{
		stack: make([]cvt.INode, 0),
	}
}
func (s *nodeStack) Push(n cvt.INode) {
	s.stack = append(s.stack, n)
}

func (s *nodeStack) Drop(count int) {
	l := len(s.stack)
	if count == 0 || l == 0 {
		return
	}

	if count >= l {
		s.stack = make([]cvt.INode, 0)
		return
	}

	s.stack = s.stack[:len(s.stack)-count]
}

func (s *nodeStack) Pop() cvt.INode {
	if len(s.stack) == 0 {
		return nil
	}
	last := s.stack[len(s.stack)-1]
	s.stack = s.stack[:len(s.stack)-1]
	return last
}

func (s *nodeStack) Peek() cvt.INode {
	if len(s.stack) == 0 {
		return nil
	}
	return s.stack[len(s.stack)-1]
}

func (s *nodeStack) Len() int {
	return len(s.stack)
}

func (s *nodeStack) IsEmpty() bool {
	return len(s.stack) == 0
}
