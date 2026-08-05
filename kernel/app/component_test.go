package app_test

import (
	"testing"

	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/app"
	"github.com/epicoon/lxgo/kernel/apptest"
)

// capturingLogger is a kernel.ILogger that just remembers the category of
// the last message written, for asserting on what Log/LogWarning/LogError
// actually reported.
type capturingLogger struct {
	category string
}

func (l *capturingLogger) Log(msg, category string)        { l.category = category }
func (l *capturingLogger) LogWarning(msg, category string) { l.category = category }
func (l *capturingLogger) LogError(msg, category string)   { l.category = category }

// namedComponent embeds *app.AppComponent and overrides LogCategory - the
// shape every real component (session.Storage, jspp's JSPreprocessor, ws's
// WSServer, ...) uses.
type namedComponent struct {
	*app.AppComponent
}

func (c *namedComponent) Name() string        { return "Named" }
func (c *namedComponent) LogCategory() string { return "NamedComponent" }

// TestAppComponent_Log_UsesEmbeddingLogCategory is a regression test:
// AppComponent.Log/LogWarning/LogError called c.LogCategory() where c was
// typed *AppComponent, so the call always resolved to AppComponent's own
// LogCategory() ("AppComponent") - Go has no virtual dispatch through
// embedding, so an override declared on the outer struct was never reached
// unless that struct also redefined Log/LogWarning/LogError itself. A
// component set up the normal way (RegisterComponent/InitComponent) that
// only overrides LogCategory() used to log everything under "AppComponent"
// instead of its own category.
func TestAppComponent_Log_UsesEmbeddingLogCategory(t *testing.T) {
	a, err := apptest.New(kernel.Dict{
		"Components": kernel.Dict{"Named": kernel.Dict{}},
	})
	if err != nil {
		t.Fatalf("apptest.New: %v", err)
	}
	logger := &capturingLogger{}
	a.SetLogger(logger)

	c := &namedComponent{AppComponent: app.NewAppComponent()}
	if err := app.RegisterComponent(a, c, "named", "Components.Named"); err != nil {
		t.Fatalf("RegisterComponent: %v", err)
	}

	c.Log("hello")
	if logger.category != "NamedComponent" {
		t.Fatalf("Log() reported category %q, want %q", logger.category, "NamedComponent")
	}

	c.LogWarning("hello")
	if logger.category != "NamedComponent" {
		t.Fatalf("LogWarning() reported category %q, want %q", logger.category, "NamedComponent")
	}

	c.LogError("hello")
	if logger.category != "NamedComponent" {
		t.Fatalf("LogError() reported category %q, want %q", logger.category, "NamedComponent")
	}
}

// TestAppComponent_Log_FallsBackWithoutInitComponent checks the case where
// a component is used without ever going through
// RegisterComponent/InitComponent (so self was never bound) - it must still
// work, falling back to calling LogCategory() directly on the embedded
// *AppComponent (i.e. behaving exactly as before this fix).
func TestAppComponent_Log_FallsBackWithoutInitComponent(t *testing.T) {
	a, err := apptest.New()
	if err != nil {
		t.Fatalf("apptest.New: %v", err)
	}
	logger := &capturingLogger{}
	a.SetLogger(logger)

	c := &namedComponent{AppComponent: app.NewAppComponent()}
	c.SetApp(a)

	c.Log("hello")
	if logger.category != "AppComponent" {
		t.Fatalf("Log() reported category %q, want %q", logger.category, "AppComponent")
	}
}
