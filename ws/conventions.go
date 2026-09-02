package ws

import (
	"time"

	"github.com/epicoon/lxgo/kernel"
)

// APP_COMPONENT_KEY is the key this component registers itself under via
// kernel.IApp.SetComponent - see AppComponent in the component package.
const APP_COMPONENT_KEY = "lxgo_ws"

// IWSServer is the WS server application component - see
// component.SetAppComponent/component.AppComponent to set it up and get a
// handle to it.
type IWSServer interface {
	kernel.IAppComponent

	// MaxRequestsPerMinute is the per-connection rate limit (0/unset means
	// no limit) - see Components.WSServer.MaxRequestsPerMinute in the README.
	MaxRequestsPerMinute() int
	// MaxConnectionsPerIp is the per-IP connection limit (0/unset means no
	// limit) - see Components.WSServer.MaxConnectionsPerIp in the README.
	MaxConnectionsPerIp() int
	// MaxChannelsPerConnection is the per-connection cap on client-created
	// channels (0/unset means no limit) - see
	// Components.WSServer.MaxChannelsPerConnection in the README.
	MaxChannelsPerConnection() int
	// EmptyChannelTTL is, in seconds, how long a non-proprietary channel may
	// sit empty before IChannelRepo auto-closes it (0/unset disables this
	// entirely) - see Components.WSServer.EmptyChannelTTL in the README.
	EmptyChannelTTL() int
	// AllowedOrigins is the Origin allowlist for the WS handshake -
	// empty/unset means no restriction - see Components.WSServer.AllowedOrigins
	// in the README.
	AllowedOrigins() []string
	// ReconnectionAllowed reports whether a disconnected connection may
	// reconnect (see Components.WSServer.ReconnectionAllowed in the README).
	ReconnectionAllowed() bool
	// ReconnectionDuration is, in milliseconds, how long a disconnected
	// connection stays eligible to reconnect before it's permanently gone -
	// see Components.WSServer.ReconnectionDuration in the README.
	ReconnectionDuration() int
	// DefaultChannelKey is Components.WSServer.DefaultChannel.Key from the
	// config, or "" if no default channel is configured.
	DefaultChannelKey() string
	// DefaultChannelData is Components.WSServer.DefaultChannel.SharedData
	// from the config.
	DefaultChannelData() map[string]any

	// Connections is the server's connection registry.
	Connections() IConnRepo
	// Channels is the server's channel registry.
	Channels() IChannelRepo
	// Router dispatches __lxws_request__ messages to registered HTTP-like
	// resources - see "Using the existing API" in the README.
	Router() IRouter
	// CreateMessage returns a new, empty IMessage bound to this server.
	CreateMessage() IMessage

	// SetChannelValidator sets the component-level gate for channel
	// creation - see ChannelValidator's doc-comment.
	SetChannelValidator(v ChannelValidator)
	// ChannelValidator returns the handler set via SetChannelValidator, or
	// nil if none was set.
	ChannelValidator() ChannelValidator
	// SetChannelCreatedHandler sets the component-level hook that runs for
	// every successfully created channel - see ChannelCreatedHandler's
	// doc-comment.
	SetChannelCreatedHandler(h ChannelCreatedHandler)
	// ChannelCreatedHandler returns the handler set via
	// SetChannelCreatedHandler, or nil if none was set.
	ChannelCreatedHandler() ChannelCreatedHandler

	// Start opens the TCP listener and blocks, accepting connections until
	// Stop is called (or the listener errors) - run it in its own goroutine.
	Start() error
	// Stop closes the listener and waits for in-flight connections and
	// background sweepers to finish.
	Stop()
	// LifecycleLog logs msg if Components.WSServer.LifecycleLog is enabled.
	LifecycleLog(msg string, params ...any)
	// LifecycleError logs msg if Components.WSServer.LifecycleError is enabled.
	LifecycleError(msg string, params ...any)
}

// IConnRepo is the server's connection registry - tracks live connections,
// per-IP/per-minute limits, and disconnected-but-reconnectable ("tombstoned")
// connections.
type IConnRepo interface {
	// Close stops the registry's background sweeper (which permanently
	// drops connections whose reconnection window has expired) and waits
	// for it to finish.
	Close()
	// Add registers c as a live connection.
	Add(c IConnection)
	// RemoveImmediate drops c right away, with no reconnection window.
	RemoveImmediate(c IConnection)
	// MarkDisconnected drops c from the live set but keeps it reconnectable
	// (via Reconnect) until ReconnectionDuration elapses.
	MarkDisconnected(c IConnection)
	// Reconnect looks for a tombstoned connection with the given old ID and,
	// if found (and from the same IP), transfers its state onto conn and
	// reports true; reports false if there's nothing to reconnect to.
	Reconnect(c IConnection, ID string) bool
	// Has reports whether a connection with this ID is currently live.
	Has(ID string) bool
	// Get returns the live connection with this ID, or nil.
	Get(ID string) IConnection
	// GetAll returns every currently live connection, keyed by ID.
	GetAll() map[string]IConnection
	// CheckRequestLimit reports whether c is still within
	// MaxRequestsPerMinute (always true if that limit is 0/unset).
	CheckRequestLimit(c IConnection) bool
	// CheckIPLimit reports whether c's IP is still within
	// MaxConnectionsPerIp (always true if that limit is 0/unset).
	CheckIPLimit(c IConnection) bool
}

// Connection lifecycle statuses - see IConnection.Status.
const (
	// ConnStatusCreated is the status right after construction, before the
	// WS handshake completes.
	ConnStatusCreated = iota
	// ConnStatusConnecting is the status during the handshake/origin check,
	// before the "connect"/"reconnect" action is processed.
	ConnStatusConnecting
	// ConnStatusActive is the status of a normally connected, usable connection.
	ConnStatusActive
	// ConnStatusDisconnected is the status right after a drop that's still
	// within its reconnection window - not yet permanent.
	ConnStatusDisconnected
	// ConnStatusReconnecting is the status of a connection, still finishing
	// a Reconnect(), while its channel memberships are being restored.
	ConnStatusReconnecting
	// ConnStatusClosed is the final status - either an explicit "close"
	// action, or a disconnect whose reconnection window has expired.
	ConnStatusClosed
)

// IConnection is a single WS connection - see internal/src.Connection for
// the implementation.
type IConnection interface {
	// SetID overrides the connection's ID - used by Reconnect to adopt a
	// tombstoned connection's old ID.
	SetID(ID string)
	// SetStatus changes the connection's lifecycle status - see the
	// ConnStatus* constants.
	SetStatus(stat int)
	// SetChannels replaces the connection's per-channel shared-data
	// overlay wholesale - used by Reconnect to restore prior memberships.
	SetChannels(keys map[string]map[string]any)
	// ID returns the connection's current ID.
	ID() string
	// IP returns the connection's remote IP.
	IP() string
	// Status returns the connection's current lifecycle status - see the
	// ConnStatus* constants.
	Status() int
	// SharedData returns the connection's baseline shared data (from its
	// "connect"/"reconnect" action's "shared" field).
	SharedData() map[string]any
	// SharedDataForChannel returns SharedData() layered with whatever
	// per-channel overlay was set for ch (e.g. via enterChannel's sharedData).
	SharedDataForChannel(ch IChannel) map[string]any
	// Channels returns the connection's per-channel shared-data overlays,
	// keyed by channel key - its keys are exactly the channels it's a member of.
	Channels() map[string]map[string]any
	// Handle runs the connection's full lifecycle (handshake, origin check,
	// message loop) - blocks until the connection closes.
	Handle()
	// Send encodes payload as a WS frame of the given type ("text"/"binary"/
	// "close"/"ping"/"pong") and writes it to the socket.
	Send(payload any, typ string, masked bool) error
	// Close tears down the connection - marks it disconnected (reconnectable)
	// or permanently closed depending on how it was asked to close, and
	// leaves every channel it was in.
	Close()
	// Break sends msg as a close frame and closes the connection immediately,
	// with no reconnection window (e.g. on a rate-limit violation).
	Break(msg string)
	// IsChannelMate reports whether this connection is currently a member
	// of ch.
	IsChannelMate(ch IChannel) bool
	// EnterChannel joins ch (subject to ch's ChannelAuthHandler, if any) -
	// message is the same map the triggering action (connect/enterChannel/
	// createChannel) carried, so a password or similar travels through it.
	EnterChannel(ch IChannel, message map[string]any) (bool, string)
	// LeaveChannel leaves ch - a no-op if this connection wasn't a member.
	LeaveChannel(ch IChannel)
	// LeaveAllChannels leaves every channel this connection is currently in.
	LeaveAllChannels()
	// CreatedChannelsCount returns how many channels this connection has
	// created and are still open (see MaxChannelsPerConnection).
	CreatedChannelsCount() int
	// IncrementCreatedChannels records one more successful channel creation
	// by this connection.
	IncrementCreatedChannels()
	// DecrementCreatedChannels records one of this connection's created
	// channels closing (a no-op if the count is already 0).
	DecrementCreatedChannels()
	// SetCreatedChannelsCount overrides the created-channels count directly -
	// used to restore it across a reconnect.
	SetCreatedChannelsCount(n int)
}

// IClient is an outbound WS connection this process opened to a remote WS
// server - see internal/src.Client for the implementation, and
// component.Dial to open one. Unlike IConnection (a server's own view of
// one of the many connections it accepted), there is exactly one known peer
// here - no reconnection window, channel membership, or ConnRepo
// bookkeeping.
//
// It owns the connection's single read loop from the moment it's created,
// dispatching every received message either to whichever Request is
// currently waiting on it or to the onPush callback given at construction -
// there is no exposed Receive(), so nothing outside the implementation can
// contend with that loop for the connection's incoming side.
type IClient interface {
	// Send encodes payload as a WS frame of the given type ("text"/
	// "binary"/"close"/"ping"/"pong") and writes it to the socket, masked -
	// RFC 6455 requires every client->server frame to be masked.
	Send(payload any, typ string) error

	// Request sends a request for route (with the given params, nil meaning
	// none) and blocks for its matching response, or until timeout elapses.
	// The returned Response's Body is already decoded from JSON, exactly
	// once - callers never see the wire-level response frame. This is the
	// same request/response protocol a server's IRouter.Handle answers and
	// a browser client's own request() method uses.
	Request(route string, params map[string]any, timeout time.Duration) (Response, error)

	// Close closes the underlying connection.
	Close() error
}

// Response is a Request's answer, standardized the same way for every
// IClient - Code, Headers and a Body already parsed from JSON, matching
// what a browser client's own request() resolves with.
type Response struct {
	Code    int
	Headers map[string]any
	Body    any
}

// IChannelRepo is the server's channel registry.
type IChannelRepo interface {
	// Init creates the configured DefaultChannel, if any - called once at
	// server startup.
	Init()
	// Close stops the registry's background sweeper (which auto-closes
	// empty non-proprietary channels past EmptyChannelTTL) and waits for it
	// to finish.
	Close()
	// CreateChannel constructs and registers a new channel per builder - see
	// IChannelBuilder's doc-comment for the full contract (key generation,
	// ChannelValidator, ChannelCreatedHandler, MaxChannelsPerConnection).
	// Returns nil and a reason on failure.
	CreateChannel(builder IChannelBuilder) (IChannel, string)
	// Has reports whether a channel with this key currently exists.
	Has(key string) bool
	// Get returns the channel with this key, or nil.
	Get(key string) IChannel
	// Remove drops the channel with this key from the registry (does not
	// notify its members - see IChannel.Close for that).
	Remove(key string)
	// Channels returns every currently registered channel, keyed by key.
	Channels() map[string]IChannel
	// PublicChannels returns every channel with IsPublic() true.
	PublicChannels() []IChannel
}

// IChannel is a named, server-tracked group of connections that receive
// broadcast messages and membership-lifecycle events together - see "Using
// channels" in the README.
type IChannel interface {
	// Server returns the server this channel belongs to.
	Server() IWSServer
	// Key returns the channel's key.
	Key() string
	// SharedData returns the channel's own shared data (set at creation,
	// layered under each member's own SharedDataForChannel).
	SharedData() map[string]any
	// IsPublic reports whether this channel's key is announced to every
	// connection (via connect/reconnect and onChannelCreated) - see "Public
	// channels & discovering what's available" in the README.
	IsPublic() bool
	// IsProprietary reports whether this channel closes when its creator
	// leaves it for good, rather than via the empty-channel sweep - see
	// "Channel closing" in the README.
	IsProprietary() bool
	// CreatorID returns the ID of the connection that created this channel,
	// or "" if it was created by server/application code (a nil Creator()
	// on the IChannelBuilder).
	CreatorID() string
	// MatesData returns every current member's ID and per-channel shared data.
	MatesData() []MateData
	// MateIDs returns every current member's ID.
	MateIDs() []string
	// Has reports whether c is currently a member.
	Has(c IConnection) bool
	// HasID reports whether the connection with this ID is currently a member.
	HasID(id string) bool
	// AddConnection adds conn as a member, broadcasting "mateEntered" (or
	// "mateReconnected" if conn.Status() is ConnStatusReconnecting) to the
	// others already in it.
	AddConnection(conn IConnection)
	// Enter is Has(conn) if already a member, otherwise runs the
	// ChannelAuthHandler (if any) and, if it passes, calls AddConnection.
	Enter(c IConnection, message map[string]any) (bool, string)
	// Leave removes conn as a member, broadcasting "mateLeft"/"mateDisconnected"
	// to the others - or, if conn is this proprietary channel's creator
	// genuinely leaving (not just a temporary disconnect), closes the whole
	// channel instead (ChannelCloseCodeCreatorGone) - see "Channel closing"
	// in the README.
	Leave(c IConnection)
	// Close force-closes the channel: kicks every remaining member (each
	// gets onChannelClosed client-side, with this code), credits back the
	// creator's MaxChannelsPerConnection quota if it's still connected, and
	// removes the channel from the registry. code is relayed to clients
	// as-is - see ChannelCloseCodeServer/ChannelCloseCodeCreatorGone, or
	// pass any other string for an application-defined reason. A no-op if
	// the channel is already closed.
	Close(code string)
	// SetEventHandler sets the per-channel hook that runs for every
	// IChannelEvent triggered in it - see ChannelEventHandler's doc-comment.
	SetEventHandler(handler ChannelEventHandler)
	// EventHandler returns the handler set via SetEventHandler, or nil if
	// none was set.
	EventHandler() ChannelEventHandler
	// SetAuthHandler sets the per-channel hook that gates Enter - see
	// ChannelAuthHandler's doc-comment.
	SetAuthHandler(handler ChannelAuthHandler)
	// AuthHandler returns the handler set via SetAuthHandler, or nil if
	// none was set.
	AuthHandler() ChannelAuthHandler
}

// Standard IChannel.Close reason codes - relayed to every kicked member as
// "code" in the "closed" channel message (see IChannel.Close's doc-comment).
// Application code is free to pass any other string for its own reasons -
// these two are just what this package itself uses.
const (
	ChannelCloseCodeServer      = "100" // explicit application call, or ChannelRepo's empty-channel sweep (see EmptyChannelTTL)
	ChannelCloseCodeCreatorGone = "200" // a proprietary channel's creator left it for good (see IChannelBuilder.Proprietary)
)

// IChannelBuilder bundles the parameters of a channel creation - who's
// creating it (nil Creator() means the server/application code itself, not
// a client), and the same public/sharedData/initData a client's
// createChannel action carries. Passed both to ChannelValidator (to
// approve/deny) and used internally by IChannelRepo.CreateChannel to
// actually construct the channel. An empty Key() means "let the server
// generate one" - the common case for client-initiated creation, avoiding
// any key-collision concern between different clients.
type IChannelBuilder interface {
	// Creator returns the connection creating the channel, or nil if it's
	// being created by server/application code.
	Creator() IConnection
	// SetCreator sets the creating connection (or nil for server/application
	// code) and returns the builder for chaining.
	SetCreator(c IConnection) IChannelBuilder
	// Key returns the channel's key - "" means "let the server generate one".
	Key() string
	// SetKey sets the channel's key and returns the builder for chaining.
	SetKey(key string) IChannelBuilder
	// Public returns whether the channel will be publicly discoverable -
	// see IChannel.IsPublic.
	Public() bool
	// SetPublic sets whether the channel will be publicly discoverable and
	// returns the builder for chaining.
	SetPublic(pub bool) IChannelBuilder
	// Proprietary returns whether the channel will close when its creator
	// leaves - see IChannel.IsProprietary.
	Proprietary() bool
	// SetProprietary sets whether the channel will close when its creator
	// leaves and returns the builder for chaining.
	SetProprietary(prop bool) IChannelBuilder
	// SharedData returns the channel's initial shared data.
	SharedData() map[string]any
	// SetSharedData sets the channel's initial shared data and returns the
	// builder for chaining.
	SetSharedData(data map[string]any) IChannelBuilder
	// InitData returns the creation-time-only data passed to
	// ChannelValidator/ChannelCreatedHandler - never stored on the channel
	// or sent to anyone.
	InitData() map[string]any
	// SetInitData sets the creation-time-only data and returns the builder
	// for chaining.
	SetInitData(data map[string]any) IChannelBuilder
}

// ChannelValidator is the application-supplied, component-level gate for
// channel creation - set via IWSServer.SetChannelValidator, it runs for
// every IChannelRepo.CreateChannel call (client-initiated via createChannel,
// or server-initiated with a nil Creator()) except the automatic
// DefaultChannel bootstrap at startup, which always skips it. nil (the
// default) means "allow all". A false result's reason string is relayed
// back to the client as an explicit createChannel action error.
type ChannelValidator func(builder IChannelBuilder) (bool, string)

// ChannelCreatedHandler is the application-supplied, component-level hook
// set via IWSServer.SetChannelCreatedHandler - it runs once for every
// successfully created channel, including the automatic DefaultChannel
// (which skips ChannelValidator but still goes through this), letting
// application code wire up that channel's own SetEventHandler/
// SetAuthHandler right after creation.
type ChannelCreatedHandler func(channel IChannel, initData map[string]any)

// ChannelAuthHandler decides whether a connection may enter a channel -
// via a client-initiated enterChannel, or the automatic DefaultChannel join
// on connect/reconnect. nil (the default) means "allow all", the same
// convention as ChannelEventHandler. A false result's reason string is
// relayed back to the client as an explicit enterChannel action error (see
// Connection.sendActionError) - there's no silent-denial mode here, since a
// deliberate access decision (unlike a malformed message) always deserves an
// explicit reason.
type ChannelAuthHandler func(conn IConnection, message map[string]any) (bool, string)

// IRouter dispatches __lxws_request__ messages to HTTP-like resources - see
// "Using the existing API" in the README.
type IRouter interface {
	// RegisterResources registers a batch of routes at once.
	RegisterResources(routes kernel.HttpResourcesList)
	// RegisterResource registers a single route.
	RegisterResource(route string, cResource kernel.CHttpResource)
	// Handle runs the resource registered at route with params and returns
	// its response.
	Handle(route string, params map[string]any) kernel.IHttpResponse
}

// MateData is one channel member's ID and per-channel shared data - see
// IChannel.MatesData.
type MateData struct {
	ID   string         `json:"id"`
	Data map[string]any `json:"data"`
}

// IMessage is a piece of data addressed to one or more connections, with
// per-connection data overrides - see SendMessage. IChannelMessage/
// IChannelEvent extend it for channel-scoped traffic.
type IMessage interface {
	// Server returns the server this message belongs to.
	Server() IWSServer
	// SetData sets the message's common data (sent to every receiver that
	// has no per-connection override) and returns the message for chaining.
	SetData(d any) IMessage
	// Data returns the common data set via SetData.
	Data() any
	// AddData merges d into the existing common data (if it's a
	// map[string]any) or, if the existing data isn't a map, nests it under
	// "__data__" alongside d - and returns the message for chaining.
	AddData(d map[string]any) IMessage
	// SetDataForConnection overrides the data sent to conn specifically and
	// returns the message for chaining.
	SetDataForConnection(conn IConnection, d any) IMessage
	// AddDataForConnection merges d into conn's existing per-connection
	// override (creating one if there wasn't any yet) and returns the
	// message for chaining.
	AddDataForConnection(conn IConnection, d map[string]any) IMessage
	// SetReceiverIds restricts delivery to exactly these connection IDs and
	// returns the message for chaining.
	SetReceiverIds(IDs []string) IMessage
	// SetReceiver restricts delivery to exactly this connection and returns
	// the message for chaining.
	SetReceiver(conn IConnection) IMessage
	// SetReceivers restricts delivery to exactly these connections and
	// returns the message for chaining.
	SetReceivers(conns []IConnection) IMessage
	// AddReceiver adds conn to the receiver set and returns the message for chaining.
	AddReceiver(conn IConnection) IMessage
	// AddReceivers adds conns to the receiver set and returns the message for chaining.
	AddReceivers(conns []IConnection) IMessage
	// ExceptReceiver excludes conn from delivery, even if otherwise addressed,
	// and returns the message for chaining.
	ExceptReceiver(conn IConnection) IMessage
	// ExceptReceivers excludes conns from delivery, even if otherwise
	// addressed, and returns the message for chaining.
	ExceptReceivers(conns []IConnection) IMessage
	// ReceiverIDs returns every connection ID this message should be
	// delivered to (every currently connected one, if no receivers were
	// explicitly set).
	ReceiverIDs() []string
	// ValidateConnectionID reports whether id is still a valid delivery
	// target (connected, and not excluded via ExceptReceiver(s)).
	ValidateConnectionID(id string) bool
	// PrepareDataForConnection returns the final payload for connID - its
	// per-connection override merged over the common data, or vice versa.
	PrepareDataForConnection(connID string) any
}

// IChannelMessage is an IMessage scoped to a channel - see
// internal/src.ChannelMessage for the implementation, and "Sending a
// channel message" in the README.
type IChannelMessage interface {
	IMessage
	// SetSender records which connection sent this message and returns it
	// for chaining.
	SetSender(id string) IChannelMessage
	// ReturnToSender controls whether the sender is included among the
	// receivers (a no-op if SetSender was never called) and returns the
	// message for chaining.
	ReturnToSender(val bool) IChannelMessage
	// SetPrivate marks the message as private (delivered only to its
	// explicit receivers, not broadcast) and returns it for chaining.
	SetPrivate(val bool) IChannelMessage
}

// IChannelEvent is a channel message triggered by Channel.trigger() on the
// client - unlike a plain IChannelMessage, application code can intercept
// it server-side (see IChannel.SetEventHandler) to inspect/mutate it, narrow
// its receivers, or Stop() it entirely before it's relayed to anyone.
type IChannelEvent interface {
	IChannelMessage
	// Name returns the event's name, as passed to the client's channel.trigger().
	Name() string
	// Initiator returns the connection that triggered this event.
	Initiator() IConnection
	// Stop suppresses delivery entirely - not even to the sender - once the
	// ChannelEventHandler (if any) returns.
	Stop()
	// IsStopped reports whether Stop was called.
	IsStopped() bool
}

// ChannelEventHandler is the application-supplied hook set via
// IChannel.SetEventHandler - it runs server-side for every IChannelEvent
// triggered in that channel, before (unless Stop() is called) the event is
// relayed via SendMessage. A nil handler (the default) means events are
// just relayed unchanged, same as a plain channel message.
type ChannelEventHandler func(event IChannelEvent)
