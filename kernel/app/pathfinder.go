package app

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/config"
)

/** @interface kernel.IPathfinder */

// Pathfinder is the default kernel.IPathfinder implementation - resolves
// relative paths against a fixed root directory.
type Pathfinder struct {
	root string
}

var _ kernel.IPathfinder = (*Pathfinder)(nil)

/** @constructor */

// NewPathfinder constructs a Pathfinder rooted at root.
func NewPathfinder(root string) *Pathfinder {
	pf := &Pathfinder{root: root}
	return pf
}

// GetRoot returns the pathfinder's root directory.
func (pf *Pathfinder) GetRoot() string {
	return pf.root
}

// GetAbsPath resolves path against the root directory (returning path unchanged if it's already absolute).
func (pf *Pathfinder) GetAbsPath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(pf.root, path)
}

/** @interface kernel.IPathfinder */

type appPathfinder struct {
	*Pathfinder
	app     kernel.IApp
	aliases map[string]string
}

var _ kernel.IPathfinder = (*appPathfinder)(nil)

/** @constructor */

// NewAppPathfinder constructs the application's default IPathfinder, rooted
// at the nearest ancestor directory containing go.mod or bin, with support
// for "@alias/..." paths resolved via the app's config
// (Pathfinder.Aliases) - "@app/..." is a built-in alias for the root itself.
func NewAppPathfinder(app kernel.IApp) kernel.IPathfinder {
	return &appPathfinder{
		Pathfinder: NewPathfinder(getProjectRoot()),
		app:        app,
	}
}

func (pf *appPathfinder) GetAbsPath(path string) string {
	if path == "" {
		return pf.Pathfinder.GetAbsPath(path)
	}

	// Process aliases
	if path[0] == '@' {
		re := regexp.MustCompile(`^@([^/]+)/?(.*)$`)
		matches := re.FindStringSubmatch(path)
		if len(matches) < 3 {
			//TODO log?
			return pf.Pathfinder.GetAbsPath(path)
		}
		key := matches[1]
		if key == "app" {
			return pf.Pathfinder.GetAbsPath(strings.TrimPrefix(path, "@app/"))
		}

		aliases := pf.getAliases()
		if len(aliases) == 0 {
			//TODO log?
			return pf.Pathfinder.GetAbsPath(path)
		}

		replacement, exists := aliases[key]
		if !exists {
			//TODO log?
			return pf.Pathfinder.GetAbsPath(path)
		}

		tail := matches[2]
		return pf.Pathfinder.GetAbsPath(filepath.Join(replacement, tail))
	}

	return pf.Pathfinder.GetAbsPath(path)
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

func (pf *appPathfinder) getAliases() map[string]string {
	if pf.aliases != nil {
		return pf.aliases
	}

	c := pf.app.Config()
	if !config.HasParam(c, "Pathfinder") {
		pf.aliases = make(map[string]string)
		return pf.aliases
	}

	pfConf, err := config.GetParam[kernel.Dict](c, "Pathfinder")
	if err != nil {
		pf.app.LogError(fmt.Sprintf("can not get application config parameter 'Pathfinder': %v", err), "App")
		pf.aliases = make(map[string]string)
		return pf.aliases
	}
	if !config.HasParam(pfConf, "Aliases") {
		pf.aliases = make(map[string]string)
		return pf.aliases
	}

	aliases, err := config.GetParam[kernel.Dict](pfConf, "Aliases")
	if err != nil {
		pf.app.LogError(fmt.Sprintf("can not get application config parameter 'Pathfinder.Aliases': %v", err), "App")
		pf.aliases = make(map[string]string)
		return pf.aliases
	}

	pf.aliases = make(map[string]string, len(aliases))
	for key, val := range aliases {
		str, ok := val.(string)
		if !ok {
			pf.app.LogError(fmt.Sprintf("can not cast to string config parameter 'Pathfinder.Aliases.%s' = %v", key, val), "App")
			continue
		}
		pf.aliases[key] = str
	}

	return pf.aliases
}

func getProjectRoot() string {
	wd, _ := os.Getwd()
	for !isGoRoot(wd) && wd != "/" {
		wd = filepath.Dir(wd)
	}
	return wd
}

func isGoRoot(path string) bool {
	if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(path, "bin")); err == nil {
		return true
	}
	return false
}
