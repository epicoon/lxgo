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

	output        string
	outputKeyword string
	tree          cvt.ITree
	nStack        nodeStack
	widgets       []string
	lines         []string
	texts         []string
	currentLine   int
	currentDepth  int
	stdShift      int
	inHTML        bool
	err           string
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

func (p *lxmlParser) SetOutputKeyword(kw string) cvt.IParser {
	p.outputKeyword = kw
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
	c.SetOutputKeyword(p.outputKeyword)
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
	// Ignore empty lines
	line := p.lines[p.currentLine]
	if line == "" {
		return
	}
	re := regexp.MustCompile(`^\s*$`)
	if re.MatchString(line) {
		return
	}
	// Ignore comments
	re = regexp.MustCompile(`^\s*//.*$`)
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
		if depth > p.currentDepth {
			p.AddError(fmt.Sprintf("wrong depth: %d or less expected", p.currentDepth))
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
	synt := []string{"call", "if", "elseif", "else", "for"}
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

	// Remove custom line breaks - done before text extraction, so a widget's
	// attribute list joined via trailing "\" is a single physical line by the
	// time extractTexts looks at line heads to decide where a quote is
	// allowed to open a text attribute (see extractTexts).
	re = regexp.MustCompile(`\\(?:\r\n|\r|\n)\s*`)
	text = re.ReplaceAllString(text, " ")

	// Extract widget text attributes ('...' / "...") into placeholders
	text = p.extractTexts(text)

	// Get lines
	re = regexp.MustCompile(`(\r\n|\r|\n)`)
	text = re.ReplaceAllString(text, "\n")
	re = regexp.MustCompile(`\n\n+`)
	text = re.ReplaceAllString(text, "\n")
	p.lines = strings.Split(text, "\n")
}

var lxmlLineHeadRe = regexp.MustCompile(`^(?: |\t)*<([\w\d._]+)`)

// isLxmlWidgetLineHead reports whether line's head is a recognized widget
// tag (e.g. "<lx.Box>...") as opposed to raw HTML/SVG content, a control-flow
// line (if/for/call:...), or a reusable-block line (<*Name>/<&Name>) - none
// of which can carry a quoted text attribute.
func (p *lxmlParser) isLxmlWidgetLineHead(line string) bool {
	m := lxmlLineHeadRe.FindStringSubmatch(line)
	if m == nil {
		return false
	}
	return slices.Contains(p.Widgets(), m[1])
}

// lxmlLineAt returns the physical line (up to, but not including, the next
// newline) starting at position from in text.
func lxmlLineAt(text string, from int) string {
	if idx := strings.IndexByte(text[from:], '\n'); idx != -1 {
		return text[from : from+idx]
	}
	return text[from:]
}

// extractTexts scans text for widget text attributes delimited by a single
// or double quote, replacing each with a "[|N|]" placeholder (the same
// mechanism the widget parser already reads via Texts()[N]). A quote only
// opens a text attribute at paren/brace depth 0 (quotes inside a raw (...)
// or {...} attribute are left untouched, since those spans are consumed
// whole by the widget parser's own brace matching) AND on a line whose head
// is a recognized widget tag - this keeps quotes inside raw HTML/SVG content
// nested under a widget (e.g. `<path d="M10 10" fill="none"/>`) from being
// misread as text-attribute delimiters, since such lines are never widget
// lines themselves.
func (p *lxmlParser) extractTexts(text string) string {
	var b strings.Builder
	parenDepth := 0
	braceDepth := 0
	n := len(text)
	i := 0
	atLineStart := true
	lineAllowsText := false
	for i < n {
		if atLineStart {
			lineAllowsText = p.isLxmlWidgetLineHead(lxmlLineAt(text, i))
			atLineStart = false
		}

		c := text[i]
		switch {
		case c == '\n':
			atLineStart = true
			b.WriteByte(c)
			i++
		case c == '(':
			parenDepth++
			b.WriteByte(c)
			i++
		case c == ')':
			if parenDepth > 0 {
				parenDepth--
			}
			b.WriteByte(c)
			i++
		case c == '{':
			braceDepth++
			b.WriteByte(c)
			i++
		case c == '}':
			if braceDepth > 0 {
				braceDepth--
			}
			b.WriteByte(c)
			i++
		case (c == '\'' || c == '"') && parenDepth == 0 && braceDepth == 0 && lineAllowsText:
			quote := c
			start := i + 1
			closeIdx := findUnescapedByte(text, start, quote)
			if closeIdx == -1 {
				b.WriteByte(c)
				i++
				continue
			}

			raw := text[start:closeIdx]
			raw = strings.ReplaceAll(raw, "\\"+string(quote), string(quote))

			inx := len(p.texts)
			p.texts = append(p.texts, processLxmlText(raw))
			b.WriteString(fmt.Sprintf("[|%d|]", inx))
			i = closeIdx + 1
		default:
			b.WriteByte(c)
			i++
		}
	}
	return b.String()
}

// findUnescapedByte returns the index (>= from) of the next occurrence of
// target in text that is not preceded by an odd number of backslashes, or -1.
func findUnescapedByte(text string, from int, target byte) int {
	n := len(text)
	for j := from; j < n; j++ {
		if text[j] != target {
			continue
		}
		bs := 0
		k := j - 1
		for k >= from && text[k] == '\\' {
			bs++
			k--
		}
		if bs%2 == 0 {
			return j
		}
	}
	return -1
}

var lxmlWhitespaceRunRe = regexp.MustCompile(`\s+`)

const lxmlPreOpenTag = "<pre>"
const lxmlPreCloseTag = "</pre>"

// processLxmlText applies the widget text attribute's whitespace rules:
// outside <pre>...</pre>, any run of whitespace (including newlines)
// collapses to a single space (HTML-like); inside <pre>...</pre>, formatting
// is preserved verbatim except for a common leading indentation, taken from
// the indentation immediately preceding the <pre> tag itself, stripped from
// every line. Multiple <pre> spans are each dedented independently; nesting
// is not supported.
func processLxmlText(raw string) string {
	// A leading/trailing whitespace run that itself contains a newline is
	// structural (the author's own indentation between the opening/closing
	// quote and the real content) and is dropped entirely - unlike a same-line
	// boundary space/tab (e.g. `Color: `), which is meaningful and only gets
	// collapsed, never removed.
	original := raw
	raw = trimLxmlNewlineBoundary(raw, true)
	leadingCut := len(original) - len(raw)
	raw = trimLxmlNewlineBoundary(raw, false)

	var out strings.Builder
	pos := 0
	for {
		openIdx := strings.Index(raw[pos:], lxmlPreOpenTag)
		if openIdx == -1 {
			out.WriteString(collapseLxmlWhitespace(raw[pos:]))
			break
		}
		openIdx += pos

		closeIdx := strings.Index(raw[openIdx:], lxmlPreCloseTag)
		if closeIdx == -1 {
			out.WriteString(collapseLxmlWhitespace(raw[pos:]))
			break
		}
		closeIdx += openIdx

		out.WriteString(collapseLxmlWhitespace(raw[pos:openIdx]))

		// The indentation immediately preceding <pre> is read from the
		// original, untrimmed text so a <pre> that ends up at the very start
		// of raw (after the leading-newline trim above) still finds its own
		// line's indent rather than an empty prefix.
		origOpenIdx := openIdx + leadingCut
		lineStart := strings.LastIndex(original[:origOpenIdx], "\n") + 1
		indent := original[lineStart:origOpenIdx]
		if strings.TrimSpace(indent) != "" {
			indent = ""
		}

		content := raw[openIdx+len(lxmlPreOpenTag) : closeIdx]
		out.WriteString(dedentLxmlPre(content, indent))

		pos = closeIdx + len(lxmlPreCloseTag)
	}

	return out.String()
}

func collapseLxmlWhitespace(s string) string {
	return lxmlWhitespaceRunRe.ReplaceAllString(s, " ")
}

func isLxmlWhitespaceByte(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

// trimLxmlNewlineBoundary strips a leading (or, with leading=false, trailing)
// run of whitespace from s, but only if that run contains a newline.
func trimLxmlNewlineBoundary(s string, leading bool) string {
	n := len(s)
	sawNewline := false
	if leading {
		i := 0
		for i < n && isLxmlWhitespaceByte(s[i]) {
			if s[i] == '\n' || s[i] == '\r' {
				sawNewline = true
			}
			i++
		}
		if sawNewline {
			return s[i:]
		}
		return s
	}

	i := n
	for i > 0 && isLxmlWhitespaceByte(s[i-1]) {
		if s[i-1] == '\n' || s[i-1] == '\r' {
			sawNewline = true
		}
		i--
	}
	if sawNewline {
		return s[:i]
	}
	return s
}

func dedentLxmlPre(content string, indent string) string {
	content = trimLxmlNewlineBoundary(content, true)
	content = trimLxmlNewlineBoundary(content, false)
	if indent == "" {
		return content
	}
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimPrefix(line, indent)
	}
	return strings.Join(lines, "\n")
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
