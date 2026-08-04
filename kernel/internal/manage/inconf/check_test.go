package inconf

import (
	"strings"
	"testing"

	"github.com/epicoon/lxgo/kernel"
)

// fakeApp is a minimal kernel.IApp stub - only Config/SetConfig do
// anything real, everything else is an unused no-op. A real app.App can't
// be used here: the app package imports this one (for manage:refresh-config/
// manage:inject-config), so importing it back from a test in this package
// would be an import cycle.
type fakeApp struct {
	config kernel.IDict
}

var _ kernel.IApp = (*fakeApp)(nil)

func (a *fakeApp) BaseApp() kernel.IApp                       { return a }
func (a *fakeApp) SetPort(int)                                {}
func (a *fakeApp) ConfigPath() string                         { return "" }
func (a *fakeApp) SetConfig(c kernel.IDict)                   { a.config = c }
func (a *fakeApp) SetConfigParam(string, any)                 {}
func (a *fakeApp) ConfigParam(string) any                     { return nil }
func (a *fakeApp) Config() kernel.IDict                       { return a.config }
func (a *fakeApp) SetComponent(any, kernel.IAppComponent)     {}
func (a *fakeApp) HasComponent(any) bool                      { return false }
func (a *fakeApp) Component(any) kernel.IAppComponent         { return nil }
func (a *fakeApp) SetConnection(kernel.IConnection)           {}
func (a *fakeApp) Pathfinder() kernel.IPathfinder             { return nil }
func (a *fakeApp) DIContainer() kernel.IDIContainer           { return nil }
func (a *fakeApp) Connection() kernel.IConnection             { return nil }
func (a *fakeApp) Router() kernel.IRouter                     { return nil }
func (a *fakeApp) TemplateHolder() kernel.ITemplateHolder     { return nil }
func (a *fakeApp) TemplateRenderer() kernel.ITemplateRenderer { return nil }
func (a *fakeApp) Events() kernel.IEventManager               { return nil }
func (a *fakeApp) Log(string, string)                         {}
func (a *fakeApp) LogWarning(string, string)                  {}
func (a *fakeApp) LogError(string, string)                    {}
func (a *fakeApp) Logger() kernel.ILogger                     { return nil }
func (a *fakeApp) SetLogger(kernel.ILogger)                   {}
func (a *fakeApp) Run()                                       {}
func (a *fakeApp) Final()                                     {}

func TestParseArrayAccess(t *testing.T) {
	cases := []struct {
		in      string
		wantKey string
		wantIdx *int
	}{
		{"Servers[3]", "Servers", intPtr(3)},
		{"Params", "Params", nil},
		{"Bad[abc]", "Bad", nil},
		{"NoClose[3", "NoClose[3", nil},
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			key, idx := parseArrayAccess(tc.in)
			if key != tc.wantKey {
				t.Errorf("key = %q, want %q", key, tc.wantKey)
			}
			if (idx == nil) != (tc.wantIdx == nil) {
				t.Fatalf("idx = %v, want %v", idx, tc.wantIdx)
			}
			if idx != nil && *idx != *tc.wantIdx {
				t.Errorf("idx = %d, want %d", *idx, *tc.wantIdx)
			}
		})
	}
}

func intPtr(i int) *int { return &i }

func TestGetNestedValue(t *testing.T) {
	cfg := kernel.Dict{
		"Port": 8080,
		"Database": kernel.Dict{
			"Host": "localhost",
		},
		"Servers": []any{"a", "b"},
	}

	t.Run("nil_cfg", func(t *testing.T) {
		if _, ok := getNestedValue(nil, "Port"); ok {
			t.Fatal("expected not found for nil cfg")
		}
	})

	t.Run("top_level_key", func(t *testing.T) {
		val, ok := getNestedValue(cfg, "Port")
		if !ok || val != 8080 {
			t.Fatalf("got %v, %v", val, ok)
		}
	})

	t.Run("nested_dict", func(t *testing.T) {
		val, ok := getNestedValue(cfg, "Database.Host")
		if !ok || val != "localhost" {
			t.Fatalf("got %v, %v", val, ok)
		}
	})

	t.Run("array_index", func(t *testing.T) {
		val, ok := getNestedValue(cfg, "Servers[1]")
		if !ok || val != "b" {
			t.Fatalf("got %v, %v", val, ok)
		}
	})

	t.Run("array_index_out_of_range", func(t *testing.T) {
		if _, ok := getNestedValue(cfg, "Servers[5]"); ok {
			t.Fatal("expected not found for out-of-range index")
		}
	})

	t.Run("missing_key", func(t *testing.T) {
		if _, ok := getNestedValue(cfg, "Nope"); ok {
			t.Fatal("expected not found for missing key")
		}
	})

	t.Run("missing_nested_key", func(t *testing.T) {
		if _, ok := getNestedValue(cfg, "Database.Port"); ok {
			t.Fatal("expected not found for missing nested key")
		}
	})

	t.Run("index_into_non_slice", func(t *testing.T) {
		if _, ok := getNestedValue(cfg, "Port[0]"); ok {
			t.Fatal("expected not found when indexing a non-slice value")
		}
	})
}

func newTestApp(cfg kernel.Dict) kernel.IApp {
	return &fakeApp{config: cfg}
}

func TestCheckParams(t *testing.T) {
	a := newTestApp(kernel.Dict{"Port": 8080})

	var report []string
	checkParams(a, map[string]any{"Port": 8080}, &report)
	if len(report) != 1 || !strings.Contains(report[0], "OK") {
		t.Fatalf("expected an 'OK' line for a matching type, got %v", report)
	}

	report = nil
	checkParams(a, map[string]any{"Port": "8080"}, &report)
	if len(report) != 1 || !strings.Contains(report[0], "type mismatch") {
		t.Fatalf("expected a type-mismatch line, got %v", report)
	}

	report = nil
	checkParams(a, map[string]any{"Missing": 1}, &report)
	if len(report) != 1 || !strings.Contains(report[0], "not found") {
		t.Fatalf("expected a not-found line, got %v", report)
	}
}

func TestCheckArrAdd(t *testing.T) {
	a := newTestApp(kernel.Dict{"Servers": []any{"a", "b"}})

	var report []string
	checkArrAdd(a, map[string][]any{"Servers": {"a", "c"}}, &report)
	if len(report) != 2 {
		t.Fatalf("expected 2 lines, got %v", report)
	}
	if !strings.Contains(report[0], "already exists") {
		t.Errorf("expected 'a' to already exist, got %q", report[0])
	}
	if !strings.Contains(report[1], "will be added") {
		t.Errorf("expected 'c' to be reported as addable, got %q", report[1])
	}

	report = nil
	checkArrAdd(a, map[string][]any{"Missing": {"x"}}, &report)
	if len(report) != 1 || !strings.Contains(report[0], "will be created") {
		t.Fatalf("expected a will-be-created line, got %v", report)
	}

	notArrApp := newTestApp(kernel.Dict{"NotAnArray": "scalar"})
	report = nil
	checkArrAdd(notArrApp, map[string][]any{"NotAnArray": {"x"}}, &report)
	if len(report) != 1 || !strings.Contains(report[0], "is not an array") {
		t.Fatalf("expected a not-an-array line, got %v", report)
	}
}

func TestCheckArrRemove(t *testing.T) {
	a := newTestApp(kernel.Dict{"Servers": []any{"a", "b"}})

	var report []string
	checkArrRemove(a, map[string][]any{"Servers": {"a", "z"}}, &report)
	if len(report) != 2 {
		t.Fatalf("expected 2 lines, got %v", report)
	}
	if !strings.Contains(report[0], "will be removed") {
		t.Errorf("expected 'a' to be reported as removable, got %q", report[0])
	}
	if !strings.Contains(report[1], "element not found") {
		t.Errorf("expected 'z' to be reported as absent, got %q", report[1])
	}
}
