package plugins

type snippet struct {
	key        string
	params     map[string]any
	html       string
	clientData *snippetConf
}

type snippetConf struct {
	Self map[string]any `json:"self"`
	Data map[string]any `json:"data"`
	Lx   []any          `json:"lx"`
	JS   string         `json:"js"`
}

func newSnippet(key string, params map[string]any) *snippet {
	return &snippet{
		key:        key,
		params:     params,
		clientData: &snippetConf{},
	}
}
