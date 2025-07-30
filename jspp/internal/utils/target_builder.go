package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/epicoon/lxgo/jspp"
	"github.com/epicoon/lxgo/jspp/internal/base"
	"github.com/epicoon/lxgo/kernel/conv"
)

type targetBuilder struct {
	pp     jspp.IPreprocessor
	path   string
	lang   string
	target *base.Target
}

/** @constructor */
func NewTargetBuilder(pp jspp.IPreprocessor, path, lang string) *targetBuilder {
	return &targetBuilder{
		pp:   pp,
		path: path,
		lang: lang,
	}
}

func (tb *targetBuilder) Build() {
	//TODO cache

	tb.target = tb.defineTarget()
	if tb.target == nil {
		return
	}

	c := tb.prepareCompiler()
	code, err := c.Run()
	if err != nil {
		tb.pp.LogError("Can not build js-bundle '%s' from js-file '%s': %v", tb.path, tb.target.EntryPoint, err)
		return
	}

	if tb.target.Type == "app" && tb.pp.Config().CssScopeRenderSide == "server" {
		code = tb.renderCss(c) + code
	}

	tb.genOutput(code)
}

func (tb *targetBuilder) renderCss(c jspp.ICompiler) string {
	sCode := ""
	for _, m := range c.CompiledModules() {
		sCode += "@lx:use " + m + ";"
	}

	pCode := `return {css: lx.app.cssManager.pack()};`
	res := struct {
		Css []any `dict:"css"`
	}{}

	sc := tb.pp.CompilerBuilder().
		SetLang(tb.lang).
		SetServerContext().
		SetAppContext().
		SetUnwrapped().
		SetCode(sCode).
		SetPostCode(pCode).
		Compiler()
	code, err := sc.Run()
	if err != nil {
		tb.pp.LogError("Can not build css-renderer code for js-file '%s': %v", tb.path, err)
		return ""
	}

	executor := tb.pp.ExecutorBuilder().
		SetCode(code).
		Executor()
	rawRes, err := executor.Exec()
	if err != nil {
		tb.pp.LogError("can not run server-side css-renderer for '%s': %v", tb.path, err)
		return ""
	}
	if rawRes.Fatal() != "" {
		tb.pp.LogError("can not run server-side css-renderer for '%s':\n%v", tb.path, rawRes.Fatal())
		return ""
	}

	conv.MapToStruct(rawRes.Result().(map[string]any), &res)
	css, err := json.Marshal(res.Css)
	if err != nil {
		tb.pp.LogError("can not encode css for '%s': %v", tb.path, err)
		return ""
	}

	return fmt.Sprintf("lx.app._css=%s;", string(css))
}

func (tb *targetBuilder) prepareCompiler() jspp.ICompiler {
	cb := tb.pp.CompilerBuilder().
		SetLang(tb.lang).
		SetClientContext().
		SetFilePath(tb.target.EntryPoint)
	if tb.target.Type == "app" {
		cb.SetAppContext()
	}
	return cb.Compiler()
}

func (tb *targetBuilder) genOutput(code string) {
	dest := tb.pp.App().Pathfinder().GetAbsPath(tb.target.Output)
	err := os.WriteFile(dest, []byte(code), 0644)
	if err != nil {
		tb.pp.LogError("Can not write js-bundle '%s' to js-file '%s': %v", tb.target.EntryPoint, dest, err)
	}
}

func (tb *targetBuilder) defineTarget() *base.Target {
	targets := tb.pp.Config().Targets
	for _, iTarget := range targets {
		if strings.HasSuffix(tb.path, iTarget.Output) {
			return &iTarget
		}
	}
	return nil
}
