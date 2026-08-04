// Package trigger implements the "trigger" manage-socket command: fires a
// named app event (app.Events().Trigger) from outside the running process,
// optionally with a payload - see ManageCommand's "trigger" action in
// lxgo-kernel/cmd/manage.go.
package trigger

import (
	"net"
	"strings"

	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/internal/manage/inconf"
)

// Run parses cmdParams (the "event=NAME" and optional "params=..." pairs
// sent after "trigger&&" over the manage socket - see ManageCommand.trigger/
// prepareMsg) and fires the named event on app, replying to conn with the
// outcome.
func Run(app kernel.IApp, conn net.Conn, cmdParams []string) {
	var eventName string
	var payload map[string]any

	for _, str := range cmdParams {
		pair := strings.SplitN(str, "=", 2)
		if len(pair) < 2 {
			continue
		}
		key := strings.TrimSpace(pair[0])
		val := strings.TrimSpace(pair[1])

		switch key {
		case "event":
			eventName = val
		case "params":
			errList := make([]string, 0)
			payload = inconf.ParseParamList(val, &errList)
			if len(errList) > 0 {
				conn.Write([]byte("Syntax error:\n" + strings.Join(errList, "\n") + "\n"))
				return
			}
		}
	}

	if eventName == "" {
		conn.Write([]byte("event name is required\n"))
		return
	}

	if len(payload) > 0 {
		app.Events().Trigger(eventName, kernel.Dict(payload))
	} else {
		app.Events().Trigger(eventName)
	}

	conn.Write([]byte("Done\n"))
}
