package plugins

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/epicoon/lxgo/jspp"
	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/utils"
)

var absoluteURLRe = regexp.MustCompile(`^(http:|https:)`)

// isAbsoluteURL reports whether s is already a full http(s) URL, as opposed
// to a local filesystem path.
func isAbsoluteURL(s string) bool {
	return absoluteURLRe.MatchString(s)
}

type assetLink struct {
	key    string
	origin string
	link   string
	asset  string
}

type assetLinker struct {
	pp    jspp.IPreprocessor
	pf    kernel.IPathfinder
	links []*assetLink
}

func newAssetLinker(pp jspp.IPreprocessor, pf kernel.IPathfinder) *assetLinker {
	return &assetLinker{
		pp: pp,
		pf: pf,
	}
}

func (al *assetLinker) reset() {
	al.links = nil
}

func (al *assetLinker) getAssetsSlice(links []string) []string {
	al.setFromArray(links)
	al.defineLinks()
	al.createLinks()
	res := make([]string, 0, len(links))
	for _, l := range al.links {
		res = append(res, l.asset)
	}
	return res
}

func (al *assetLinker) getAssetsMap(links map[string]string) map[string]string {
	al.setFromMap(links)
	al.defineLinks()
	al.createLinks()
	res := make(map[string]string, len(links))
	for _, l := range al.links {
		res[l.key] = l.asset
	}
	return res
}

func (al *assetLinker) getAsset(link string) string {
	return al.getAssetsSlice([]string{link})[0]
}

func (al *assetLinker) setFromArray(links []string) {
	pf := al.pf
	app := al.pp.App()
	for _, str := range links {
		if str == "" {
			continue
		}
		al.links = append(al.links, &assetLink{
			origin: resolveOrigin(pf, app, str),
		})
	}
}

func (al *assetLinker) setFromMap(links map[string]string) {
	pf := al.pf
	app := al.pp.App()
	for key, str := range links {
		if str == "" {
			continue
		}
		al.links = append(al.links, &assetLink{
			key:    key,
			origin: resolveOrigin(pf, app, str),
		})
	}
}

// resolveOrigin resolves str into an assetLink's origin. An absolute
// http(s) URL whose host:port is this same app's own (app.Host()/
// app.Port() - see kernel.IApp) is treated as a local path after all.
func resolveOrigin(pf kernel.IPathfinder, app kernel.IApp, str string) string {
	if isAbsoluteURL(str) {
		if local, ok := stripSelfHost(app, str); ok {
			return app.Pathfinder().GetAbsPath(local)
		}
		return str
	}
	return pf.GetAbsPath(str)
}

// stripSelfHost reports whether rawURL's host:port matches app's own
// (app.Host(), app.Port()) - if so, ok is true and local is rawURL's own
// path component, with the leading "/" removed so it resolves relative
// to the app's root.
func stripSelfHost(app kernel.IApp, rawURL string) (local string, ok bool) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", false
	}
	if u.Hostname() != app.Host() || u.Port() == "" || u.Port() != strconv.Itoa(app.Port()) {
		return "", false
	}
	return strings.TrimPrefix(u.Path, "/"), true
}

func (al *assetLinker) defineLinks() {
	pp := al.pp
	innerPath := pp.App().Pathfinder().GetAbsPath(pp.Config().AssetLinksPath.Inner)
	outerPath := pp.Config().AssetLinksPath.Outer
	outerIsURL := isAbsoluteURL(outerPath)
	if !outerIsURL && outerPath[0] != '/' {
		outerPath = "/" + outerPath
	}

	for _, l := range al.links {
		if isAbsoluteURL(l.origin) {
			l.asset = l.origin
			continue
		}

		re := regexp.MustCompile(fmt.Sprintf(`^%s`, innerPath))
		if re.MatchString(l.origin) {
			asset, _ := strings.CutPrefix(l.origin, innerPath)
			l.asset = joinOuter(outerPath, outerIsURL, asset)
			continue
		}

		ext := filepath.Ext(l.origin)
		hash := utils.Md5(l.origin)
		l.asset = joinOuter(outerPath, outerIsURL, hash+ext)
		if !outerIsURL {
			l.link = filepath.Join(innerPath, hash+ext)
		}
	}
}

func joinOuter(outerPath string, outerIsURL bool, suffix string) string {
	if !outerIsURL {
		return filepath.Join(outerPath, suffix)
	}
	u, err := url.Parse(outerPath)
	if err != nil {
		return filepath.Join(outerPath, suffix)
	}
	u.Path = path.Join(u.Path, suffix)
	return u.String()
}

func (al *assetLinker) createLinks() {
	for _, l := range al.links {
		if l.link == "" {
			continue
		}

		if _, err := os.Lstat(l.link); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			al.pp.LogError("Error checking link %s: %v", l.link, err)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(l.link), 0755); err != nil {
			al.pp.LogError("Failed to create directories for %s: %v", l.link, err)
			continue
		}

		if err := os.Symlink(l.origin, l.link); err != nil {
			al.pp.LogError("Failed to create symlink from %s to %s: %v", l.origin, l.link, err)
			continue
		}
	}
}
