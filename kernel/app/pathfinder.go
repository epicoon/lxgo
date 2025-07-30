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

type Pathfinder struct {
	root string
}

/** @interface */
var _ kernel.IPathfinder = (*Pathfinder)(nil)

func NewPathfinder(root string) *Pathfinder {
	pf := &Pathfinder{root: root}
	return pf
}

func (pf *Pathfinder) GetRoot() string {
	return pf.root
}

func (pf *Pathfinder) GetAbsPath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(pf.root, path)
}

type appPathfinder struct {
	*Pathfinder
	app     kernel.IApp
	aliases map[string]string
}

/** @interface */
var _ kernel.IPathfinder = (*appPathfinder)(nil)

func NewAppPathfinder(app kernel.IApp) *appPathfinder {
	return &appPathfinder{
		Pathfinder: NewPathfinder(getProjectRoot()),
		app:        app,
	}
}

func (pf *appPathfinder) GetAbsPath(path string) string {
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

	pfConf, err := config.GetParam[kernel.Config](c, "Pathfinder")
	if err != nil {
		pf.app.LogError(fmt.Sprintf("can not get application config parameter 'Pathfinder': %v", err), "App")
		pf.aliases = make(map[string]string)
		return pf.aliases
	}
	if !config.HasParam(&pfConf, "Aliases") {
		pf.aliases = make(map[string]string)
		return pf.aliases
	}

	aliases, err := config.GetParam[kernel.Config](&pfConf, "Aliases")
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
	for !isGoModPresent(wd) && wd != "/" {
		wd = filepath.Dir(wd)
	}
	return wd
}

func isGoModPresent(path string) bool {
	_, err := os.Stat(filepath.Join(path, "go.mod"))
	return err == nil
}
