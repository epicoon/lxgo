package md

import (
	"fmt"
	"regexp"
	"strings"
)

// renderer renders a parsed block tree to an HTML string.
type renderer struct {
	result     strings.Builder
	useWrapper bool
}

func newRenderer() *renderer {
	return &renderer{useWrapper: true}
}

func (r *renderer) setUseWrapper(v bool) *renderer {
	r.useWrapper = v
	return r
}

func (r *renderer) run(blocks []mdBlock) string {
	r.result.Reset()
	r.renderMap(blocks)
	rendered := r.renderInLines(r.result.String())
	if !r.useWrapper {
		return rendered
	}
	return r.openTag("div", nil) + rendered + "</div>"
}

func (r *renderer) renderMap(blocks []mdBlock) {
	for _, blk := range blocks {
		r.renderBlock(blk)
	}
}

func (r *renderer) renderBlock(blk mdBlock) {
	switch blk.blockType {
	case typeLine:
		r.result.WriteString("<hr>")
	case typeTitle1:
		r.renderTitle("h1", blk.lines[0].line)
	case typeTitle2:
		r.renderTitle("h2", blk.lines[0].line)
	case typeTitle3:
		r.renderTitle("h3", blk.lines[0].line)
	case typeTitle4:
		r.renderTitle("h4", blk.lines[0].line)
	case typeTitle5:
		r.renderTitle("h5", blk.lines[0].line)
	case typeTitle6:
		r.renderTitle("h6", blk.lines[0].line)
	case typeTable:
		r.renderTable(blk)
	case typeParagraph:
		r.renderParagraph(blk)
	case typeCodeBlock:
		r.renderCodeBlock(blk)
	case typeCodeBlockTyped:
		r.renderCodeBlockTyped(blk)
	case typeBlockquote:
		r.renderBlockquote(blk)
	case typeOrderedList, typeUnorderedList:
		r.renderList(blk)
	}
}

var (
	titleHashStripRe = regexp.MustCompile(`^#{1,6}`)
	titleIdRe        = regexp.MustCompile(`\{#([\w\d_-]+?)\}$`)
	titleIdStripRe   = regexp.MustCompile(`\s*\{#[\w\d_-]+?\}$`)
)

func (r *renderer) renderTitle(tag, line string) {
	line = titleHashStripRe.ReplaceAllString(line, "")
	if m := titleIdRe.FindStringSubmatch(line); m != nil {
		line = titleIdStripRe.ReplaceAllString(line, "")
		r.result.WriteString(r.openTag(tag, map[string]string{"id": m[1]}))
	} else {
		r.result.WriteString(r.openTag(tag, nil))
	}
	r.result.WriteString(line + "</" + tag + ">")
}

var (
	tableSepRe   = regexp.MustCompile(`^\|(\s*:?-+:?\s*\|)+\s*$`)
	tableSplitRe = regexp.MustCompile(`\s*\|\s*`)
)

func (r *renderer) renderTable(blk mdBlock) {
	r.result.WriteString(r.openTag("table", nil))

	firstRow := 0
	var aligns []string

	hasSubHeader := false
	var subHeaderLine string
	if len(blk.lines) > 1 {
		subHeaderLine = blk.lines[1].line
		hasSubHeader = tableSepRe.MatchString(subHeaderLine)
	}

	if hasSubHeader {
		firstRow = 2
		header := strings.Trim(blk.lines[0].line, "| ")
		titles := tableSplitRe.Split(header, -1)
		r.result.WriteString(r.openTag("thead", nil))
		r.result.WriteString(r.openTag("tr", nil))
		for _, title := range titles {
			r.result.WriteString(r.openTag("th", map[string]string{"style": "text-align: center"}))
			r.result.WriteString(title + "</th>")
		}
		r.result.WriteString("</tr></thead>")

		rawAligns := tableSplitRe.Split(strings.Trim(subHeaderLine, "| "), -1)
		aligns = make([]string, len(rawAligns))
		for i, a := range rawAligns {
			pre := len(a) > 0 && a[0] == ':'
			post := len(a) > 0 && a[len(a)-1] == ':'
			switch {
			case pre && post:
				aligns[i] = "center"
			case pre:
				aligns[i] = "left"
			case post:
				aligns[i] = "right"
			default:
				aligns[i] = "none"
			}
		}
	}

	r.result.WriteString(r.openTag("tbody", nil))
	for i := firstRow; i < len(blk.lines); i++ {
		r.result.WriteString(r.openTag("tr", nil))
		line := blk.lines[i].line
		values := tableSplitRe.Split(strings.Trim(line, "| "), -1)
		for col, value := range values {
			align := "none"
			if col < len(aligns) {
				align = aligns[col]
			}
			var params map[string]string
			if align != "none" {
				params = map[string]string{"style": "text-align: " + align}
			}
			r.result.WriteString(r.openTag("td", params))
			r.result.WriteString(value + "</td>")
		}
		r.result.WriteString("</tr>")
	}
	r.result.WriteString("</tbody>")

	r.result.WriteString("</table>")
}

func (r *renderer) renderParagraph(blk mdBlock) {
	r.result.WriteString(r.openTag("p", nil))
	var sb strings.Builder
	for _, ld := range blk.lines {
		line := trailingDoubleSpaceRe.ReplaceAllString(ld.line, "<br>")
		sb.WriteString(" " + line)
	}
	r.result.WriteString(sb.String() + "</p>")
}

func (r *renderer) renderCodeBlock(blk mdBlock) {
	r.result.WriteString(r.openTag("pre", nil))
	lines := make([]string, len(blk.lines))
	for i, ld := range blk.lines {
		lines[i] = escapeHTML(leading4SpacesRe.ReplaceAllString(ld.line, ""))
	}
	r.result.WriteString(strings.Join(lines, "<br>") + "</pre>")
}

var codeFencePrefixRe = regexp.MustCompile("^(```|~~~)")

func (r *renderer) renderCodeBlockTyped(blk mdBlock) {
	r.result.WriteString("<div>")
	var params map[string]string
	if blk.codeType != "" {
		params = map[string]string{"code-type": blk.codeType}
	}
	r.result.WriteString(r.openTag("pre", params))
	var lines []string
	for _, ld := range blk.lines {
		if codeFencePrefixRe.MatchString(ld.line) {
			continue
		}
		lines = append(lines, escapeHTML(ld.originLine))
	}
	r.result.WriteString(strings.Join(lines, "<br>") + "</pre>")
	r.result.WriteString(`<img src="" onerror="if(lx.MdHighlighter)lx.MdHighlighter.highlight(this.parentNode.children[0]);this.parentNode.removeChild(this)">`)
}

// escapeHTML escapes the characters that would otherwise be parsed as markup
// when source code is inserted directly into rendered HTML.
func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func (r *renderer) renderBlockquote(blk mdBlock) {
	r.result.WriteString(r.openTag("blockquote", nil))
	sub := newRenderer().setUseWrapper(false)
	r.result.WriteString(sub.run(blk.content) + "</blockquote>")
}

func (r *renderer) renderList(blk mdBlock) {
	tag := "ul"
	itemRe := unorderedItemRe
	if blk.blockType == typeOrderedList {
		tag = "ol"
		itemRe = orderedItemRe
	}
	r.result.WriteString(r.openTag(tag, nil))

	for _, ld := range blk.lines {
		line := itemRe.ReplaceAllString(ld.line, "")
		r.result.WriteString(r.openTag("li", nil))
		r.result.WriteString(line)
		if ld.contentSet {
			sub := newRenderer().setUseWrapper(false)
			r.result.WriteString(sub.run(ld.content))
		}
		r.result.WriteString("</li>")
	}
	r.result.WriteString("</" + tag + ">")
}

var (
	imgRe    = regexp.MustCompile(`!\[([^\]]+?)\]\(([^"]+?)\s*(?:"([^"]+?)")?\)`)
	linkRe   = regexp.MustCompile(`\[([^\]]+?)\]\(([^"]+?)\s*(?:"([^"]+?)")?\)`)
	boldRe   = regexp.MustCompile(`(?:(?:\*\*(.+?)\*\*)|(?:\b__(.+?)__\b))`)
	italicRe = regexp.MustCompile(`(?:(?:\*(.+?)\*)|(?:\b_(.+?)_\b))`)
	delRe    = regexp.MustCompile(`~~(.+?)~~`)
	markRe   = regexp.MustCompile(`==(.+?)==`)
	subRe    = regexp.MustCompile(`~(.+?)~`)
	supRe    = regexp.MustCompile(`\^(.+?)\^`)
	codeRe   = regexp.MustCompile("`(.+?)`")
)

var preBlockRe = regexp.MustCompile(`(?s)<pre\b[^>]*>.*?</pre>`)

// renderInLines applies inline formatting as a sequence of regex passes over
// the fully-rendered block HTML. The order is significant: images before
// links (image syntax is a superset of link syntax), bold before italic (**
// would otherwise be partially eaten by a single-star match), del before sub
// (~~ would otherwise be partially eaten by a single-tilde match).
//
// Code block content is protected from all of these passes first: source
// code routinely contains sequences that look like inline markdown syntax
// (e.g. "[T any](a, b T)" in a generic function signature), and that must be
// displayed literally, not reinterpreted.
func (r *renderer) renderInLines(html string) string {
	var preBlocks []string
	html = preBlockRe.ReplaceAllStringFunc(html, func(m string) string {
		preBlocks = append(preBlocks, m)
		return fmt.Sprintf("\x00%d\x00", len(preBlocks)-1)
	})

	html = imgRe.ReplaceAllStringFunc(html, func(m string) string {
		sub := imgRe.FindStringSubmatch(m)
		alt, link, title := sub[1], sub[2], sub[3]
		if title != "" {
			return fmt.Sprintf(`<img src="%s" title="%s" alt="%s">`, link, title, alt)
		}
		return fmt.Sprintf(`<img src="%s" alt="%s">`, link, alt)
	})

	html = linkRe.ReplaceAllStringFunc(html, func(m string) string {
		sub := linkRe.FindStringSubmatch(m)
		content, link, title := sub[1], sub[2], sub[3]
		if title != "" {
			return fmt.Sprintf(`<a href="%s" title="%s">%s</a>`, link, title, content)
		}
		return fmt.Sprintf(`<a href="%s">%s</a>`, link, content)
	})

	html = boldRe.ReplaceAllStringFunc(html, func(m string) string {
		sub := boldRe.FindStringSubmatch(m)
		text := sub[1]
		if text == "" {
			text = sub[2]
		}
		return "<strong>" + text + "</strong>"
	})

	html = italicRe.ReplaceAllStringFunc(html, func(m string) string {
		sub := italicRe.FindStringSubmatch(m)
		text := sub[1]
		if text == "" {
			text = sub[2]
		}
		return "<em>" + text + "</em>"
	})

	html = delRe.ReplaceAllString(html, "<del>$1</del>")
	html = markRe.ReplaceAllString(html, "<mark>$1</mark>")
	html = subRe.ReplaceAllString(html, "<sub>$1</sub>")
	html = supRe.ReplaceAllString(html, "<sup>$1</sup>")
	html = codeRe.ReplaceAllString(html, "<code>$1</code>")

	for i, block := range preBlocks {
		html = strings.Replace(html, fmt.Sprintf("\x00%d\x00", i), block, 1)
	}

	return html
}

func (r *renderer) openTag(tag string, args map[string]string) string {
	var sb strings.Builder
	sb.WriteString("<" + tag)
	if class := cssClass(tag); class != "" {
		sb.WriteString(` class="` + class + `"`)
	}
	for k, v := range args {
		sb.WriteString(" " + k + `="` + v + `"`)
	}
	sb.WriteString(">")
	return sb.String()
}

func cssClass(tag string) string {
	switch tag {
	case "div":
		return "lx-md-container"
	case "p":
		return "lx-md-paragraph"
	case "pre":
		return "lx-md-codeblock"
	case "blockquote":
		return "lx-md-blockquote"
	case "table":
		return "lx-md-table"
	case "th":
		return "lx-md-table-header"
	case "td":
		return "lx-md-table-cell"
	}
	return ""
}
