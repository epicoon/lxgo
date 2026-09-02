package i18n

import (
	"fmt"
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
	return LocalizeWithLookup(text, func(key string) string {
		if !ok {
			return ""
		}
		return langMap[key]
	})
}

// LocalizeWithLookup replaces every lx.i18n(key) / lx.i18n(key,
// {placeholders}) call in text with lookup(key)'s translation (placeholders
// substituted in, if any), or the key itself (stripped of any leading
// module-<name>- prefix) if lookup returns "".
//
// It exists as its own entry point, separate from Localize's single fixed
// dictionary, so a caller juggling more than one translation source (e.g. a
// module's own i18n data plus a plugin's own) can offer both to ONE pass
// via a single combined lookup - running two independent Localize passes
// back to back doesn't work for that: the first pass's own "no translation
// found - fall back to the bare key" would give up permanently on any key
// only the second pass actually has, since by the time the second pass
// runs, that call's lx.i18n(...) syntax has already been rewritten away.
func LocalizeWithLookup(text string, lookup func(key string) string) string {
	markerRe := regexp.MustCompile(`lx\.i18n\(`)
	modulePrefixRe := regexp.MustCompile(`^module\-[^\-]+\-`)
	do := true
	for do {
		inxs := markerRe.FindStringIndex(text)
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
		key = strings.Trim(key, `'"`)

		tr := lookup(key)
		if tr != "" {
			if len(params) > 0 {
				tr = "`" + tr + "`"
				pp := make([]string, len(params))
				i := 0
				for name, val := range params {
					mangled := "i18n_" + name
					pp[i] = mangled + "=" + val
					tr = regexp.MustCompile(`\$\{\s*`+regexp.QuoteMeta(name)+`\s*\}`).ReplaceAllLiteralString(tr, "${"+mangled+"}")
					i++
				}
				spp := "let " + strings.Join(pp, ",") + ";"
				tr = fmt.Sprintf("(()=>{%sreturn %s})()", spp, tr)
			} else {
				tr = "'" + tr + "'"
			}
			text = strings.Replace(text, orig, tr, 1)
			continue
		}

		key = modulePrefixRe.ReplaceAllString(key, "")
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
		item = strings.Trim(item, " \n\r\t")
		if item == "" {
			continue
		}
		pare := strings.Split(item, ":")
		switch len(pare) {
		case 1:
			// JS shorthand property syntax - {points} means {points: points}.
			name := strings.Trim(pare[0], " \n\r\t")
			m[name] = name
		case 2:
			m[strings.Trim(pare[0], " \n\r\t")] = strings.Trim(pare[1], " \n\r\t")
		default:
			//TODO log error
		}
	}

	return key, m
}
