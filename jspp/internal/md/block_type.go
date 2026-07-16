package md

// BlockType is the classification of a markdown block.
type BlockType string

const (
	typeNone           BlockType = "none"
	typeLine           BlockType = "line"
	typeTitle1         BlockType = "title1"
	typeTitle2         BlockType = "title2"
	typeTitle3         BlockType = "title3"
	typeTitle4         BlockType = "title4"
	typeTitle5         BlockType = "title5"
	typeTitle6         BlockType = "title6"
	typeParagraph      BlockType = "paragraph"
	typeCodeBlock      BlockType = "codeBlock"
	typeCodeBlockTyped BlockType = "codeBlockTyped"
	typeBlockquote     BlockType = "blockquote"
	typeOrderedList    BlockType = "orderedList"
	typeUnorderedList  BlockType = "unorderedList"
	typeTable          BlockType = "table"
)

// mdLine is one normalized line of source text.
type mdLine struct {
	line       string
	originLine string
	indent     int

	// content holds nested blocks parsed from indented text under this
	// specific line, when it is a list item with its own sub-content.
	content    []mdBlock
	contentSet bool
}

// mdBlock is a contiguous run of lines classified as one block type.
type mdBlock struct {
	blockType BlockType
	lines     []mdLine

	// content replaces lines for blockquote blocks (nested, recursively parsed).
	content []mdBlock

	// codeType holds the fence language tag for typed code blocks, e.g. "js".
	codeType string
}
