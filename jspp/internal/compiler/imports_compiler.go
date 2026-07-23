package compiler

import (
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/epicoon/lxgo/jspp/internal/utils"
)

const importMarker = "lx.import("

// importPathArg is one path argument parsed out of an lx.import(...) call,
// together with whichever -R/-F/-U flags apply to it (its own glued flag,
// merged with any call-wide standalone flag argument).
type importPathArg struct {
	Path  string
	Flags Flags
}

// importCall is one lx.import(...) occurrence found in a source file: its
// span in the original text (for in-place replacement with the compiled
// path code) plus its parsed path and module arguments.
type importCall struct {
	start, end int
	paths      []importPathArg
	modules    []string
}

var importFlagOnlyRe = regexp.MustCompile(`^-[A-Za-z]+$`)
var importLeadingFlagRe = regexp.MustCompile(`^-([A-Za-z]+)\s+`)
var importTrailingFlagRe = regexp.MustCompile(`\s+-([A-Za-z]+)$`)

// findImportCalls scans code for lx.import(...) calls (top level, not
// nested inside another lx.import) and parses each one's argument list.
func findImportCalls(code string) []importCall {
	var calls []importCall
	n := len(code)
	i := 0
	for i < n {
		rel := strings.Index(code[i:], importMarker)
		if rel == -1 {
			break
		}
		idx := i + rel

		if idx > 0 && isLxmlIdentByte(code[idx-1]) {
			i = idx + 1
			continue
		}

		openParen := idx + len(importMarker) - 1
		closeParen := utils.FindMatchingBrace(code, openParen, '(')
		if closeParen == -1 {
			i = idx + len(importMarker)
			continue
		}

		argsText := code[openParen+1 : closeParen]
		rawArgs := splitImportArgs(argsText)

		callFlags := Flags{}
		var paths []importPathArg
		var modules []string
		for _, raw := range rawArgs {
			pathArg, module, standaloneFlag, isPath := parseImportArg(raw)
			switch {
			case standaloneFlag != nil:
				mergeFlags(&callFlags, *standaloneFlag)
			case isPath:
				paths = append(paths, *pathArg)
			case module != "":
				modules = append(modules, module)
			}
		}
		for k := range paths {
			mergeFlags(&paths[k].Flags, callFlags)
		}

		calls = append(calls, importCall{
			start:   idx,
			end:     closeParen + 1,
			paths:   paths,
			modules: modules,
		})

		i = closeParen + 1
	}

	return calls
}

// splitImportArgs splits an lx.import(...) call's raw argument list text
// on top-level commas - commas inside a quoted path or nested brackets don't
// split.
func splitImportArgs(argsText string) []string {
	var args []string
	depth := 0
	var quote byte
	start := 0
	n := len(argsText)
	for i := 0; i < n; i++ {
		c := argsText[i]
		switch {
		case quote != 0:
			if c == quote {
				bs := 0
				for k := i - 1; k >= start && argsText[k] == '\\'; k-- {
					bs++
				}
				if bs%2 == 0 {
					quote = 0
				}
			}
		case c == '\'' || c == '"':
			quote = c
		case c == '(' || c == '[' || c == '{':
			depth++
		case c == ')' || c == ']' || c == '}':
			depth--
		case c == ',' && depth == 0:
			args = append(args, strings.TrimSpace(argsText[start:i]))
			start = i + 1
		}
	}
	if last := strings.TrimSpace(argsText[start:]); last != "" {
		args = append(args, last)
	}
	return args
}

// parseImportArg classifies one raw (already comma-split, trimmed)
// argument: a quoted argument is a path (isPath=true, with its own glued
// flags already stripped and merged into pathArg.Flags) unless, once any
// glued flags are stripped, nothing is left - then it's a standalone flag
// argument (standaloneFlag != nil) that applies to every path in the same
// call. An unquoted argument is a bare module reference (module != "").
func parseImportArg(raw string) (pathArg *importPathArg, module string, standaloneFlag *Flags, isPath bool) {
	if raw == "" {
		return nil, "", nil, false
	}

	quote := raw[0]
	if quote != '\'' && quote != '"' {
		return nil, raw, nil, false
	}
	if len(raw) < 2 || raw[len(raw)-1] != quote {
		// malformed (unterminated quote) - best-effort: treat the rest as a path
		return &importPathArg{Path: raw[1:]}, "", nil, true
	}

	inner := raw[1 : len(raw)-1]
	inner = strings.ReplaceAll(inner, "\\"+string(quote), string(quote))

	if importFlagOnlyRe.MatchString(inner) {
		f := Flags{}
		applyFlagLetters(&f, inner[1:])
		return nil, "", &f, false
	}

	flags := Flags{}
	for {
		changed := false
		if m := importLeadingFlagRe.FindStringSubmatch(inner); m != nil {
			applyFlagLetters(&flags, m[1])
			inner = inner[len(m[0]):]
			changed = true
		}
		if m := importTrailingFlagRe.FindStringSubmatch(inner); m != nil {
			applyFlagLetters(&flags, m[1])
			inner = inner[:len(inner)-len(m[0])]
			changed = true
		}
		if !changed {
			break
		}
	}

	if inner == "" {
		return nil, "", &flags, false
	}

	return &importPathArg{Path: inner, Flags: flags}, "", nil, true
}

func applyFlagLetters(flags *Flags, letters string) {
	if strings.Contains(letters, "R") {
		flags.Recursive = true
	}
	if strings.Contains(letters, "F") {
		flags.Force = true
	}
	if strings.Contains(letters, "U") {
		flags.Unwrapped = true
	}
}

func mergeFlags(dst *Flags, src Flags) {
	dst.Recursive = dst.Recursive || src.Recursive
	dst.Force = dst.Force || src.Force
	dst.Unwrapped = dst.Unwrapped || src.Unwrapped
}

// processImport replaces every lx.import(...) call with the compiled code
// of its path arguments (inlined in place, like the old @lx:require) while
// its module arguments are resolved through the same module registry/
// dependency mechanism the old @lx:use used, accumulating into c.modulesCode
// rather than being inlined at the call site.
func (c *Compiler) processImport(code, rootPath string) (string, error) {
	calls := findImportCalls(code)
	if len(calls) == 0 {
		return code, nil
	}

	parentDir := ""
	if rootPath != "" {
		parentDir = filepath.Dir(rootPath) + "/"
	}

	var allModuleNames []string
	for _, call := range calls {
		allModuleNames = append(allModuleNames, call.modules...)
	}

	if len(allModuleNames) > 0 {
		if !c.buildModules {
			c.compiledModules = allModuleNames
		} else {
			var filePaths []string
			var modulesForBuild []string
			for _, moduleName := range allModuleNames {
				c.checkModule(moduleName, &modulesForBuild, &filePaths)
			}

			modulesCode, err := c.compileFileGroup(filePaths, Flags{}, rootPath)
			if err != nil {
				return "", err
			}

			for _, m := range modulesForBuild {
				if !slices.Contains(c.compiledModules, m) {
					c.compiledModules = append(c.compiledModules, m)
				}
			}

			c.modulesCode += modulesCode
		}
	}

	var out strings.Builder
	pos := 0
	for _, call := range calls {
		out.WriteString(code[pos:call.start])
		for _, p := range call.paths {
			includedCode, err := c.plugRequire(p.Path, p.Flags, parentDir, rootPath)
			if err != nil {
				c.pp.LogError("Can not process lx.import path %s: %v", p.Path, err)
				continue
			}
			out.WriteString(includedCode)
		}
		pos = call.end
	}
	out.WriteString(code[pos:])

	return out.String(), nil
}
