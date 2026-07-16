package md

import (
	"regexp"
	"strings"
)

var (
	allWhitespaceRe       = regexp.MustCompile(`^( |\t)+$`)
	leadingSpacesRe       = regexp.MustCompile(`^( *)`)
	leadingTabsRe         = regexp.MustCompile(`^(\t*)`)
	leadingWhitespaceRe   = regexp.MustCompile(`^( |\t)*`)
	lineSplitRe           = regexp.MustCompile(`\r\n|\r|\n`)
	blockquotePrefixRe    = regexp.MustCompile(`^> ?`)
	leading4SpacesRe      = regexp.MustCompile(`^ {4}`)
	trailingDoubleSpaceRe = regexp.MustCompile(`  $`)
	orderedNestedItemRe   = regexp.MustCompile(`^ {4}\d+\. `)
	unorderedNestedItemRe = regexp.MustCompile(`^ {4}(\*|\+|-) `)
)

// parse converts raw markdown text into a tree of blocks.
func parse(mdText string) []mdBlock {
	rawLines := lineSplitRe.Split(mdText, -1)
	lines := make([]mdLine, len(rawLines))
	for i, l := range rawLines {
		lines[i] = normalizeLine(l)
	}
	return processMap(lines)
}

// normalizeLine converts tab/space leading whitespace into a 4-space-per-level
// indent unit, and treats whitespace-only lines as empty.
func normalizeLine(raw string) mdLine {
	line := raw
	if allWhitespaceRe.MatchString(line) {
		line = ""
	}

	var indent int
	if m := leadingSpacesRe.FindStringSubmatch(line); m[1] != "" {
		indent = len(m[1]) / 4
	} else {
		m := leadingTabsRe.FindStringSubmatch(line)
		indent = len(m[1])
	}

	origin := line
	normalized := leadingWhitespaceRe.ReplaceAllString(line, strings.Repeat(" ", indent*4))

	return mdLine{line: normalized, originLine: origin, indent: indent}
}

func cutTrailingEmpty(lines []mdLine) []mdLine {
	for len(lines) > 0 && lines[len(lines)-1].line == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// processMap builds the flat block list for lines, then recursively expands
// list/blockquote blocks that carry nested indented content.
func processMap(lines []mdLine) []mdBlock {
	lines = cutTrailingEmpty(lines)
	blocks := buildBlocks(lines)

	for i := range blocks {
		blocks[i].lines = cutTrailingEmpty(blocks[i].lines)

		switch blocks[i].blockType {
		case typeUnorderedList, typeOrderedList:
			processListBlock(&blocks[i])
		case typeBlockquote:
			processBlockquoteBlock(&blocks[i])
		}
	}

	return blocks
}

func processBlockquoteBlock(b *mdBlock) {
	lines := make([]mdLine, 0, len(b.lines))
	for _, ld := range b.lines {
		stripped := blockquotePrefixRe.ReplaceAllString(ld.line, "")
		lines = append(lines, normalizeLine(stripped))
	}
	b.content = processMap(lines)
	b.lines = nil
}

// processListBlock walks a flat list block's lines and, for any item that
// owns nested indented content (a sub-list or a paragraph continuation under
// it), extracts that content and recursively parses it via processMap,
// attaching the result to the owning line.
func processListBlock(b *mdBlock) {
	lines := b.lines
	l := len(lines)
	var result []mdLine

	for i := 0; i < l; i++ {
		current := lines[i]

		if i+1 >= l {
			result = append(result, current)
			continue
		}
		next := lines[i+1]

		var farLineData *mdLine
		realFar := false

		if next.line != "" {
			itemRe, nestedItemRe := unorderedItemRe, unorderedNestedItemRe
			if b.blockType == typeOrderedList {
				itemRe, nestedItemRe = orderedItemRe, orderedNestedItemRe
			}

			if itemRe.MatchString(next.line) {
				result = append(result, current)
				continue
			}

			if nestedItemRe.MatchString(next.line) {
				nf := next
				farLineData = &nf
			} else {
				current.line = trailingDoubleSpaceRe.ReplaceAllString(current.line, "<br>")
				current.line = current.line + " " + next.line
				lines = removeAt(lines, i+1)
				l--
				lines[i] = current
				i--
				continue
			}
		}

		var j int
		if farLineData != nil {
			j = i + 1
		} else {
			if i+2 >= l {
				result = append(result, current)
				i++
				continue
			}
			f := lines[i+2]
			farLineData = &f
			j = i + 2
			realFar = true
		}

		if farLineData.indent == 0 {
			result = append(result, current)
			i++
			continue
		}

		var nestedLines []mdLine
		toDelete := map[int]bool{}
		tempLineData := *farLineData
		done := false
		for !done {
			if tempLineData.line != "" {
				tempLineData.indent--
				tempLineData.line = leading4SpacesRe.ReplaceAllString(tempLineData.line, "")
			}
			nestedLines = append(nestedLines, tempLineData)
			toDelete[j] = true
			j++
			if j == l {
				done = true
				break
			}
			tempLineData = lines[j]
			if tempLineData.indent == 0 && tempLineData.line != "" {
				done = true
			}
		}
		if realFar {
			toDelete[i+1] = true
		}

		current.content = processMap(nestedLines)
		current.contentSet = true

		newLines := make([]mdLine, 0, len(lines)-len(toDelete))
		newLines = append(newLines, lines[:i]...)
		newLines = append(newLines, current)
		for idx := i + 1; idx < len(lines); idx++ {
			if !toDelete[idx] {
				newLines = append(newLines, lines[idx])
			}
		}
		lines = newLines
		l = len(lines)
		i--
	}

	b.lines = result
}

func removeAt(s []mdLine, idx int) []mdLine {
	out := make([]mdLine, 0, len(s)-1)
	out = append(out, s[:idx]...)
	out = append(out, s[idx+1:]...)
	return out
}
