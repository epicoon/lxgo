package utils

func FindMatchingBrace(code string, start int, brace rune) int {
	var contr rune
	switch brace {
	case '{':
		contr = '}'
	case '(':
		contr = ')'
	case '[':
		contr = ']'
	}

	stack := 0
	for i := start; i < len(code); i++ {
		if code[i] == byte(brace) {
			stack++
		} else if code[i] == byte(contr) {
			stack--
			if stack == 0 {
				return i
			}
		}
	}
	return -1
}
