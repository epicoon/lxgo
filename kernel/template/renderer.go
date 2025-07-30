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
	holder    *holder
	namespace string
	useNmsp   bool
	name      string

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
		useNmsp:   false,
		paramsMap: make(map[string]any),
	}
}

func (r *renderer) SetNamespace(nmsp string) kernel.ITemplateRenderer {
	r.namespace = nmsp
	r.useNmsp = true
	return r
}

func (r *renderer) SetName(name string) kernel.ITemplateRenderer {
	r.name = name
	return r
}

func (r *renderer) SetLayout(code string) kernel.ITemplateRenderer {
	r.layout = code
	return r
}

func (r *renderer) SetContent(code string) kernel.ITemplateRenderer {
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

func (r *renderer) Render() (string, error) {
	if r.name != "" {
		return r.renderByName()
	}

	if r.layout == "" && r.content == "" {
		return "", errors.New("nothing to render")
	}

	templates := template.New("layout")
	templates, err := templates.Parse(r.layout)
	if err != nil {
		return "", fmt.Errorf("layout parse error:%v", err)
	}

	templates, err = templates.Parse(r.content)
	if err != nil {
		return "", fmt.Errorf("content parse error:%v", err)
	}

	var tpl string
	if r.useNmsp {
		if r.holder.confParseError {
			return "", errors.New("can not parse 'Templates' config option")
		}
		scope, exists := r.holder.conf[r.namespace]
		if !exists {
			return "", fmt.Errorf("template scope namespaced by '%s' does not exist", r.namespace)
		}
		tpl = scope.Layout
		r.holder.app.Events().Trigger(kernel.EVENT_RENDERER_BEFORE_RENDER, kernel.NewData(map[string]any{
			"Namespace": r.namespace,
			"Layout":    scope.Layout,
			"Renderer":  r,
		}))
	} else {
		tpl = "layout"
	}

	return r.renderTpl(templates, tpl)
}

func (r *renderer) renderByName() (string, error) {
	if r.holder.confParseError {
		return "", errors.New("can not parse 'Templates' config option")
	}

	var nmsp, tpl string
	if r.useNmsp {
		nmsp = r.namespace
		tpl = r.name
	} else {
		route := strings.Split(r.name, ".")
		if len(route) > 2 {
			return "", fmt.Errorf("wrong template key: '%s', only one dot available", r.name)
		}
		if len(route) == 1 {
			nmsp = ""
			tpl = route[0]
		} else {
			nmsp = route[0]
			tpl = route[1]
		}
	}

	scope, exists := r.holder.conf[nmsp]
	if !exists {
		return "", fmt.Errorf("template scope namespaced by '%s' does not exist", nmsp)
	}

	dir := r.holder.app.Pathfinder().GetAbsPath(scope.Dir)
	tplPath := filepath.Join(dir, tpl)
	ext := filepath.Ext(tplPath)
	if ext == "" {
		tplPath += ".html"
	}

	_, err := os.Stat(tplPath)
	if err != nil {
		return "", fmt.Errorf("can no read file '%s': %v", tplPath, err)
	}

	r.holder.app.Events().Trigger(kernel.EVENT_RENDERER_BEFORE_RENDER, kernel.NewData(map[string]any{
		"Namespace": nmsp,
		"Layout":    scope.Layout,
		"Renderer":  r,
	}))

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
	params := conv.ToMap(r.params)
	maps.Copy(params, r.paramsMap)

	var buf bytes.Buffer
	err := templates.ExecuteTemplate(&buf, tplName, params)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
