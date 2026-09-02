package component

import (
	"github.com/epicoon/lxgo/ws"
	"github.com/epicoon/lxgo/ws/internal/src"
)

// Dial opens addr ("host:port") as an outbound WS connection to a remote WS
// server, at path (e.g. "/"), and wraps it as a ws.IClient - see IClient's
// doc-comment for what onPush/onDropped are for (either may be nil). This
// is the public entry point for internal/src.Client.
func Dial(addr, path string, onPush func(msg any), onDropped func()) (ws.IClient, error) {
	conn, err := src.DialClientConnection(addr, path)
	if err != nil {
		return nil, err
	}
	return src.NewClient(conn, onPush, onDropped), nil
}
