package query

import (
	"reflect"
	"strings"
)

func tableName[T any]() string {
	t := reflect.TypeOf((*T)(nil)).Elem()
	return toSnake(t.Name()) + "s"
}

func toSnake(str string) string {
	var result []rune
	for i, r := range str {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
}
