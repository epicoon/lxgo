package compiler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/epicoon/lxgo/jspp/internal/md"
)

var mdDirectiveRe = regexp.MustCompile(`@lx:md\s*\(\s*['"]?(.*?)['"]?\s*\)`)

// parseMd handles the `@lx:md('rel/path')` directive: the path is resolved
// relative to the directory of the file being compiled, ".md" is appended if
// missing, and the directive is replaced with a JSON-encoded string of the
// rendered HTML (or "" if the file doesn't exist).
//
// No guard is needed against matching inside a not-yet-uncommented
// "// @lx:md(...)" — by the time this runs, cutComments has already stripped
// all real comments, and @lx:-prefixed directives were already uncommented
// earlier in the pipeline (see compileCodeInnerDirectives).
func (c *Compiler) parseMd(code, path string) string {
	return mdDirectiveRe.ReplaceAllStringFunc(code, func(m string) string {
		sub := mdDirectiveRe.FindStringSubmatch(m)
		relPath := sub[1]
		if !strings.HasSuffix(relPath, ".md") {
			relPath += ".md"
		}

		fullPath := relPath
		if path != "" {
			fullPath = filepath.Join(filepath.Dir(path), relPath)
		}

		if _, err := os.Stat(fullPath); err != nil {
			return `""`
		}

		html, err := md.ConvertFile(fullPath)
		if err != nil {
			c.pp.LogError("can not convert markdown file '%s': %v", fullPath, err)
			return `""`
		}

		encoded, _ := json.Marshal(html)
		return string(encoded)
	})
}
