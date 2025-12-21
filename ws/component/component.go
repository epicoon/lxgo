package component

import (
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/epicoon/lxgo/kernel"
	lxApp "github.com/epicoon/lxgo/kernel/app"
	"github.com/epicoon/lxgo/ws"
	"github.com/epicoon/lxgo/ws/internal/src"
)

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * WSServer
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
type WSServer struct {
	*lxApp.AppComponent

	listener net.Listener
	conns    ws.IConnRepo
	router   ws.IRouter
	channels ws.IChannelRepo
	secret   string
	wg       sync.WaitGroup
}

/** @interface */
var _ ws.IWSServer = (*WSServer)(nil)

func SetAppComponent(app kernel.IApp, configKey string) error {
	if app.HasComponent(ws.APP_COMPONENT_KEY) {
		return fmt.Errorf("the application already has component: %s", ws.APP_COMPONENT_KEY)
	}

	wsServer := NewWSServer()
	err := lxApp.InitComponent(wsServer, app, configKey)
	if err != nil {
		return fmt.Errorf("can not init WS-server component: %s", err)
	}

	app.SetComponent(ws.APP_COMPONENT_KEY, wsServer)
	return nil
}

func AppComponent(app kernel.IApp) (*WSServer, error) {
	c := app.Component(ws.APP_COMPONENT_KEY)
	if c == nil {
		return nil, fmt.Errorf("application component '%s' not found", ws.APP_COMPONENT_KEY)
	}

	wsServer, ok := c.(*WSServer)
	if !ok {
		return nil, fmt.Errorf("application component '%s' is not '*WSServer'", ws.APP_COMPONENT_KEY)
	}

	return wsServer, nil
}

/** @constructor */
func NewWSServer() *WSServer {
	s := &WSServer{
		AppComponent: lxApp.NewAppComponent(),
		secret:       src.RandHash(),
	}
	s.conns = src.NewConnRepo(s)
	s.router = src.NewRouter(s)
	s.channels = src.NewChannelRepo(s)
	return s
}

func (s *WSServer) AfterInit() {
	s.channels.Init()
}

func (s *WSServer) Name() string {
	return "WSServer"
}

func (s *WSServer) LogCategory() string {
	return "WSServer"
}

func (pp *WSServer) CConfig() kernel.CAppComponentConfig {
	return NewWSServerConfig
}

func (pp *WSServer) Config() *WSServerConfig {
	return (pp.GetConfig()).(*WSServerConfig)
}

func (s *WSServer) MaxRequestsPerMinute() int {
	return s.Config().MaxRequestsPerMinute
}

func (s *WSServer) MaxConnectionsPerIp() int {
	return s.Config().MaxConnectionsPerIp
}

func (s *WSServer) ReconnectionAllowed() bool {
	return s.Config().ReconnectionAllowed
}

func (s *WSServer) ReconnectionDuration() int {
	return s.Config().ReconnectionDuration
}

func (s *WSServer) DefaultChannelKey() string {
	return s.Config().DefaultChannel.Key
}

func (s *WSServer) DefaultChannelData() map[string]any {
	return s.Config().DefaultChannel.SharedData

	// return map[string]any{}
}

func (s *WSServer) Connections() ws.IConnRepo {
	return s.conns
}

func (s *WSServer) Channels() ws.IChannelRepo {
	return s.channels
}

func (s *WSServer) Router() ws.IRouter {
	return s.router
}

func (s *WSServer) CreateMessage() ws.IMessage {
	return src.NewMessage(s)
}

func (s *WSServer) Start() error {
	addr := fmt.Sprintf("%s:%d", s.Config().Host, s.Config().Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("error creating socket: %w", err)
	}
	s.listener = ln
	log.Printf("WS Server started on %s", addr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept error: %v", err)
			continue
		}
		c := src.NewConnection(s, conn)
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			c.Handle()
		}()
	}
}

func (s *WSServer) Stop() {
	_ = s.listener.Close()
	s.wg.Wait()
	s.conns.Close()
}

func (s *WSServer) LifecycleLog(msg string, params ...any) {
	if s.Config().LifecycleLog {
		s.Log(msg, params...)
	}
}

func (s *WSServer) LifecycleError(msg string, params ...any) {
	if s.Config().LifecycleError {
		s.Log(msg, params...)
	}
}
