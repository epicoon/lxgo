package compiler

import (
	"fmt"
	"slices"
	"strings"

	"github.com/epicoon/lxgo/jspp/internal/lxml/cvt"
)

type iNodeCompiler interface {
	run() string
}

/** @interface cvt.INodeCompiler */
type treeCompiler struct {
	parser   cvt.IParser
	output   string
	fCounter int
	fMap     map[string]string
	widgets  []string
	wCounter int
}

var _ cvt.ITreeCompiler = (*treeCompiler)(nil)

/** @constructor */
func NewTreeCompiler(parser cvt.IParser) cvt.ITreeCompiler {
	return &treeCompiler{
		parser:  parser,
		fMap:    make(map[string]string, 0),
		widgets: make([]string, 0),
	}
}

func (c *treeCompiler) SetOutput(out string) cvt.ITreeCompiler {
	c.output = out
	return c
}

func (c *treeCompiler) Run() string {
	tree := c.parser.Tree()
	if tree == nil {
		c.parser.AddError("tree is not built")
		return ""
	}

	var codeStart string
	if c.output != "" {
		f := "function _out(key,name){if(!(key in __out__)){__out__[key]=name;}else{" +
			"if(!lx.isArray(__out__[key]))__out__[key]=[__out__[key]];" +
			"__out__[key].push(name)}}"
		codeStart = fmt.Sprintf("const %s=(()=>{let __out__={};%s", c.output, f)
	} else {
		codeStart = "(()=>{"
	}

	code := codeStart
	tree.EachBlock(func(n cvt.INode) {
		bc, err := newBlockCompiler(c, n)
		if err != nil {
			c.parser.AddError(err.Error())
			return
		}
		fName := fmt.Sprintf("_f%d", c.fCounter)
		c.fCounter++
		c.fMap[bc.node.Name] = fName
		bc.fName = fName
		code += bc.run()
	})
	if c.parser.HasError() {
		return ""
	}

	tree.EachRoot(func(n cvt.INode) {
		nc, err := c.newNodeCompiler(n)
		if err != nil {
			c.parser.AddError(err.Error())
			return
		}
		code += nc.run()
	})

	for key, name := range c.fMap {
		code = strings.ReplaceAll(code, "[|"+key+"|]", name)
	}

	if c.output == "" {
		code += "})();"
	} else {
		code += "return __out__;})();"
	}

	widgets := ""
	for _, w := range c.widgets {
		widgets += "@lx:use " + w + ";"
	}

	return widgets + code
}

func (c *treeCompiler) compileContent(node cvt.INode) string {
	code := ""
	node.EachNested(func(n cvt.INode) {
		code += c.compileNode(n)
	})
	return code
}

func (c *treeCompiler) compileNode(node cvt.INode) string {
	nc, err := c.newNodeCompiler(node)
	if err != nil {
		c.parser.AddError(err.Error())
		return ""
	}

	return nc.run()
}

func (c *treeCompiler) registerWidget(name string) {
	if name == "lx.Rect" || name == "lx.Box" {
		return
	}
	if !slices.Contains(c.widgets, name) {
		c.widgets = append(c.widgets, name)
	}
}

func (c *treeCompiler) newNodeCompiler(node cvt.INode) (iNodeCompiler, error) {
	switch node.Type() {
	case cvt.NodeTypeWidget:
		return newWidgetCompiler(c, node)
	case cvt.NodeTypeSyntax:
		return newSyntaxCompiler(c, node)
	case cvt.NodeTypeBlock:
		return newBlockCompiler(c, node)
	}
	return nil, fmt.Errorf("unknown node type: %v", node.Type())
}
