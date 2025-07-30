package plugins

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/epicoon/lxgo/jspp"
)

type renderer struct {
	manager *PluginManager
	plugin  jspp.IPlugin
	lang    string
}

type filler struct {
	Title  string
	Icon   string
	CoreJS string
	Html   string
	JS     string
	Assets struct {
		Scripts []string
		Css     []string
	}
}

func newRenderer(manager *PluginManager, plugin jspp.IPlugin, lang string) *renderer {
	return &renderer{
		manager: manager,
		plugin:  plugin,
		lang:    lang,
	}
}

func (r *renderer) render() (string, error) {
	f, err := r.prepareFiller()
	if err != nil {
		return "", err
	}

	plugin := r.plugin
	var layoutPath string
	nmsp := plugin.Config().Page().Template().Namespace
	layoutPath = plugin.App().TemplateHolder().LayoutPath(nmsp)

	if layoutPath == "" {
		return r.fillDefault(f), nil
	} else {
		return r.fillCustom(layoutPath, f), nil
	}
}

func (r *renderer) fillDefault(f *filler) string {
	// Prepare assets
	assets := ""
	for _, css := range f.Assets.Css {
		assets += fmt.Sprintf("<link rel=\"stylesheet\" type=\"text/css\" href=\"%s\">", css)
	}
	for _, js := range f.Assets.Scripts {
		assets += fmt.Sprintf("<script src=\"%s\"></script>", js)
	}

	layout := `<!DOCTYPE html>
		<html lang="en">
		<head>
			<meta charset="UTF-8">
			<title>%s</title>
			<link rel="shortcut icon" type="image/png" href="%s" />
			<script defer src="%s"></script>
			%s
		</head>
		<body>
		<div lxid="lx-root" class="lxbody">
		%s
		</div>
		<script>document.addEventListener("DOMContentLoaded",()=>{%s})</script>
		</body>
		</html>`
	return fmt.Sprintf(layout, f.Title, f.Icon, f.CoreJS, assets, f.Html, f.JS)
}

func (r *renderer) fillCustom(layoutPath string, f *filler) string {
	d, err := os.ReadFile(layoutPath)
	if err != nil {
		if os.IsNotExist(err) {
			r.manager.pp.LogError("layout file '%s' not found", layoutPath)
		} else {
			r.manager.pp.LogError("can not read layout file '%s': %v", layoutPath, err)
		}
		return r.fillDefault(f)
	}

	layout := string(d)
	plugin := r.plugin

	// Insert Title
	if f.Title != "" {
		title := r.plugin.I18n().Localize(f.Title, r.lang)
		re := regexp.MustCompile(`<title.+?</title>`)
		if re.MatchString(layout) {
			layout = re.ReplaceAllString(layout, "<title>"+title+"</title>")
		} else {
			re := regexp.MustCompile(`</head>`)
			layout = re.ReplaceAllString(layout, "<title>"+title+"</title></head>")
		}
	}

	// Insert Icon
	if f.Icon != "" {
		re := regexp.MustCompile(`<link .*?rel\s*=\s*"icon".*?>`)
		layout = re.ReplaceAllString(layout, "")
		re = regexp.MustCompile(`</head>`)
		layout = re.ReplaceAllString(layout, "<link rel=\"icon\" href=\""+f.Icon+"\" /></head>")
	}

	// Insert assets
	assets := ""
	if !strings.Contains(layout, f.CoreJS) {
		assets += fmt.Sprintf("<script defer src=\"%s\"></script>", f.CoreJS)
	}
	for _, css := range f.Assets.Css {
		if !strings.Contains(layout, css) {
			assets += fmt.Sprintf("<link rel=\"stylesheet\" type=\"text/css\" href=\"%s\">", css)
		}
	}
	for _, js := range f.Assets.Scripts {
		if !strings.Contains(layout, js) {
			assets += fmt.Sprintf("<script src=\"%s\"></script>", js)
		}
	}
	re := regexp.MustCompile(`</head>`)
	layout = re.ReplaceAllString(layout, assets+"</head>")

	block := fmt.Sprintf(`{{define "%s"}}
	<div lxid="lx-root" class="lxbody">%s</div>
	<script>document.addEventListener("DOMContentLoaded",()=>{%s})</script>
	{{end}}`, plugin.Config().Page().Template().Block, f.Html, f.JS)

	res, err := plugin.App().TemplateRenderer().
		SetNamespace(plugin.Config().Page().Template().Namespace).
		SetLayout(layout).
		SetContent(block).
		Render()
	if err != nil {
		r.manager.pp.LogError("can not render plugin '%s' with layout: %v", plugin.Name(), err)
		return r.fillDefault(f)
	}

	return res
}

func (r *renderer) prepareFiller() (*filler, error) {
	pp := r.manager.pp
	plugin := r.plugin

	// Render plugin
	res, err := r.manager.Render(plugin, r.lang)
	if err != nil {
		return nil, fmt.Errorf("can not render plugin '%s': %v", plugin.Name(), err)
	}

	// Define core.js route
	corePath := pp.Config().CorePath
	coreFile := filepath.Base(corePath)
	coreDir := filepath.Dir(corePath)
	coreRoute := pp.App().Router().GetAssetRoute(coreDir)
	coreRoute = filepath.Join(coreRoute, coreFile)

	// Build app start code
	infoLx, err := json.Marshal(res.Lx)
	if err != nil {
		return nil, fmt.Errorf("can not marshal json lx-info for plugin '%s': %v", plugin.Name(), err)
	}
	code, err := pp.CompilerBuilder().
		SetClientContext().
		SetAppContext().
		SetLang(r.lang).
		UseModules(res.Assets.Modules).
		SetCode(fmt.Sprintf(`lx.app.root.setPlugin({info:{root:'%s',lx:%s}})`, res.Root, string(infoLx))).
		Compiler().Run()
	if err != nil {
		return nil, fmt.Errorf("can not marshal json lx-info for plugin '%s': %v", plugin.Name(), err)
	}

	// Page title and icon
	title := plugin.Config().Page().Title()
	title = plugin.I18n().Localize(title, r.lang)
	icon := plugin.Config().Page().Icon()
	if icon != "data:," {
		al := newAssetLinker(pp, plugin.Pathfinder())
		icon = al.getAsset(icon)
	}

	return &filler{
		Title:  title,
		Icon:   icon,
		CoreJS: coreRoute,
		Assets: struct {
			Scripts []string
			Css     []string
		}{
			Scripts: res.Assets.Scripts,
			Css:     res.Assets.Css,
		},
		Html: res.Html,
		JS:   code,
	}, nil
}
