package utils

import (
	"path/filepath"
	"regexp"

	"github.com/epicoon/lxgo/jspp"
	"github.com/epicoon/lxgo/kernel"
	lxApp "github.com/epicoon/lxgo/kernel/app"
)

type pathfinder struct {
	*lxApp.Pathfinder

	pp jspp.IPreprocessor
}

func NewPathfinder(pp jspp.IPreprocessor) kernel.IPathfinder {
	return &pathfinder{
		Pathfinder: lxApp.NewPathfinder(pp.App().Pathfinder().GetRoot()),
		pp:         pp,
	}
}

func (pf *pathfinder) GetAbsPath(path string) string {
	if path == "" {
		return ""
	}

	if pPath, ok := ResolvePluginPath(pf.pp, path); ok {
		return pPath
	}

	//TODO smth else?

	return pf.pp.App().Pathfinder().GetAbsPath(path)
}

func ResolvePluginPath(pp jspp.IPreprocessor, path string) (string, bool) {
	if len(path) == 0 {
		return "", false
	}

	if path[0] != '{' {
		return "", false
	}

	// {plugin:PluginName}/path/to/file
	re := regexp.MustCompile(`^\{plugin:([^}]+?)\}(.*)$`)
	matches := re.FindStringSubmatch(path)
	if len(matches) == 3 {
		plugin := pp.PluginManager().Get(matches[1])
		if plugin == nil {
			pp.LogError("can not find plugin '%s'", matches[1])
			return "", false
		}
		return filepath.Join(plugin.Pathfinder().GetRoot(), matches[2]), true
	}

	return "", false
}
