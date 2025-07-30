package kernel

func (d Dict) ToMap() map[string]any {
	return map[string]any(d)
}

func (c Config) ToMap() map[string]any {
	return map[string]any(c)
}

func (c Config) ToDict() Dict {
	return Dict(c)
}

/** @interface IData */
type Data struct {
	list map[string]any
}

/** @constructor */
func NewData(data map[string]any) *Data {
	return &Data{list: data}
}

/** @constructor */
func NewEmptyData() *Data {
	return &Data{}
}

func (d *Data) Set(key string, val any) {
	if d.list == nil {
		d.list = make(map[string]any)
	}
	d.list[key] = val
}

func (d *Data) Get(key string) any {
	val, ok := d.list[key]
	if ok {
		return val
	}
	return nil
}

func (d *Data) Has(key string) bool {
	_, ok := d.list[key]
	return ok
}
