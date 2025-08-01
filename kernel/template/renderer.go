package template

import (
	"bytes"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/conv"
)

/** @interface kernel.ITemplateBulder */
type renderer struct {
	holder *holder

	namespace string

	// template file name
	name string

	// code by text
	layout  string
	content string

	params    any
	paramsMap map[string]any
}

var _ kernel.ITemplateRenderer = (*renderer)(nil)

/** @constructor */
func newTemplateRenderer(holder *holder) kernel.ITemplateRenderer {
	return &renderer{
		holder:    holder,
		paramsMap: make(map[string]any),
	}
}

func (r *renderer) SetNamespace(nmsp string) kernel.ITemplateRenderer {
	r.namespace = nmsp
	return r
}

func (r *renderer) SetTemplateName(name string) kernel.ITemplateRenderer {
	route := strings.Split(name, ":")
	if len(route) == 2 {
		r.namespace = route[0]
		r.name = route[1]
		return r
	}

	r.name = name
	return r
}

func (r *renderer) SetLayout(code string) kernel.ITemplateRenderer {
	r.layout = code
	return r
}

func (r *renderer) SetTemplate(code string) kernel.ITemplateRenderer {
	r.content = code
	return r
}

func (r *renderer) SetParams(params any) kernel.ITemplateRenderer {
	r.params = params
	return r
}

func (r *renderer) AddParam(name string, val any) kernel.ITemplateRenderer {
	r.paramsMap[name] = val
	return r
}

func (r *renderer) Namespace() string {
	return r.namespace
}

func (r *renderer) TemplateName() string {
	return r.name
}

func (r *renderer) Layout() string {
	return r.layout
}

func (r *renderer) Template() string {
	return r.content
}

func (r *renderer) Render() (string, error) {
	if r.name != "" {
		return r.renderByName()
	}

	if r.layout == "" && r.content == "" {
		return "", errors.New("nothing to render")
	}

	var tplWrapper string
	if r.namespace != "" {
		if r.holder.confParseError {
			return "", errors.New("can not parse 'Templates' config option")
		}
		scope, exists := r.holder.conf[r.namespace]
		if !exists {
			return "", fmt.Errorf("template scope namespaced by '%s' does not exist", r.namespace)
		}
		tplWrapper = scope.Layout
	} else {
		tplWrapper = "layout"
	}

	r.holder.app.Events().Trigger(kernel.EVENT_RENDERER_BEFORE_RENDER, kernel.NewData(map[string]any{
		"renderer": r,
	}))

	templates := template.New("layout")
	templates, err := templates.Parse(r.layout)
	if err != nil {
		return "", fmt.Errorf("layout parse error:%v", err)
	}

	templates, err = templates.Parse(r.content)
	if err != nil {
		return "", fmt.Errorf("content parse error:%v", err)
	}

	return r.renderTpl(templates, tplWrapper)
}

func (r *renderer) renderByName() (string, error) {
	if r.holder.confParseError {
		return "", errors.New("can not parse 'Templates' config option")
	}

	scope, exists := r.holder.conf[r.namespace]
	if !exists {
		return "", fmt.Errorf("template scope namespaced by '%s' does not exist", r.namespace)
	}

	dir := r.holder.app.Pathfinder().GetAbsPath(scope.Dir)
	tplPath := filepath.Join(dir, r.name)
	ext := filepath.Ext(tplPath)
	if ext == "" {
		tplPath += ".html"
	}

	_, err := os.Stat(tplPath)
	if err != nil {
		return "", fmt.Errorf("can no read file '%s': %v", tplPath, err)
	}

	r.holder.app.Events().Trigger(kernel.EVENT_RENDERER_BEFORE_RENDER, kernel.NewData(map[string]any{
		"renderer": r,
	}))

	tpl := r.name
	var templates *template.Template
	if scope.Layout == "" {
		templates = template.Must(template.ParseFiles(tplPath))
	} else {
		layoutPath := filepath.Join(dir, scope.Layout)
		ext := filepath.Ext(layoutPath)
		if ext == "" {
			layoutPath += ".html"
		}
		_, err := os.Stat(layoutPath)
		if err != nil {
			return "", fmt.Errorf("can no read file '%s': %v", layoutPath, err)
		}
		templates = template.Must(template.ParseFiles(layoutPath, tplPath))
		tpl = "layout"
	}

	return r.renderTpl(templates, tpl)
}

func (r *renderer) renderTpl(templates *template.Template, tplName string) (string, error) {
	var params map[string]any
	if r.params == nil {
		params = r.paramsMap
	} else {
		params = conv.ToMap(r.params)
		maps.Copy(params, r.paramsMap)
	}

	var buf bytes.Buffer
	err := templates.ExecuteTemplate(&buf, tplName, params)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
