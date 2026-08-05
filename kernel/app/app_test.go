package app_test

import (
	"os"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/app"
	"github.com/epicoon/lxgo/kernel/apptest"
)

// runComponent is a kernel.IAppComponent that records whether Run() was
// actually called on it.
type runComponent struct {
	*app.AppComponent
	ran atomic.Bool
}

func (c *runComponent) Name() string { return "RunComponent" }
func (c *runComponent) Run() error   { c.ran.Store(true); return nil }

const shutdownTestTimeout = 2 * time.Second

// TestApp_Run_StartsComponentsAndReturnsOnSIGTERM is a regression test for
// two things at once (kept in one test, not two - Router.Start() registers
// on the process-global http.DefaultServeMux, so a second App in the same
// test binary calling Run() would panic on the resulting duplicate "/"
// registration; there can only be one Run() per process here regardless):
//
//  1. The components loop in Run() used to sit right after the blocking
//     srv.ListenAndServe() call, so it was unreachable during normal
//     operation - no registered component's Run() ever actually ran.
//  2. Run() used to block on srv.ListenAndServe() forever - the only way
//     to stop it was killing the process outright, bypassing Final()
//     entirely (see e.g. lxgo-auth/cmd/default.go's app.Run(); app.Final()
//     - Final() was unreachable in practice). Run() must now return once
//     it receives SIGINT/SIGTERM, after gracefully shutting the HTTP
//     server down.
func TestApp_Run_StartsComponentsAndReturnsOnSIGTERM(t *testing.T) {
	a, err := apptest.New(kernel.Dict{"Port": 0, "Components": kernel.Dict{"Run": kernel.Dict{}}})
	if err != nil {
		t.Fatalf("apptest.New: %v", err)
	}
	c := &runComponent{AppComponent: app.NewAppComponent()}
	if err := app.RegisterComponent(a, c, "run", "Components.Run"); err != nil {
		t.Fatalf("RegisterComponent: %v", err)
	}

	done := make(chan struct{})
	go func() {
		a.Run()
		close(done)
	}()

	// Waiting for the component's Run() is also a reliable sync point for
	// "Run() is now past signal.NotifyContext and into its select" -
	// signal.NotifyContext is the very first thing Run() does, strictly
	// before the components loop, in the same goroutine. Sending SIGTERM
	// any earlier would risk the OS's default disposition instead
	// (untested, but almost certainly process death) if it arrived before
	// that registration actually ran.
	waitFor(t, func() bool { return c.ran.Load() }, "component's Run() was never called")

	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatalf("kill: %v", err)
	}

	select {
	case <-done:
	case <-time.After(shutdownTestTimeout):
		t.Fatal("Run() did not return within the timeout after SIGTERM")
	}
}

// waitFor polls cond every few milliseconds, failing t with msg if it
// hasn't become true within shutdownTestTimeout.
func waitFor(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(shutdownTestTimeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal(msg)
}
