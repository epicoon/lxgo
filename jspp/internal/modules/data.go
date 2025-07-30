package modules

import "github.com/epicoon/lxgo/jspp"

/** @interface jspp.IJSModuleData */
type data struct {
	Ename string         `json:"name"`
	Epath string         `json:"path"`
	Edata map[string]any `json:"data,omitempty"`
}

var _ jspp.IJSModuleData = (*data)(nil)

func NewJSModuleData(name, path string) *data {
	return &data{Ename: name, Epath: path}
}

func (d *data) AddData(key string, val any) {
	if d.Edata == nil {
		d.Edata = make(map[string]any, 1)
	}
	d.Edata[key] = val
}

func (d *data) Name() string {
	return d.Ename
}

func (d *data) Path() string {
	return d.Epath
}

func (d *data) Data() map[string]any {
	return d.Edata
}

func (d *data) HasData() bool {
	return len(d.Edata) > 0
}
