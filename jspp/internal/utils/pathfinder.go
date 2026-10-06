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

	// {@param(Dotted.Config.Path)}/rest/of/path - substitutes the app's
	// own config value (kernel.IApp.ConfigParam) at that dotted path. The
	// two failure cases below return ("", true), not ("", false) - the
	// "{@param(...)}" syntax was recognized, so the caller must not fall
	// through to treating the raw, unsubstituted directive text as a
	// literal path (see pathfinder.GetAbsPath).
	paramRe := regexp.MustCompile(`^\{@param\(([^)]+?)\)\}(.*)$`)
	paramMatches := paramRe.FindStringSubmatch(path)
	if len(paramMatches) == 3 {
		val := pp.App().ConfigParam(paramMatches[1])
		if val == nil {
			pp.LogError("config param '%s' not found", paramMatches[1])
			return "", true
		}
		str, ok := val.(string)
		if !ok {
			pp.LogError("config param '%s' is not a string (got %T)", paramMatches[1], val)
			return "", true
		}
		return str + paramMatches[2], true
	}

	return "", false
}
