// Package elems provides the base jspp.IElement implementation (Element) -
// embed it in your own widget/plugin's Go counterpart, see the jspp
// package doc "An element's own ajax channel" section.
package elems

import (
	"github.com/epicoon/lxgo/jspp"
	"github.com/epicoon/lxgo/kernel"
)

/** @interface jspp.IElement */

// Element is the base jspp.IElement implementation - embed it in your own
// element struct and override AjaxHandlers if it needs its own ajax endpoints.
type Element struct {
	pp  jspp.IPreprocessor
	app kernel.IApp
}

var _ jspp.IElement = (*Element)(nil)

/** @constructor */

// NewElement constructs an uninitialized Element - call Init before use.
func NewElement() *Element {
	return &Element{}
}

// Init binds the element to its owning preprocessor/app.
func (m *Element) Init(pp jspp.IPreprocessor) {
	m.pp = pp
	m.app = pp.App()
}

// App returns the owning application.
func (m *Element) App() kernel.IApp {
	return m.app
}

// Preprocessor returns the owning IPreprocessor.
func (m *Element) Preprocessor() jspp.IPreprocessor {
	return m.pp
}

// AjaxHandlers returns no routes - override it to add the element's own ajax endpoints.
func (m *Element) AjaxHandlers() kernel.HttpResourcesList {
	return make(kernel.HttpResourcesList, 0)
}
