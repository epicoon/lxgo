package app

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/config"
	"github.com/epicoon/lxgo/kernel/internal/manage/inconf"
	"github.com/epicoon/lxgo/kernel/internal/manage/reconf"
)

type manageSocket struct {
	app        kernel.IApp
	socketPath string
	listener   net.Listener
	wg         sync.WaitGroup
	stopCh     chan struct{}
}

func newManageSocket(app kernel.IApp) (*manageSocket, error) {
	p, err := config.GetParam[string](app.Config(), "ManageSocket")
	if err != nil {
		return nil, err
	}

	m := &manageSocket{
		app:    app,
		stopCh: make(chan struct{}),
	}
	m.socketPath = app.Pathfinder().GetAbsPath(p)

	return m, nil
}

func (m *manageSocket) Run() error {
	os.Remove(m.socketPath)

	dir := filepath.Dir(m.socketPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create socket dir: %v", err)
	}

	ln, err := net.Listen("unix", m.socketPath)
	if err != nil {
		return fmt.Errorf("manage socket error: %v", err)
	}
	m.listener = ln

	m.app.Log(fmt.Sprintf("Manage socket started at %s", m.socketPath), "ManageSocket")

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		for {
			conn, err := m.listener.Accept()
			if err != nil {
				select {
				case <-m.stopCh:
					return
				default:
					m.app.LogError(fmt.Sprintf("accept error: %v", err), "ManageSocket")
					continue
				}
			}
			m.wg.Add(1)
			go func(c net.Conn) {
				defer m.wg.Done()
				m.handleConn(c)
			}(conn)
		}
	}()

	return nil
}

func (m *manageSocket) handleConn(conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		msg := strings.TrimSpace(scanner.Text())
		cmdList := strings.Split(msg, "&&")
		cmd := cmdList[0]

		switch cmd {
		case "status":
			conn.Write([]byte("ok\n"))
		case "reconf":
			test := false
			if len(cmdList) > 1 && cmdList[1] == "t=true" {
				test = true
			}
			reconf.Run(m.app, conn, test)
		case "inconf":
			inconf.Run(m.app, conn, cmdList[1:])
		case "trigger":
			//TODO trigger custom events
			conn.Write([]byte("Not implemented yet\n"))
		default:
			conn.Write([]byte("unknown command\n"))
		}
	}
}

func (m *manageSocket) Final() {
	close(m.stopCh)
	if m.listener != nil {
		m.listener.Close()
	}
	m.wg.Wait()
	os.Remove(m.socketPath)
	m.app.Log("Manage socket stopped", "ManageSocket")
}
