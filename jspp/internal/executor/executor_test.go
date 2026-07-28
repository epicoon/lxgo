package executor_test

import (
	"testing"

	"github.com/epicoon/lxgo/jspp"
	"github.com/epicoon/lxgo/jspp/component"
	"github.com/epicoon/lxgo/jspp/internal/executor"
	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/apptest"
)

func newTestPreprocessor(t *testing.T) jspp.IPreprocessor {
	t.Helper()
	sysPath := t.TempDir()
	app, err := apptest.New(kernel.Dict{
		"Components": kernel.Dict{
			"JSPreprocessor": kernel.Dict{
				"SysPath":  sysPath,
				"MapsPath": sysPath,
				// CorePath deliberately left unset/missing on disk - getCore
				// logs and contributes empty code for that slot, which is
				// fine as long as the executed code doesn't touch `lx`.
			},
		},
	})
	if err != nil {
		t.Fatalf("apptest.New: %v", err)
	}
	if err := component.SetAppComponent(app, "Components.JSPreprocessor"); err != nil {
		t.Fatalf("SetAppComponent: %v", err)
	}
	pp, err := component.AppComponent(app)
	if err != nil {
		t.Fatalf("AppComponent: %v", err)
	}
	return pp
}

func TestExec_Result(t *testing.T) {
	pp := newTestPreprocessor(t)
	exec := executor.Builder().SetPreprocessor(pp).SetCode("return 1+1;").Executor()

	res, err := exec.Exec()
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if res.Fatal() != "" {
		t.Fatalf("expected no fatal error, got %q", res.Fatal())
	}
	if got, want := res.Result(), float64(2); got != want {
		t.Fatalf("Result() = %v (%T), want %v", got, got, want)
	}
}

func TestExec_Fatal_OnThrow(t *testing.T) {
	pp := newTestPreprocessor(t)
	exec := executor.Builder().SetPreprocessor(pp).SetCode("throw new Error('boom');").Executor()

	res, err := exec.Exec()
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if res.Fatal() == "" {
		t.Fatalf("expected a fatal error for a thrown exception")
	}
	if res.Result() != nil {
		t.Fatalf("expected a nil result when the code throws, got %v", res.Result())
	}
}
