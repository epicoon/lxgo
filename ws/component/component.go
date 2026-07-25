package component

import (
	"errors"
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

/** @interface kernel.IAppComponent */
/** @interface ws.IWSServer */

// WSServer is the default ws.IWSServer implementation - see SetAppComponent
// to set one up and AppComponent to get a handle to it.
type WSServer struct {
	*lxApp.AppComponent

	listener net.Listener
	conns    ws.IConnRepo
	router   ws.IRouter
	channels ws.IChannelRepo
	secret   string
	wg       sync.WaitGroup

	channelValidator      ws.ChannelValidator
	channelCreatedHandler ws.ChannelCreatedHandler
}

var _ ws.IWSServer = (*WSServer)(nil)

// SetAppComponent creates a WSServer, configures it from configKey (see
// "Setup component" in the README), and registers it on app under
// ws.APP_COMPONENT_KEY - errors if the app already has that component.
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

// AppComponent returns app's WSServer, previously set up via
// SetAppComponent - errors if there isn't one.
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

// NewWSServer constructs a bare WSServer (connection/channel/router
// registries set up, not yet configured or listening) - normally reached via
// SetAppComponent instead of calling this directly.
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

// Name returns the component's registration name ("WSServer").
func (s *WSServer) Name() string {
	return "WSServer"
}

// LogCategory returns the component's logging category ("WSServer").
func (s *WSServer) LogCategory() string {
	return "WSServer"
}

// CConfig returns the WSServerConfig constructor - see kernel.CAppComponentConfig.
func (pp *WSServer) CConfig() kernel.CAppComponentConfig {
	return NewWSServerConfig
}

// Config returns the server's bound WSServerConfig.
func (pp *WSServer) Config() *WSServerConfig {
	return (pp.GetConfig()).(*WSServerConfig)
}

// MaxRequestsPerMinute is the per-connection rate limit (0/unset means no limit).
func (s *WSServer) MaxRequestsPerMinute() int {
	return s.Config().MaxRequestsPerMinute
}

// MaxConnectionsPerIp is the per-IP connection limit (0/unset means no limit).
func (s *WSServer) MaxConnectionsPerIp() int {
	return s.Config().MaxConnectionsPerIp
}

// MaxChannelsPerConnection is the per-connection cap on client-created
// channels (0/unset means no limit).
func (s *WSServer) MaxChannelsPerConnection() int {
	return s.Config().MaxChannelsPerConnection
}

// EmptyChannelTTL is, in seconds, how long a non-proprietary channel may sit
// empty before it's auto-closed (0/unset disables this entirely).
func (s *WSServer) EmptyChannelTTL() int {
	return s.Config().EmptyChannelTTL
}

// AllowedOrigins is the Origin allowlist for the WS handshake - empty/unset
// means no restriction.
func (s *WSServer) AllowedOrigins() []string {
	return s.Config().AllowedOrigins
}

// SetChannelValidator sets the component-level gate for channel creation.
func (s *WSServer) SetChannelValidator(v ws.ChannelValidator) {
	s.channelValidator = v
}

// ChannelValidator returns the handler set via SetChannelValidator, or nil
// if none was set.
func (s *WSServer) ChannelValidator() ws.ChannelValidator {
	return s.channelValidator
}

// SetChannelCreatedHandler sets the component-level hook that runs for every
// successfully created channel.
func (s *WSServer) SetChannelCreatedHandler(h ws.ChannelCreatedHandler) {
	s.channelCreatedHandler = h
}

// ChannelCreatedHandler returns the handler set via SetChannelCreatedHandler,
// or nil if none was set.
func (s *WSServer) ChannelCreatedHandler() ws.ChannelCreatedHandler {
	return s.channelCreatedHandler
}

// ReconnectionAllowed reports whether disconnected connections may reconnect.
func (s *WSServer) ReconnectionAllowed() bool {
	return s.Config().ReconnectionAllowed
}

// ReconnectionDuration is, in milliseconds, how long a disconnected
// connection stays eligible to reconnect before it's permanently gone.
func (s *WSServer) ReconnectionDuration() int {
	return s.Config().ReconnectionDuration
}

// DefaultChannelKey is Components.WSServer.DefaultChannel.Key from the
// config, or "" if no default channel is configured.
func (s *WSServer) DefaultChannelKey() string {
	return s.Config().DefaultChannel.Key
}

// DefaultChannelData is Components.WSServer.DefaultChannel.SharedData from
// the config.
func (s *WSServer) DefaultChannelData() map[string]any {
	return s.Config().DefaultChannel.SharedData
}

// Connections returns the server's connection registry.
func (s *WSServer) Connections() ws.IConnRepo {
	return s.conns
}

// Channels returns the server's channel registry.
func (s *WSServer) Channels() ws.IChannelRepo {
	return s.channels
}

// Router returns the server's HTTP-like request router.
func (s *WSServer) Router() ws.IRouter {
	return s.router
}

// CreateMessage returns a new, empty ws.IMessage bound to this server.
func (s *WSServer) CreateMessage() ws.IMessage {
	return src.NewMessage(s)
}

// Start opens the TCP listener and blocks, accepting connections until Stop
// is called (or the listener errors) - run it in its own goroutine.
func (s *WSServer) Start() error {
	// Deliberately not in AfterInit(): that runs synchronously inside
	// SetAppComponent, before application code has a chance to call
	// SetChannelValidator/SetChannelCreatedHandler on the *WSServer it gets
	// back from AppComponent() - initializing channels (including
	// DefaultChannel) here instead means ChannelCreatedHandler is guaranteed
	// to already be registered by the time it fires for it.
	s.channels.Init()

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
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
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

// Stop closes the listener and waits for in-flight connections and
// background sweepers to finish.
func (s *WSServer) Stop() {
	if err := s.listener.Close(); err != nil {
		log.Printf("listener close error: %v", err)
	}
	s.wg.Wait()
	s.conns.Close()
	s.channels.Close()
}

// LifecycleLog logs msg if Components.WSServer.LifecycleLog is enabled.
func (s *WSServer) LifecycleLog(msg string, params ...any) {
	if s.Config().LifecycleLog {
		s.Log(msg, params...)
	}
}

// LifecycleError logs msg if Components.WSServer.LifecycleError is enabled.
func (s *WSServer) LifecycleError(msg string, params ...any) {
	if s.Config().LifecycleError {
		s.Log(msg, params...)
	}
}
