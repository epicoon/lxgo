package compiler

import (
	"github.com/epicoon/lxgo/jspp"
	"github.com/epicoon/lxgo/jspp/internal/base"
	"github.com/epicoon/lxgo/kernel"
)

type compilerBuilder struct {
	compiler Compiler
}

/** @interface */
var _ jspp.ICompilerBuilder = (*compilerBuilder)(nil)

/** @constructor */
func Builder() *compilerBuilder {
	return &compilerBuilder{compiler: *newCompiler()}
}

func (cb *compilerBuilder) Compiler() jspp.ICompiler {
	return &cb.compiler
}

func (cb *compilerBuilder) SetPreprocessor(pp jspp.IPreprocessor) jspp.ICompilerBuilder {
	cb.compiler.pp = pp
	cb.compiler.app = pp.App()
	cb.compiler.config = pp.Config()
	return cb
}

func (cb *compilerBuilder) SetApp(app kernel.IApp) jspp.ICompilerBuilder {
	cb.compiler.app = app
	return cb
}

func (cb *compilerBuilder) SetConfig(c *base.JSPreprocessorConfig) jspp.ICompilerBuilder {
	cb.compiler.config = c
	return cb
}

func (cb *compilerBuilder) SetPathfinder(pf kernel.IPathfinder) jspp.ICompilerBuilder {
	cb.compiler.pathfinder = pf
	return cb
}

func (cb *compilerBuilder) SetLang(lang string) jspp.ICompilerBuilder {
	cb.compiler.lang = lang
	return cb
}

func (cb *compilerBuilder) SetI18n(i18n jspp.II18nMap) jspp.ICompilerBuilder {
	cb.compiler.i18n = i18n
	return cb
}

func (cb *compilerBuilder) SetAppContext() jspp.ICompilerBuilder {
	cb.compiler.isApp = true
	return cb
}

func (cb *compilerBuilder) SetClientContext() jspp.ICompilerBuilder {
	cb.compiler.context = contextClient
	return cb
}

func (cb *compilerBuilder) SetServerContext() jspp.ICompilerBuilder {
	cb.compiler.context = contextServer
	return cb
}

func (cb *compilerBuilder) SetFilePath(filePath string) jspp.ICompilerBuilder {
	pf := cb.compiler.Pathfinder()
	if pf != nil {
		filePath = pf.GetAbsPath(filePath)
	}
	cb.compiler.filePath = filePath
	return cb
}

func (cb *compilerBuilder) SetCompiledModules(list []string) jspp.ICompilerBuilder {
	cb.compiler.compiledModules = list
	return cb
}

func (cb *compilerBuilder) SetPrevCode(code string) jspp.ICompilerBuilder {
	cb.compiler.prevCode = code
	return cb
}

func (cb *compilerBuilder) SetCode(code string) jspp.ICompilerBuilder {
	cb.compiler.inputCode = code
	return cb
}

func (cb *compilerBuilder) SetPostCode(code string) jspp.ICompilerBuilder {
	cb.compiler.postCode = code
	return cb
}

func (cb *compilerBuilder) UseModules(modules []string) jspp.ICompilerBuilder {
	cb.compiler.useModules = modules
	return cb
}

func (cb *compilerBuilder) SetUnwrapped() jspp.ICompilerBuilder {
	cb.compiler.flags.Unwrapped = true
	return cb
}

func (cb *compilerBuilder) BuildModules(val bool) jspp.ICompilerBuilder {
	cb.compiler.buildModules = val
	return cb
}

func (cb *compilerBuilder) IgnoreModules(list []string) jspp.ICompilerBuilder {
	cb.compiler.ignoredModules = list
	return cb
}
