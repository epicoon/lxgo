package template

import (
	"fmt"
	"path/filepath"

	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/config"
	"github.com/epicoon/lxgo/kernel/conv"
)

/** @interface kernel.ITemplateHolder */
type holder struct {
	app            kernel.IApp
	conf           map[string]tplScope
	confParseError bool
}

type tplScope struct {
	Dir    string
	Layout string
}

var _ kernel.ITemplateHolder = (*holder)(nil)

/** @constructor */
func NewTemplateHolder(app kernel.IApp) *holder {
	return &holder{
		app:            app,
		confParseError: false,
	}
}

func (h *holder) Layout(nmsp string) string {
	if err := h.promiseConf(); err != nil {
		h.confParseError = true
		return ""
	}

	scope, exists := h.conf[nmsp]
	if !exists {
		h.app.LogError(fmt.Sprintf("unknown templates namespace: %s", nmsp), "TemplateRendering")
		return ""
	}

	return scope.Layout
}

func (h *holder) LayoutPath(nmsp string) string {
	l := h.Layout(nmsp)
	if l == "" {
		return ""
	}

	ext := filepath.Ext(l)
	if ext == "" {
		l += ".html"
	}

	scope := h.conf[nmsp]

	return h.app.Pathfinder().GetAbsPath(filepath.Join(scope.Dir, l))
}

func (h *holder) TemplateRenderer() kernel.ITemplateRenderer {
	if err := h.promiseConf(); err != nil {
		h.confParseError = true
	}
	return newTemplateRenderer(h)
}

func (h *holder) promiseConf() error {
	if h.conf != nil || h.confParseError {
		return nil
	}

	conf := h.app.Config()
	if !config.HasParam(conf, "Templates") {
		h.conf = make(map[string]tplScope, 1)
		h.conf[""] = tplScope{
			Dir:    "",
			Layout: "",
		}
		return nil
	}

	raw, err := config.GetParam[[]any](conf, "Templates")
	if err != nil {
		h.app.LogError(fmt.Sprintf("wrong app config for option 'Templates': %v", err), "TemplateRendering")
		return err
	}

	h.conf = make(map[string]tplScope, len(raw))
	for _, rawVal := range raw {
		dict := kernel.Dict(rawVal.(kernel.Config))
		info := struct {
			Dir       string
			Namespace string
			Layout    string
		}{}
		err := conv.DictToStruct(&dict, &info)
		if err != nil {
			h.app.LogError(fmt.Sprintf("wrong app config for option 'Templates': %v", err), "TemplateRendering")
			return err
		}

		h.conf[info.Namespace] = tplScope{
			Dir:    info.Dir,
			Layout: info.Layout,
		}
	}

	return nil
}
