package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/epicoon/lxgo/jspp"
	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/utils"
)

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
	for _, str := range links {
		if str == "" {
			continue
		}
		al.links = append(al.links, &assetLink{
			origin: pf.GetAbsPath(str),
		})
	}
}

func (al *assetLinker) setFromMap(links map[string]string) {
	pf := al.pf
	for key, str := range links {
		if str == "" {
			continue
		}
		al.links = append(al.links, &assetLink{
			key:    key,
			origin: pf.GetAbsPath(str),
		})
	}
}

func (al *assetLinker) defineLinks() {
	pp := al.pp
	innerPath := pp.App().Pathfinder().GetAbsPath(pp.Config().AssetLinksPath.Inner)
	outerPath := pp.Config().AssetLinksPath.Outer
	if outerPath[0] != '/' {
		outerPath = "/" + outerPath
	}

	for _, l := range al.links {
		re := regexp.MustCompile(`^(http:|https:)`)
		if re.MatchString(l.origin) {
			l.asset = l.origin
			continue
		}

		re = regexp.MustCompile(fmt.Sprintf(`^%s`, innerPath))
		if re.MatchString(l.origin) {
			asset, _ := strings.CutPrefix(l.origin, innerPath)
			l.asset = filepath.Join(outerPath, asset)
			continue
		}

		ext := filepath.Ext(l.origin)
		hash := utils.Md5(l.origin)
		l.link = filepath.Join(innerPath, hash+ext)
		l.asset = filepath.Join(outerPath, hash+ext)
	}
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
