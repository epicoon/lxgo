package cvt

type IParser interface {
	SetOutput(out string) IParser
	ParseFile(path string) (string, error)
	ParseText(text string) (string, error)

	//TODO move somewhere
	AddError(msg string)
	HasError() bool
	Texts() []string
	Widgets() []string
	Tree() ITree
}

type INodeParser interface {
	Run(head, line string) INode
}

type ITreeCompiler interface {
	SetOutput(out string) ITreeCompiler
	Run() string
}

type INodeRenderer interface {
	Run() string
}

type ITree interface {
	AddNode(n INode)
	EachBlock(func(n INode))
	EachRoot(func(n INode))
}

const (
	NodeTypeWidget = iota + 1
	NodeTypeBlock
	NodeTypeSyntax
	NodeTypeHtml
)

type INode interface {
	Type() int
	Is(tp int) bool
	Depth() int
	AddNode(n INode)
	IsEmpty() bool
	Nested() []INode
	EachNested(func(n INode))

	//TODO move somewhere
	AddHTML(html string)
	HTML() string
}
