package inconf

import (
	"net"
	"strconv"
	"strings"

	"github.com/epicoon/lxgo/kernel"
)

func Run(app kernel.IApp, conn net.Conn, cmdParams []string) {
	isTest := false
	errList := make([]string, 0)
	params := make(map[string]any, 0)
	arrAdd := make(map[string][]any, 0)
	arrRemove := make(map[string][]any, 0)
	for _, str := range cmdParams {
		pair := strings.SplitN(str, "=", 2)
		if len(pair) < 2 {
			continue
		}
		key := strings.TrimSpace(pair[0])
		val := strings.TrimSpace(pair[1])
		switch key {
		case "t":
			isTest = true
		case "params":
			params = parseParamList(val, &errList)
		case "add":
			arrAdd = parseArr(val, &errList)
		case "remove":
			arrRemove = parseArr(val, &errList)
		}
	}

	if len(errList) > 0 {
		msg := strings.Join(errList, "\n")
		conn.Write([]byte("Syntax error:\n" + msg))
		return
	}

	if isTest {
		report := make([]string, 0)
		checkParams(app, params, &report)
		checkArrAdd(app, arrAdd, &report)
		checkArrRemove(app, arrRemove, &report)
		msg := strings.Join(report, "\n")
		conn.Write([]byte("Report:\n" + msg))
		return
	}

	//TODO

	conn.Write([]byte("Done\n"))
}

func parseParamList(s string, errList *[]string) map[string]any {
	res := make(map[string]any)

	tokens := splitRespectingQuotes(s)
	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}

		pair := strings.SplitN(token, ":", 2)
		if len(pair) != 2 {
			*errList = append(*errList, "invalid token: "+token)
			continue
		}

		key := strings.TrimSpace(pair[0])
		val := strings.TrimSpace(pair[1])

		if val == "false" {
			res[key] = false
			continue
		}
		if val == "true" {
			res[key] = true
			continue
		}
		if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') ||
			(val[0] == '\'' && val[len(val)-1] == '\'')) {
			res[key] = val[1 : len(val)-1]
			continue
		}
		if i, err := strconv.Atoi(val); err == nil {
			res[key] = i
			continue
		}
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			res[key] = f
			continue
		}
		// leave it as a line
		res[key] = val
	}

	return res
}

func splitRespectingQuotes(s string) []string {
	var result []string
	var current strings.Builder
	var inQuotes rune

	for _, r := range s {
		switch r {
		case '\'', '"':
			switch inQuotes {
			case 0:
				inQuotes = r
			case r:
				inQuotes = 0
			}
			current.WriteRune(r)
		case ',':
			if inQuotes != 0 {
				current.WriteRune(r)
			} else {
				result = append(result, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}

func parseArr(s string, errList *[]string) map[string][]any {
	result := make(map[string][]any)
	tokens := splitRespectingQuotesTopLevel(s, ',')
	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}

		parts := strings.SplitN(token, ":", 2)
		if len(parts) != 2 {
			*errList = append(*errList, "invalid token: "+token)
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if !strings.HasPrefix(val, "[") || !strings.HasSuffix(val, "]") {
			*errList = append(*errList, "invalid token - brackets required: "+token)
			continue
		}

		arrStr := strings.TrimSpace(val[1 : len(val)-1])
		arrTokens := splitRespectingQuotesTopLevel(arrStr, ',')

		for _, el := range arrTokens {
			el = strings.TrimSpace(el)
			if el == "" {
				continue
			}

			if len(el) >= 2 && ((el[0] == '"' && el[len(el)-1] == '"') ||
				(el[0] == '\'' && el[len(el)-1] == '\'')) {
				el = el[1 : len(el)-1]
				result[key] = append(result[key], el)
				continue
			}
			if i, err := strconv.Atoi(el); err == nil {
				result[key] = append(result[key], i)
				continue
			}
			if f, err := strconv.ParseFloat(el, 64); err == nil {
				result[key] = append(result[key], f)
				continue
			}
			result[key] = append(result[key], el)
		}
	}
	return result
}

func splitRespectingQuotesTopLevel(s string, sep rune) []string {
	var result []string
	var current strings.Builder
	var inQuotes rune
	depth := 0

	for _, r := range s {
		switch r {
		case '\'', '"':
			switch inQuotes {
			case 0:
				inQuotes = r
			case r:
				inQuotes = 0
			}
			current.WriteRune(r)
		case '[', '{', '(':
			if inQuotes == 0 {
				depth++
			}
			current.WriteRune(r)
		case ']', '}', ')':
			if inQuotes == 0 && depth > 0 {
				depth--
			}
			current.WriteRune(r)
		default:
			if r == sep && inQuotes == 0 && depth == 0 {
				result = append(result, current.String())
				current.Reset()
			} else {
				current.WriteRune(r)
			}
		}
	}

	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}
