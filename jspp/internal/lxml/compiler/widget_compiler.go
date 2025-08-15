package compiler

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/epicoon/lxgo/jspp/internal/lxml/cvt"
	"github.com/epicoon/lxgo/jspp/internal/lxml/tree"
)

/** @interface cvt.INodeCompiler */
type widgetCompiler struct {
	tc   *treeCompiler
	node *tree.WidgetNode
}

var _ iNodeCompiler = (*widgetCompiler)(nil)

/** @constructor */
func newWidgetCompiler(tc *treeCompiler, node cvt.INode) (*widgetCompiler, error) {
	n, ok := node.(*tree.WidgetNode)
	if !ok {
		return nil, errors.New("wrong node type: *tree.WidgetNode expected")
	}

	return &widgetCompiler{
		tc:   tc,
		node: n,
	}, nil
}

func (c *widgetCompiler) run() string {
	c.tc.registerWidget(c.node.Widget)

	varName := fmt.Sprintf("_w%d", c.tc.wCounter)
	c.tc.wCounter++

	nodeCode := c.compile(varName)
	if c.tc.output != "" {
		nodeCode += c.getOutString(varName)
	}

	if c.node.IsEmpty() {
		return nodeCode
	}

	if c.node.HTML() != "" {
		c.tc.parser.AddError("element can not have HTML and widgets at the same time")
		return ""
	}

	if c.node.IsMatrix {
		nested := c.node.Nested()
		if len(nested) != 1 {
			c.tc.parser.AddError("matrix mast contain exactly one child")
			return ""
		}
		elem, ok := nested[0].(*tree.WidgetNode)
		if !ok {
			c.tc.parser.AddError("matrix mast contain exactly widget child")
			return ""
		}
		cElem, err := newWidgetCompiler(c.tc, elem)
		if err != nil {
			c.tc.parser.AddError(err.Error())
			return ""
		}

		conf := cElem.getConfigCode()
		if conf == "" {
			conf = "{}"
		}
		nodeCode += varName + ".setMatrixItemBox([" + elem.Widget + "," + conf + "]);"

		methods := cElem.getActionsCode("box")
		if methods != "" || !elem.IsEmpty() {
			out := c.tc.output
			c.tc.output = ""
			nodeCode += varName + ".on('renderMatrixItem',e=>{const box=e.box;" + methods +
				c.tc.compileContent(elem) + "});"
			c.tc.output = out
		}
	} else {
		if c.node.Depth() == 0 {
			nodeCode += varName + ".useRenderCache();"
		}
		nodeCode += varName + ".begin();"
		nodeCode += c.tc.compileContent(c.node)
		nodeCode += varName + ".end();"
		if c.node.Depth() == 0 {
			nodeCode += varName + ".applyRenderCache();"
		}
	}

	return nodeCode
}

func (c *widgetCompiler) compile(varName string) string {
	code := c.getInitCode(varName) + c.getActionsCode(varName)
	return procInserts(code)
}

func (c *widgetCompiler) getOutString(varName string) string {
	var key string
	if c.node.Field != "" {
		key = c.node.Field
	}
	if c.node.Key != "" && c.node.Key != c.node.Field {
		key = c.node.Key
	}
	if key == "" {
		return ""
	}
	return fmt.Sprintf("_out('%s',%s);", key, varName)
}

func (c *widgetCompiler) getInitCode(varName string) string {
	n := c.node

	if n.Field != "" || n.Key != "" || n.Data != "" || len(n.Methods) != 0 || !n.IsEmpty() {
		return fmt.Sprintf("var %s=new %s({%s});", varName, n.Widget, c.getConfigCode())
	}

	return fmt.Sprintf("new %s({%s});", n.Widget, c.getConfigCode())
}

func (c *widgetCompiler) getConfigCode() string {
	n := c.node

	html := n.HTML()

	if n.Key == "" && n.Field == "" && !n.IsVolume && len(n.Css) == 0 &&
		n.Config == "" && len(n.Geom) == 0 && n.Text == "" && html == "" {
		return ""
	}

	config := make([]string, 0)

	if n.Key != "" {
		config = append(config, "key:\""+n.Key+"\"")
	}
	if n.Field != "" {
		if n.IsMatrix {
			config = append(config, "matrix:\""+n.Field+"\"")
		} else {
			config = append(config, "field:\""+n.Field+"\"")
		}
	}

	if n.IsVolume {
		config = append(config, "geom:true")
	}

	if len(n.Css) > 0 {
		css := make([]string, 0, len(n.Css))
		for _, item := range n.Css {
			css = append(css, "\""+item+"\"")
		}
		config = append(config, fmt.Sprintf("css:[%s]", strings.Join(css, ",")))
	}

	if n.Data != "" {
		config = append(config, "data:{"+n.Data+"}")
	}

	if n.Text != "" {
		config = append(config, "text:\""+n.Text+"\"")
	}

	if len(n.Geom) > 0 {
		geom := make([]string, 0, 4)
		for i, item := range n.Geom {
			if item == "null" {
				if i > 3 {
					continue
				}
				geom = append(geom, "null")
			} else if isNum(item) {
				geom = append(geom, item)
			} else {
				geom = append(geom, "\""+item+"\"")
			}
		}
		config = append(config, fmt.Sprintf("geom:[%s]", strings.Join(geom, ",")))
	}

	if html != "" {
		config = append(config, "html:`"+html+"`")
	}

	sConfig := strings.Join(config, ",")
	if n.Config != "" {
		if sConfig != "" {
			sConfig += ","
		}
		sConfig += n.Config
	}

	return sConfig
}

func (c *widgetCompiler) getActionsCode(varName string) string {
	n := c.node
	if len(n.Methods) == 0 {
		return ""
	}

	code := ""
	for _, method := range n.MethodsSeq {
		args := n.Methods[method]
		if args != "" {
			re := regexp.MustCompile(`^[\w\d_]+?:`)
			if re.MatchString(args) {
				args = "{" + args + "}"
			}
		}
		code += varName + "." + method + "(" + args + ");"
	}
	return code
}

func isNum(s string) bool {
	re := regexp.MustCompile(`^(?:\d+|\d+\.\d+)$`)
	return re.MatchString(s)
}

func procInserts(str string) string {
	// ${text} -> " + text + "
	re := regexp.MustCompile(`\$\{([^}]+?)\}(")?`)
	return re.ReplaceAllStringFunc(str, func(s string) string {
		match := re.FindStringSubmatch(s)
		if len(match) != 3 {
			return s
		}
		if match[2] == "\"" {
			return "\"+" + match[1]
		}
		return "\"+" + match[1] + "+\""
	})
}
