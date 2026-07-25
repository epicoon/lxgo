package kernel

// ToMap returns d as a plain map[string]any.
func (d Dict) ToMap() map[string]any {
	return map[string]any(d)
}

// ToMap returns c as a plain map[string]any.
func (c Config) ToMap() map[string]any {
	return map[string]any(c)
}

// ToDict returns c as a Dict.
func (c Config) ToDict() Dict {
	return Dict(c)
}

/** @interface IData */

// Data is the default IData implementation.
type Data struct {
	list map[string]any
}

var _ IData = (*Data)(nil)

/** @constructor */

// NewData constructs a Data pre-populated with data.
func NewData(data map[string]any) *Data {
	return &Data{list: data}
}

/** @constructor */

// NewEmptyData constructs an empty Data.
func NewEmptyData() *Data {
	return &Data{}
}

// Set stores val under key.
func (d *Data) Set(key string, val any) {
	if d.list == nil {
		d.list = make(map[string]any)
	}
	d.list[key] = val
}

// Get returns the value stored under key, or nil.
func (d *Data) Get(key string) any {
	val, ok := d.list[key]
	if ok {
		return val
	}
	return nil
}

// Has reports whether key is set.
func (d *Data) Has(key string) bool {
	_, ok := d.list[key]
	return ok
}
