package plugins

import (
	"regexp"

	"github.com/epicoon/lxgo/jspp"
	"github.com/epicoon/lxgo/jspp/internal/utils"
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

// GetAbsPath resolves path against the plugin's root directory, additionally
// supporting "@alias/..." (delegated to the app's own pathfinder),
// "{plugin:Name}/rest/of/path" (another plugin's root) and
// "{snippet:PluginName.Key}" (a snippet registered in another plugin's
// server.snippetsMap) - see jspp doc/plugins.md's path syntax.
func (p *pluginPathfinder) GetAbsPath(path string) string {
	if len(path) == 0 {
		return ""
	}

	if path[0] == '@' {
		return p.plugin.App().Pathfinder().GetAbsPath(path)
	}

	if pPath, ok := utils.ResolvePluginPath(p.plugin.Preprocessor(), path); ok {
		return pPath
	}

	if path[0] == '{' {
		// {snippet:PluginName.SnippetName}
		re := regexp.MustCompile(`^\{snippet:([^.]+?)\.(.+)\}$`)
		matches := re.FindStringSubmatch(path)
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
