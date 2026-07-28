package src

import (
	"maps"

	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/ws"
)

// fakeConnection is a lightweight ws.IConnection stand-in for tests that
// exercise Channel/Message/repo logic without a real socket - Send just
// records what would have gone out, instead of encoding a WS frame.
type fakeConnection struct {
	server ws.IWSServer

	id              string
	ip              string
	status          int
	sharedData      map[string]any
	channels        map[string]map[string]any
	createdChannels int

	sendErr error
	sent    []fakeSent
}

type fakeSent struct {
	payload any
	typ     string
	masked  bool
}

var _ ws.IConnection = (*fakeConnection)(nil)

func newFakeConnection(s ws.IWSServer, id string) *fakeConnection {
	return &fakeConnection{
		server:     s,
		id:         id,
		ip:         "127.0.0.1",
		sharedData: map[string]any{},
		channels:   map[string]map[string]any{},
	}
}

func (c *fakeConnection) SetID(ID string)            { c.id = ID }
func (c *fakeConnection) SetStatus(s int)            { c.status = s }
func (c *fakeConnection) ID() string                 { return c.id }
func (c *fakeConnection) IP() string                 { return c.ip }
func (c *fakeConnection) Status() int                { return c.status }
func (c *fakeConnection) SharedData() map[string]any { return c.sharedData }

func (c *fakeConnection) SetChannels(keys map[string]map[string]any) { c.channels = keys }
func (c *fakeConnection) Channels() map[string]map[string]any        { return c.channels }

func (c *fakeConnection) SharedDataForChannel(ch ws.IChannel) map[string]any {
	overlay, exists := c.channels[ch.Key()]
	if !exists {
		return c.sharedData
	}
	out := map[string]any{}
	maps.Copy(out, c.sharedData)
	maps.Copy(out, overlay)
	return out
}

func (c *fakeConnection) Handle() {}

func (c *fakeConnection) Send(payload any, typ string, masked bool) error {
	c.sent = append(c.sent, fakeSent{payload: payload, typ: typ, masked: masked})
	return c.sendErr
}

func (c *fakeConnection) Close()           {}
func (c *fakeConnection) Break(msg string) {}

func (c *fakeConnection) IsChannelMate(ch ws.IChannel) bool { return ch.Has(c) }

func (c *fakeConnection) EnterChannel(ch ws.IChannel, message map[string]any) (bool, string) {
	ok, reason := ch.Enter(c, message)
	if !ok {
		return false, reason
	}
	c.channels[ch.Key()] = map[string]any{}
	if raw, ok := message["sharedData"]; ok {
		if m, ok := raw.(map[string]any); ok {
			c.channels[ch.Key()] = m
		}
	}
	return true, ""
}

func (c *fakeConnection) LeaveChannel(ch ws.IChannel) {
	if !c.IsChannelMate(ch) {
		return
	}
	delete(c.channels, ch.Key())
	ch.Leave(c)
}

func (c *fakeConnection) LeaveAllChannels() {
	for key := range c.channels {
		if ch := c.server.Channels().Get(key); ch != nil {
			ch.Leave(c)
		}
	}
	c.channels = map[string]map[string]any{}
}

func (c *fakeConnection) CreatedChannelsCount() int { return c.createdChannels }
func (c *fakeConnection) IncrementCreatedChannels() { c.createdChannels++ }
func (c *fakeConnection) DecrementCreatedChannels() {
	if c.createdChannels > 0 {
		c.createdChannels--
	}
}
func (c *fakeConnection) SetCreatedChannelsCount(n int) { c.createdChannels = n }

// fakeServer is a minimal ws.IWSServer for tests - it wires up the real
// ConnRepo/ChannelRepo/Router (the code under test), just without a real
// kernel.IApp/component/config layer around them. App() is nil unless a
// test explicitly needs one (see withApp), since Channel/Message/repo logic
// never touches it - only Router.Handle does.
type fakeServer struct {
	maxRequestsPerMinute     int
	maxConnectionsPerIp      int
	maxChannelsPerConnection int
	emptyChannelTTL          int
	allowedOrigins           []string
	reconnectionAllowed      bool
	reconnectionDuration     int
	defaultChannelKey        string
	defaultChannelData       map[string]any

	conns    ws.IConnRepo
	channels ws.IChannelRepo
	router   ws.IRouter

	channelValidator      ws.ChannelValidator
	channelCreatedHandler ws.ChannelCreatedHandler

	app kernel.IApp
}

var _ ws.IWSServer = (*fakeServer)(nil)

// fakeServerOption configures a fakeServer before its background sweepers
// start - config fields must never be mutated after newFakeServer returns,
// since NewConnRepo/NewChannelRepo's sweeper goroutines read them
// concurrently (see the ConnRepo.GetAll/Reconnect races this same audit
// found and fixed in the production code - the harness must not reintroduce
// the same class of bug against itself).
type fakeServerOption func(*fakeServer)

func withEmptyChannelTTL(seconds int) fakeServerOption {
	return func(s *fakeServer) { s.emptyChannelTTL = seconds }
}

func withDefaultChannel(key string, data map[string]any) fakeServerOption {
	return func(s *fakeServer) { s.defaultChannelKey = key; s.defaultChannelData = data }
}

func withMaxChannelsPerConnection(n int) fakeServerOption {
	return func(s *fakeServer) { s.maxChannelsPerConnection = n }
}

func withMaxConnectionsPerIp(n int) fakeServerOption {
	return func(s *fakeServer) { s.maxConnectionsPerIp = n }
}

func withMaxRequestsPerMinute(n int) fakeServerOption {
	return func(s *fakeServer) { s.maxRequestsPerMinute = n }
}

func withReconnectionDuration(ms int) fakeServerOption {
	return func(s *fakeServer) { s.reconnectionDuration = ms }
}

func withAllowedOrigins(origins ...string) fakeServerOption {
	return func(s *fakeServer) { s.allowedOrigins = origins }
}

func newFakeServer(opts ...fakeServerOption) *fakeServer {
	s := &fakeServer{defaultChannelData: map[string]any{}}
	for _, opt := range opts {
		opt(s)
	}
	s.conns = NewConnRepo(s)
	s.channels = NewChannelRepo(s)
	s.router = NewRouter(s)
	return s
}

// close stops the background sweepers started by NewConnRepo/NewChannelRepo -
// call via t.Cleanup in every test that builds a fakeServer.
func (s *fakeServer) close() {
	s.conns.Close()
	s.channels.Close()
}

// kernel.IAppComponent
func (s *fakeServer) SetApp(app kernel.IApp)                 { s.app = app }
func (s *fakeServer) SetConfig(_ kernel.IAppComponentConfig) {}
func (s *fakeServer) GetConfig() kernel.IAppComponentConfig  { return nil }
func (s *fakeServer) Name() string                           { return "fakeWSServer" }
func (s *fakeServer) App() kernel.IApp                       { return s.app }
func (s *fakeServer) CConfig() kernel.CAppComponentConfig    { return nil }
func (s *fakeServer) AfterInit()                             {}
func (s *fakeServer) LogCategory() string                    { return "fakeWSServer" }
func (s *fakeServer) Log(msg string, params ...any)          {}
func (s *fakeServer) LogWarning(msg string, params ...any)   {}
func (s *fakeServer) LogError(msg string, params ...any)     {}
func (s *fakeServer) Run() error                             { return nil }
func (s *fakeServer) Final() error                           { return nil }

// ws.IWSServer
func (s *fakeServer) MaxRequestsPerMinute() int          { return s.maxRequestsPerMinute }
func (s *fakeServer) MaxConnectionsPerIp() int           { return s.maxConnectionsPerIp }
func (s *fakeServer) MaxChannelsPerConnection() int      { return s.maxChannelsPerConnection }
func (s *fakeServer) EmptyChannelTTL() int               { return s.emptyChannelTTL }
func (s *fakeServer) AllowedOrigins() []string           { return s.allowedOrigins }
func (s *fakeServer) ReconnectionAllowed() bool          { return s.reconnectionAllowed }
func (s *fakeServer) ReconnectionDuration() int          { return s.reconnectionDuration }
func (s *fakeServer) DefaultChannelKey() string          { return s.defaultChannelKey }
func (s *fakeServer) DefaultChannelData() map[string]any { return s.defaultChannelData }
func (s *fakeServer) Connections() ws.IConnRepo          { return s.conns }
func (s *fakeServer) Channels() ws.IChannelRepo          { return s.channels }
func (s *fakeServer) Router() ws.IRouter                 { return s.router }
func (s *fakeServer) CreateMessage() ws.IMessage         { return NewMessage(s) }

func (s *fakeServer) SetChannelValidator(v ws.ChannelValidator) { s.channelValidator = v }
func (s *fakeServer) ChannelValidator() ws.ChannelValidator     { return s.channelValidator }
func (s *fakeServer) SetChannelCreatedHandler(h ws.ChannelCreatedHandler) {
	s.channelCreatedHandler = h
}
func (s *fakeServer) ChannelCreatedHandler() ws.ChannelCreatedHandler { return s.channelCreatedHandler }

func (s *fakeServer) Start() error { return nil }
func (s *fakeServer) Stop()        {}

func (s *fakeServer) LifecycleLog(msg string, params ...any)   {}
func (s *fakeServer) LifecycleError(msg string, params ...any) {}
