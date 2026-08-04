package trigger_test

import (
	"io"
	"net"
	"strings"
	"testing"

	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/apptest"
	"github.com/epicoon/lxgo/kernel/internal/manage/trigger"
)

// runTrigger runs trigger.Run against a net.Pipe() and returns whatever it
// wrote back - net.Pipe is synchronous, so Run (the writer) has to run in
// its own goroutine while the test reads the other end.
func runTrigger(t *testing.T, app kernel.IApp, cmdParams []string) string {
	t.Helper()
	server, client := net.Pipe()

	go func() {
		trigger.Run(app, server, cmdParams)
		server.Close()
	}()

	buf, err := io.ReadAll(client)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	return string(buf)
}

func TestRun_FiresEventWithNoPayload(t *testing.T) {
	app, err := apptest.New()
	if err != nil {
		t.Fatalf("apptest.New: %v", err)
	}

	fired := false
	app.Events().Subscribe("my-event", func(e kernel.IEvent) {
		fired = true
	})

	resp := runTrigger(t, app, []string{"event=my-event"})
	if !fired {
		t.Fatal("expected the event to fire")
	}
	if resp != "Done\n" {
		t.Fatalf("response = %q, want %q", resp, "Done\n")
	}
}

// TestRun_FiresEventWithPayload checks the --params syntax (shared with
// inject-config via inconf.ParseParamList) reaches the handler as the
// event's payload, with the same type coercion (int/bool/quoted string).
func TestRun_FiresEventWithPayload(t *testing.T) {
	app, err := apptest.New()
	if err != nil {
		t.Fatalf("apptest.New: %v", err)
	}

	var got kernel.IDict
	app.Events().Subscribe("my-event", func(e kernel.IEvent) {
		got = e.Payload()
	})

	runTrigger(t, app, []string{"event=my-event", "params=count:42,name:'Al'"})

	if got == nil {
		t.Fatal("expected a payload")
	}
	if got.Get("count") != 42 {
		t.Fatalf("count = %#v, want 42", got.Get("count"))
	}
	if got.Get("name") != "Al" {
		t.Fatalf("name = %#v, want Al", got.Get("name"))
	}
}

func TestRun_MissingEventName(t *testing.T) {
	app, err := apptest.New()
	if err != nil {
		t.Fatalf("apptest.New: %v", err)
	}

	resp := runTrigger(t, app, []string{"params=x:1"})
	if resp != "event name is required\n" {
		t.Fatalf("response = %q", resp)
	}
}

func TestRun_InvalidParamsSyntax(t *testing.T) {
	app, err := apptest.New()
	if err != nil {
		t.Fatalf("apptest.New: %v", err)
	}

	resp := runTrigger(t, app, []string{"event=my-event", "params=not-valid"})
	if !strings.Contains(resp, "Syntax error") {
		t.Fatalf("response = %q, want a syntax error", resp)
	}
}
