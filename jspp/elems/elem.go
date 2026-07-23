package elems

import (
	"github.com/epicoon/lxgo/jspp"
	"github.com/epicoon/lxgo/kernel"
)

/** @interface jspp.IElement */
type Element struct {
	pp  jspp.IPreprocessor
	app kernel.IApp
}

var _ jspp.IElement = (*Element)(nil)

/** @constructor */
func NewElement() *Element {
	return &Element{}
}

func (m *Element) Init(pp jspp.IPreprocessor) {
	m.pp = pp
	m.app = pp.App()
}

func (m *Element) App() kernel.IApp {
	return m.app
}

func (m *Element) Preprocessor() jspp.IPreprocessor {
	return m.pp
}

func (m *Element) AjaxHandlers() kernel.HttpResourcesList {
	return make(kernel.HttpResourcesList, 0)
}
