package i18n

import (
	"regexp"

	"github.com/epicoon/lxgo/jspp"
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
	re := regexp.MustCompile(`lx\(i18n\)\.([\w\d_\-.]+)`)
	text = re.ReplaceAllStringFunc(text, func(match string) string {
		matches := re.FindStringSubmatch(match)
		if len(matches) < 2 {
			return match
		}

		key := matches[1]
		var tr string
		if ok {
			tr = langMap[key]
		}
		if tr != "" {
			return "'" + tr + "'"
		}

		return "'" + key + "'"
	})

	return text
}
