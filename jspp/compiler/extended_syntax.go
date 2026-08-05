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
		// @lx:const NAME = value;
		re := regexp.MustCompile(`@lx:const +?\b(.+?)\b\s*=\s*([^;]+?);`)
		classCode := info.fullCode
		matches := re.FindAllStringSubmatch(classCode, -1)
		for _, match := range matches {
			str := match[0]
			name := match[1]
			val := match[2]
			constCode := fmt.Sprintf(`static get %s(){return %s;}`, name, val)
			classCode = strings.Replace(classCode, str, constCode, 1)
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
