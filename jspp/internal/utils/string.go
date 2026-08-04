package utils

import "strings"

// FindMatchingBrace returns the index of the brace that closes the one at
// code[start] (must be brace itself - '{', '(', or '[') - skipping over
// any of the same bracket characters that appear inside a string literal
// ('...', "...", `...`, with backslash-escape support), a // / /* */
// comment, or a JS regex literal (/pattern/flags), so they don't throw
// off the depth count. Returns -1 if the brace is never closed.
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

	n := len(code)
	stack := 0
	var quote byte
	var lastSignificant byte

	for i := start; i < n; i++ {
		ch := code[i]

		if quote != 0 {
			if ch == '\\' && i+1 < n {
				i++
				continue
			}
			if ch == quote {
				quote = 0
				lastSignificant = 'x'
			}
			continue
		}

		if ch == '\'' || ch == '"' || ch == '`' {
			quote = ch
			continue
		}

		if ch == '/' && i+1 < n && code[i+1] == '/' {
			for i < n && code[i] != '\n' {
				i++
			}
			continue
		}

		if ch == '/' && i+1 < n && code[i+1] == '*' {
			if end := strings.Index(code[i+2:], "*/"); end != -1 {
				i = i + 2 + end + 1
				continue
			}
			// no closing "*/" - fall through, treat "/" as ordinary code
		}

		if ch == '/' && LooksLikeRegexStart(lastSignificant) {
			if end := FindRegexLiteralEnd(code, i+1); end != -1 {
				i = end
				for i+1 < n && IsRegexFlag(code[i+1]) {
					i++
				}
				lastSignificant = 'x'
				continue
			}
		}

		if ch == byte(brace) {
			stack++
		} else if ch == byte(contr) {
			stack--
			if stack == 0 {
				return i
			}
		}

		if ch != ' ' && ch != '\t' && ch != '\n' && ch != '\r' {
			lastSignificant = ch
		}
	}
	return -1
}

// LooksLikeRegexStart decides whether a '/' encountered right after
// lastSignificant (the last non-whitespace character already scanned, 0 if
// none yet) is more likely to open a JS regex literal than to be a division
// operator. JS itself only disambiguates this by full tokenization (a '/'
// after a value - identifier, number, string, ')', ']' - is division,
// anywhere else it starts a regex) - this covers the common, unambiguous
// cases without going that far.
func LooksLikeRegexStart(lastSignificant byte) bool {
	switch lastSignificant {
	case 0, '(', '=', ',', ':', ';', '[', '{', '!', '&', '|', '?', '+', '-', '*', '%', '<', '>', '\n':
		return true
	}
	return false
}

// IsRegexFlag reports whether ch is one of the JS regex literal flag letters
// that can follow the closing '/' (e.g. the "g" in /foo/g).
func IsRegexFlag(ch byte) bool {
	switch ch {
	case 'g', 'i', 'm', 's', 'u', 'y', 'd':
		return true
	}
	return false
}

// FindRegexLiteralEnd returns the index of the '/' that closes a regex
// literal whose body starts at from (right after the opening '/'), honoring
// backslash-escapes and character classes ([...], where an unescaped '/'
// doesn't end the literal). Returns -1 if the body reaches a newline or the
// end of code without closing - not a regex literal after all.
func FindRegexLiteralEnd(code string, from int) int {
	n := len(code)
	inClass := false
	for i := from; i < n; i++ {
		ch := code[i]
		if ch == '\\' && i+1 < n {
			i++
			continue
		}
		if ch == '\n' {
			return -1
		}
		if ch == '[' {
			inClass = true
			continue
		}
		if ch == ']' {
			inClass = false
			continue
		}
		if ch == '/' && !inClass {
			return i
		}
	}
	return -1
}
