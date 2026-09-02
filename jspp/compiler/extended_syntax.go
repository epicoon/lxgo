package compiler

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/epicoon/lxgo/jspp/internal/utils"
)

func applyExtendedSyntax(code, path string) (string, error) {
	// lx.self(KEY). => this.constructor.KEY
	re := regexp.MustCompile(`lx\.self\(`)
	do := true
	for do {
		inxs := re.FindStringIndex(code)
		if len(inxs) == 0 {
			do = false
			continue
		}

		start, finish := inxs[0], inxs[1]
		end := utils.FindMatchingBrace(code, finish-1, '(')

		key := code[finish:end]
		orig := code[start : end+1]
		code = strings.Replace(code, orig, "this.constructor."+key, 1)
	}

	// lx(elem)>>child>child => element.find('child').get('child')
	re = regexp.MustCompile(`lx\((.+?)\)(?:(?:>>|>)\b[\w\d_]+\b)+`)
	code = re.ReplaceAllStringFunc(code, func(match string) string {
		matches := re.FindStringSubmatch(match)
		if len(matches) < 2 {
			return match
		}
		str := matches[1]
		keys := strings.Split(matches[0], ">")[1:]
		find := false
		for _, key := range keys {
			if key == "" {
				find = true
				continue
			}
			if find {
				str += fmt.Sprintf(".find('%s')", key)
			} else {
				str += fmt.Sprintf(".get('%s')", key)
			}
			find = false
		}
		return str
	})

	return applyExtendedSyntaxForClasses(code, path)
}

func applyExtendedSyntaxForClasses(code, path string) (string, error) {
	classesInfo, err := findClasses(code)
	if err != nil {
		return "", fmt.Errorf("can not parse classes in '%s': %w", path, err)
	}
	if classesInfo == nil {
		return code, nil
	}

	for _, info := range classesInfo {
		// @lx:const NAME = value[;] - see applyLxConst for the supported
		// value shapes (scalar, string, multi-line array/object).
		classCode, err := applyLxConst(info.fullCode)
		if err != nil {
			return "", fmt.Errorf("can not parse @lx:const in '%s': %w", path, err)
		}

		// @lx:behavior(s) Name1, Name2; - only meaningful on a class that
		// extends something with lx.Object's __afterDefinition (which calls
		// __injectBehaviors() if the class defines one) - a class with no
		// parent at all would just define a static method nothing ever calls.
		if info.extends != "" {
			re := regexp.MustCompile(`@lx:behaviors?\s+([^;]+?)\s*;`)
			matches := re.FindAllStringSubmatch(classCode, -1)
			for _, match := range matches {
				str := match[0]
				names := regexp.MustCompile(`\s*,\s*`).Split(match[1], -1)
				var calls strings.Builder
				for _, name := range names {
					calls.WriteString(name)
					calls.WriteString(".injectInto(this);")
				}
				behaviorsCode := fmt.Sprintf(`static __injectBehaviors(){%s}`, calls.String())
				classCode = strings.Replace(classCode, str, behaviorsCode, 1)
			}
		}

		// @lx:namespace NMSP;
		if info.namespace != "" {
			re := regexp.MustCompile(fmt.Sprintf(`@lx:namespace\s+%s\s*;\s*`, info.namespace))
			classCode = re.ReplaceAllString(classCode, "")
			classCode = fmt.Sprintf(`lx.createNamespace('%s');if('%s' in lx.globalContext.%s)return;`, info.namespace, info.name, info.namespace) +
				classCode +
				fmt.Sprintf(`%s.__namespace='%s';lx.globalContext.%s.%s=%s;`, info.name, info.namespace, info.namespace, info.name, info.name)
		}
		classCode += fmt.Sprintf(`if(%s.__afterDefinition)%s.__afterDefinition();`, info.name, info.name)
		if info.namespace != "" {
			classCode = `(()=>{` + classCode + `})();`
		}

		code = strings.Replace(code, info.fullCode, classCode, 1)
	}

	return code, nil
}

// constDeclRe matches "@lx:const NAME =" - the marker, the constant's
// identifier, and the "=" sign - leaving the value's own text for
// applyLxConst to locate separately, since a value can be a scalar (ends at
// the first whitespace/";"), a quoted string (ends at its own closing
// quote, which may come after an internal ";"), or a multi-line array/
// object literal (ends at its own matching bracket).
var constDeclRe = regexp.MustCompile(`@lx:const\s+\b([\w\d_]+)\b\s*=\s*`)

// applyLxConst rewrites every "@lx:const NAME = VALUE;" (or "@lx:const
// NAME = VALUE" with no trailing ";" - optional either way) declaration in
// classCode into a static read-only getter. VALUE is read according to
// what it starts with:
//   - a quote ('/"/`) - up to its own matching, unescaped closing quote,
//     so a string value may itself contain ";" without ending the value early
//     (e.g. @lx:const GREETING = "hi; bye";)
//   - "[" or "{" - up to its own matching closing bracket (utils.
//     FindMatchingBrace), so an array/object value may freely span several
//     lines and nest further arrays/objects
//   - anything else (a bare scalar) - up to the first whitespace or ";"
func applyLxConst(classCode string) (string, error) {
	for {
		loc := constDeclRe.FindStringSubmatchIndex(classCode)
		if loc == nil {
			return classCode, nil
		}
		matchStart, nameStart, nameEnd, valStart := loc[0], loc[2], loc[3], loc[1]
		name := classCode[nameStart:nameEnd]

		if valStart >= len(classCode) {
			return "", fmt.Errorf("@lx:const %s: missing value", name)
		}

		var valEnd int
		switch classCode[valStart] {
		case '\'', '"', '`':
			q := findMatchingQuote(classCode, valStart)
			if q == -1 {
				return "", fmt.Errorf("@lx:const %s: unterminated string value", name)
			}
			valEnd = q + 1
		case '[':
			end := utils.FindMatchingBrace(classCode, valStart, '[')
			if end == -1 {
				return "", fmt.Errorf("@lx:const %s: unterminated array value", name)
			}
			valEnd = end + 1
		case '{':
			end := utils.FindMatchingBrace(classCode, valStart, '{')
			if end == -1 {
				return "", fmt.Errorf("@lx:const %s: unterminated object value", name)
			}
			valEnd = end + 1
		default:
			valEnd = scalarValueEnd(classCode, valStart)
		}

		value := strings.TrimSpace(classCode[valStart:valEnd])

		consumedEnd := valEnd
		if consumedEnd < len(classCode) && classCode[consumedEnd] == ';' {
			consumedEnd++
		}

		constCode := fmt.Sprintf(`static get %s(){return %s;}`, name, value)
		classCode = classCode[:matchStart] + constCode + classCode[consumedEnd:]
	}
}

// findMatchingQuote returns the index of the unescaped quote character
// matching the one at code[start] (which must itself be a quote: ', ", or
// `), or -1 if the string literal never closes.
func findMatchingQuote(code string, start int) int {
	quote := code[start]
	n := len(code)
	for i := start + 1; i < n; i++ {
		if code[i] == '\\' {
			i++
			continue
		}
		if code[i] == quote {
			return i
		}
	}
	return -1
}

// scalarValueEnd returns the index of the first whitespace character or
// ";" at or after start, or len(code) if the scalar runs to the end.
func scalarValueEnd(code string, start int) int {
	n := len(code)
	for i := start; i < n; i++ {
		switch code[i] {
		case ' ', '\t', '\n', '\r', ';':
			return i
		}
	}
	return n
}

type classInfo struct {
	namespace string
	name      string
	extends   string
	fullCode  string
}

func findClasses(code string) ([]classInfo, error) {
	re := regexp.MustCompile(`(?:@lx:namespace\s+[\w_][\w\d_.]*;)?\s*class\s+\b.+?\b[^{]*?{`)
	matches := re.FindAllStringIndex(code, -1)
	if len(matches) == 0 {
		return nil, nil
	}

	result := make([]classInfo, 0, 1)
	for _, match := range matches {
		info := new(classInfo)
		start, finish := match[0], match[1]
		end := utils.FindMatchingBrace(code, finish-1, '{')
		if end == -1 {
			return nil, errors.New("wrong braces matching")
		}

		info.fullCode = code[start : end+1]
		re := regexp.MustCompile(`(?:@lx:namespace\s+([\w_][\w\d_.]*?);)?\s*class\s+\b(.+?)\b\s+(?:extends\s+([\w_][\w\d_.]*?))?`)
		matches := re.FindAllStringSubmatch(info.fullCode, -1)
		info.namespace = matches[0][1]
		info.name = matches[0][2]
		info.extends = matches[0][3]
		result = append(result, *info)
	}

	return result, nil
}
