// Package md is the platform's own markdown-to-HTML converter, used by the
// `lx.md(...)` preprocessor directive
package md

import "os"

// Convert renders markdown text to an HTML string.
func Convert(mdText string) string {
	blocks := parse(mdText)
	return newRenderer().run(blocks)
}

// ConvertFile reads a markdown file and renders it to an HTML string.
func ConvertFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return Convert(string(data)), nil
}
