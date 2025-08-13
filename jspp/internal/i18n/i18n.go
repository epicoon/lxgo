package i18n

import (
	"regexp"
	"strings"

	"github.com/epicoon/lxgo/jspp"
	"github.com/epicoon/lxgo/jspp/internal/utils"
)

/** @interface conventions.II18nMap */
type I18nMap struct {
	tr map[string]map[string]string
}

var _ jspp.II18nMap = (*I18nMap)(nil)

/** @constructor */
func NewI18nMap(tr map[string]map[string]string) jspp.II18nMap {
	return &I18nMap{tr: tr}
}

func (m *I18nMap) IsEmpty() bool {
	return len(m.tr) == 0
}

func (m *I18nMap) Get(lang string, key string) string {
	l, exists := m.tr[lang]
	if !exists {
		return ""
	}
	return l[key]
}

func (m *I18nMap) Localize(text string, lang string) string {
	langMap, ok := m.tr[lang]

	re := regexp.MustCompile(`lx\.i18n\(`)
	do := true
	for do {
		inxs := re.FindStringIndex(text)
		if len(inxs) == 0 {
			do = false
			continue
		}

		start, finish := inxs[0], inxs[1]
		end := utils.FindMatchingBrace(text, finish-1, '(')

		key := strings.Trim(text[finish:end], `'"`)
		orig := text[start : end+1]
		var params map[string]string
		key, params = extractParams(key)

		var tr string
		if ok {
			tr = langMap[key]
		}
		if tr != "" {
			if len(params) > 0 {
				re := regexp.MustCompile(`\$\{(.+?)\}`)
				tr = re.ReplaceAllStringFunc(tr, func(s string) string {
					matches := re.FindStringSubmatch(s)
					if len(matches) < 2 {
						return s
					}
					k := matches[1]
					val, exists := params[k]
					if !exists {
						val = ""
					}
					return val
				})
			}
			tr = "'" + tr + "'"
			text = strings.Replace(text, orig, tr, 1)
			continue
		}

		re = regexp.MustCompile(`^module\-[^\-]+\-`)
		key = re.ReplaceAllString(key, "")
		text = strings.Replace(text, orig, "'"+key+"'", 1)
	}

	return text
}

func extractParams(text string) (string, map[string]string) {
	m := make(map[string]string, 0)
	re := regexp.MustCompile(`^(.+?)\s*,\s*([\w\W]+?)$`)
	match := re.FindStringSubmatch(text)
	if len(match) < 3 {
		return text, m
	}

	key := match[1]
	sParams := strings.Split(strings.Trim(match[2], "{}\n\r\t"), ",")
	for _, item := range sParams {
		pare := strings.Split(item, ":")
		if len(pare) != 2 {
			//TODO log error
			continue
		}
		m[strings.Trim(pare[0], " \n\r\t")] = strings.Trim(pare[1], " \n\r\t")
	}

	return key, m
}
