package compiler

import (
	"regexp"

	"github.com/epicoon/lxgo/jspp/internal/utils"
)

// applyMacros handles the `@lx:macros NAME { ... };` directive: every
// occurrence of `lx>>>NAME` elsewhere in the code is replaced with the macro's
// body, then the macro declarations themselves are stripped from the code.
func (c *Compiler) applyMacros(code string) string {
	declRe := regexp.MustCompile(`@lx:macros\s+([\w\d_]+)\s*{`)
	macros := map[string]string{}

	for {
		loc := declRe.FindStringSubmatchIndex(code)
		if loc == nil {
			break
		}

		name := code[loc[2]:loc[3]]
		braceStart := loc[1] - 1
		braceEnd := utils.FindMatchingBrace(code, braceStart, '{')
		if braceEnd == -1 {
			break
		}

		macros[name] = code[braceStart+1 : braceEnd]

		end := braceEnd + 1
		if end < len(code) && code[end] == ';' {
			end++
		}
		code = code[:loc[0]] + code[end:]
	}

	for name, text := range macros {
		useRe := regexp.MustCompile(`lx>>>` + regexp.QuoteMeta(name) + `\b`)
		code = useRe.ReplaceAllLiteralString(code, text)
	}

	return code
}
