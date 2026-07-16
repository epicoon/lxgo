package md

import "regexp"

// parseData is per-run scratch state for the block classifier.
type parseData struct {
	line         string
	indent       int
	spaces       int
	spacesBefore int
	blockLines   []mdLine
	codeType     string
}

func (p *parseData) incSpaces() {
	p.spacesBefore = p.spaces
	p.spaces++
}

func (p *parseData) dropSpaces() {
	p.spacesBefore = p.spaces
	p.spaces = 0
}

// blocksBuilder classifies a flat run of lines into blocks, one line at a time.
type blocksBuilder struct {
	typeDefined bool
	prevType    BlockType
	currentType BlockType

	// inFence tracks whether we're between an opening and closing fence
	// marker of a typed code block, independent of currentType — a blank
	// line inside the fence must not end the block the way it ends every
	// other block type, but the block still needs to end on the first blank
	// line after the closing marker.
	inFence bool
}

// buildBlocks classifies lines into a flat (non-recursive) list of blocks.
func buildBlocks(lines []mdLine) []mdBlock {
	b := &blocksBuilder{prevType: typeNone, currentType: typeNone}
	return b.run(lines)
}

func (b *blocksBuilder) run(lines []mdLine) []mdBlock {
	var result []mdBlock
	pd := &parseData{}

	flush := func() {
		result = append(result, mdBlock{
			blockType: b.prevType,
			lines:     pd.blockLines,
			codeType:  pd.codeType,
		})
		pd.blockLines = nil
		pd.codeType = ""
	}

	for _, ld := range lines {
		b.typeDefined = false
		pd.line = ld.line

		if pd.line == "" {
			if b.isCurrentType(typeNone) {
				continue
			}
			pd.incSpaces()
			if b.isCurrentType(typeCodeBlock) || b.inFence ||
				(pd.spaces == 1 && b.isCurrentType(typeUnorderedList, typeOrderedList)) {
				b.defineType(b.currentType)
			} else {
				pd.dropSpaces()
				b.defineType(typeNone)
			}
		} else {
			pd.dropSpaces()
		}
		pd.indent = ld.indent

		for _, check := range blockCheckers {
			if b.typeDefined {
				break
			}
			check(b, pd)
		}

		if b.typeIsChanged() {
			flush()
		}
		if !b.isCurrentType(typeNone) {
			pd.blockLines = append(pd.blockLines, ld)
		}
	}

	if len(pd.blockLines) > 0 {
		result = append(result, mdBlock{
			blockType: b.currentType,
			lines:     pd.blockLines,
			codeType:  pd.codeType,
		})
	}

	return result
}

func (b *blocksBuilder) isCurrentType(types ...BlockType) bool {
	for _, t := range types {
		if b.currentType == t {
			return true
		}
	}
	return false
}

func (b *blocksBuilder) defineType(t BlockType) {
	b.prevType = b.currentType
	b.currentType = t
	b.typeDefined = true
}

func (b *blocksBuilder) typeIsChanged() bool {
	return b.currentType != b.prevType && b.prevType != typeNone
}

var (
	lineRe             = regexp.MustCompile(`^(---+|\*\*\*+)\s*$`)
	titleRe            = regexp.MustCompile(`^(#{1,6})`)
	titleDashUnderline = regexp.MustCompile(`^-+\s*$`)
	titleEqUnderline   = regexp.MustCompile(`^=+\s*$`)
	tableRe            = regexp.MustCompile(`^\|.+?\|\s*$`)
	codeFenceRe        = regexp.MustCompile("^(```|~~~)(\\w+)?")
	orderedItemRe      = regexp.MustCompile(`^\d+\. `)
	unorderedItemRe    = regexp.MustCompile(`^(\*|\+|-) `)
)

func checkLine(b *blocksBuilder, pd *parseData) {
	if lineRe.MatchString(pd.line) {
		b.defineType(typeLine)
	}
}

func checkTitle(b *blocksBuilder, pd *parseData) {
	if m := titleRe.FindStringSubmatch(pd.line); m != nil {
		switch len(m[1]) {
		case 1:
			b.defineType(typeTitle1)
		case 2:
			b.defineType(typeTitle2)
		case 3:
			b.defineType(typeTitle3)
		case 4:
			b.defineType(typeTitle4)
		case 5:
			b.defineType(typeTitle5)
		case 6:
			b.defineType(typeTitle6)
		}
	} else if pd.indent == 0 && len(pd.blockLines) == 1 && titleDashUnderline.MatchString(pd.line) {
		// Defining the type twice makes prevType equal currentType, so this
		// line merges into the preceding paragraph line instead of flushing
		// it as a separate block first (see typeIsChanged).
		b.defineType(typeTitle1)
		b.defineType(typeTitle1)
	} else if pd.indent == 0 && len(pd.blockLines) == 1 && titleEqUnderline.MatchString(pd.line) {
		b.defineType(typeTitle2)
		b.defineType(typeTitle2)
	}
}

func checkTable(b *blocksBuilder, pd *parseData) {
	if tableRe.MatchString(pd.line) {
		b.defineType(typeTable)
	}
}

func checkCodeBlock(b *blocksBuilder, pd *parseData) {
	if pd.indent != 0 && b.isCurrentType(
		typeNone, typeCodeBlock,
		typeTitle1, typeTitle2, typeTitle3, typeTitle4, typeTitle5, typeTitle6,
	) {
		b.defineType(typeCodeBlock)
	}
}

func checkCodeBlockTyped(b *blocksBuilder, pd *parseData) {
	m := codeFenceRe.FindStringSubmatch(pd.line)
	if m == nil {
		if b.inFence {
			b.defineType(typeCodeBlockTyped)
		}
		return
	}
	if b.inFence {
		// Closing marker: stop tolerating blank lines, but still include
		// this line in the block.
		b.inFence = false
		b.defineType(typeCodeBlockTyped)
		return
	}
	if m[2] != "" {
		pd.codeType = m[2]
	}
	b.inFence = true
	b.defineType(typeCodeBlockTyped)
}

func checkBlockquote(b *blocksBuilder, pd *parseData) {
	if (len(pd.line) > 0 && pd.line[0] == '>') || (pd.line != "" && b.isCurrentType(typeBlockquote)) {
		b.defineType(typeBlockquote)
	}
}

func checkOrderedList(b *blocksBuilder, pd *parseData) {
	if orderedItemRe.MatchString(pd.line) ||
		(b.isCurrentType(typeOrderedList) && (pd.spaces == 1 ||
			(pd.line != "" && pd.indent == 0 && pd.spacesBefore == 0) ||
			(pd.line != "" && pd.indent != 0))) {
		b.defineType(typeOrderedList)
	}
}

func checkUnorderedList(b *blocksBuilder, pd *parseData) {
	if unorderedItemRe.MatchString(pd.line) ||
		(b.isCurrentType(typeUnorderedList) && (pd.spaces == 1 ||
			(pd.line != "" && pd.indent == 0 && pd.spacesBefore == 0) ||
			(pd.line != "" && pd.indent != 0))) {
		b.defineType(typeUnorderedList)
	}
}

func checkParagraph(b *blocksBuilder, pd *parseData) {
	b.defineType(typeParagraph)
}

// checkTitle must run before checkLine: a dash-underline title (`^-+\s*$`) is
// a subset of the hr pattern (`^(---+|\*\*\*+)\s*$`) for 3+ dashes, so if
// checkLine ran first a title underlined with the common 3-dash style would
// always be swallowed as an <hr> instead — only 1-2 dash underlines could
// ever reach checkTitle's dash branch. The len(blockLines)==1 guard in
// checkTitle means this ordering doesn't affect standalone `---`/`***`
// thematic breaks (those have no preceding paragraph line accumulated in the
// current block).
var blockCheckers = []func(*blocksBuilder, *parseData){
	checkTitle, checkLine, checkTable, checkCodeBlock, checkCodeBlockTyped,
	checkBlockquote, checkOrderedList, checkUnorderedList, checkParagraph,
}
