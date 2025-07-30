package plugins

import (
	"path/filepath"
	"regexp"

	"github.com/epicoon/lxgo/jspp"
	"github.com/epicoon/lxgo/kernel"
	lxApp "github.com/epicoon/lxgo/kernel/app"
)

type pluginPathfinder struct {
	*lxApp.Pathfinder
	plugin jspp.IPlugin
}

var _ kernel.IPathfinder = (*pluginPathfinder)(nil)

func newPluginPathfinder(plugin jspp.IPlugin) *pluginPathfinder {
	return &pluginPathfinder{
		Pathfinder: lxApp.NewPathfinder(plugin.Path()),
		plugin:     plugin,
	}
}

func (p *pluginPathfinder) GetAbsPath(path string) string {
	if path[0] == '@' {
		return p.plugin.App().Pathfinder().GetAbsPath(path)
	}

	if path[0] == '{' {
		// {plugin:PluginName}/path/to/file
		re := regexp.MustCompile(`^\{plugin:([^}]+?)\}(.*)$`)
		matches := re.FindStringSubmatch(path)
		if len(matches) == 3 {
			plugin := p.plugin.Preprocessor().PluginManager().Get(matches[1])
			if plugin == nil {
				p.plugin.Preprocessor().LogError("can not find plugin '%s'", matches[1])
				return ""
			}
			return filepath.Join(plugin.Pathfinder().GetRoot(), matches[2])
		}

		// {snippet:PluginName.SnippetName}
		re = regexp.MustCompile(`^\{snippet:([^.]+?)\.(.+)\}$`)
		matches = re.FindStringSubmatch(path)
		if len(matches) == 3 {
			plugin := p.plugin.Preprocessor().PluginManager().Get(matches[1])
			if plugin == nil {
				p.plugin.Preprocessor().LogError("can not find plugin '%s'", matches[1])
				return ""
			}
			sm := plugin.Config().Server().SnippetsMap()
			path, exists := sm[matches[2]]
			if !exists {
				p.plugin.Preprocessor().LogError("can not find snippet '%s' in plugin '%s'", matches[2], matches[1])
				return ""
			}
			return plugin.Pathfinder().GetAbsPath(path)
		}

		//TODO log?
		return ""
	}

	return p.Pathfinder.GetAbsPath(path)
}
