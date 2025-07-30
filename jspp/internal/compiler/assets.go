package compiler

import (
	"github.com/epicoon/lxgo/jspp"
)

const depTypeJS = "js"
const depTypeCSS = "css"
const depTypeModule = "module"

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * IAsset
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

/** @interface cnv.IAsset */
type Asset struct {
	path string
	tp   string
}

var _ jspp.IAsset = (*Asset)(nil)

func (a Asset) Type() string {
	return a.tp
}

func (a Asset) Path() string {
	return a.path
}

func (a Asset) IsJS() bool {
	return a.tp == depTypeJS
}

func (a Asset) IsCSS() bool {
	return a.tp == depTypeCSS
}

func (a Asset) IsModule() bool {
	return a.tp == depTypeModule
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * IAssets
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

/** @interface cnv.IAssets */
type Assets struct {
	list []jspp.IAsset
}

var _ jspp.IAssets = (*Assets)(nil)

func (as *Assets) AddJS(path string) {
	if hasAsset(as, path) {
		return
	}
	as.list = append(as.list, Asset{
		path: path,
		tp:   depTypeJS,
	})
}

func (as *Assets) AddCSS(path string) {
	if hasAsset(as, path) {
		return
	}
	as.list = append(as.list, Asset{
		path: path,
		tp:   depTypeCSS,
	})
}

func (as *Assets) AddModule(name string) {
	if hasAsset(as, name) {
		return
	}
	as.list = append(as.list, Asset{
		path: name,
		tp:   depTypeModule,
	})
}

func (as *Assets) Merge(asset jspp.IAssets) {
	for _, a := range asset.All() {
		addAsset(as, a.Path(), a.Type())
	}
}

func (as *Assets) All() []jspp.IAsset {
	return as.list
}

func addAsset(as jspp.IAssets, path, tp string) {
	switch tp {
	case depTypeJS:
		as.AddJS(path)
	case depTypeCSS:
		as.AddCSS(path)
	case depTypeModule:
		as.AddModule(path)
	}
}

func hasAsset(as *Assets, path string) bool {
	for _, dep := range as.list {
		if dep.Path() == path {
			return true
		}
	}
	return false
}
