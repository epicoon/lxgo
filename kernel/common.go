package kernel

var _ IDict = (*Dict)(nil)

// Set stores val under key.
func (d Dict) Set(key string, val any) {
	d[key] = val
}

// Get returns the value stored under key, or nil.
func (d Dict) Get(key string) any {
	if val, ok := d[key]; ok {
		return val
	}
	return nil
}

// Has reports whether key is set.
func (d Dict) Has(key string) bool {
	_, ok := d[key]
	return ok
}

// ToMap returns d as a plain map[string]any.
func (d Dict) ToMap() map[string]any {
	return map[string]any(d)
}
