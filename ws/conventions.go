package ws

import "github.com/epicoon/lxgo/kernel"

const APP_COMPONENT_KEY = "lxgo_ws"

type IWSServer interface {
	kernel.IAppComponent
	MaxRequestsPerMinute() int
	MaxConnectionsPerIp() int
	MaxChannelsPerConnection() int
	EmptyChannelTTL() int
	AllowedOrigins() []string
	ReconnectionAllowed() bool
	ReconnectionDuration() int
	DefaultChannelKey() string
	DefaultChannelData() map[string]any

	Connections() IConnRepo
	Channels() IChannelRepo
	Router() IRouter
	CreateMessage() IMessage

	SetChannelValidator(v ChannelValidator)
	ChannelValidator() ChannelValidator
	SetChannelCreatedHandler(h ChannelCreatedHandler)
	ChannelCreatedHandler() ChannelCreatedHandler

	Start() error
	Stop()
	LifecycleLog(msg string, params ...any)
	LifecycleError(msg string, params ...any)
}

type IConnRepo interface {
	Close()
	Add(c IConnection)
	RemoveImmediate(c IConnection)
	MarkDisconnected(c IConnection)
	Reconnect(c IConnection, ID string) bool
	Has(id string) bool
	Get(id string) IConnection
	GetAll() map[string]IConnection
	CheckRequestLimit(c IConnection) bool
	CheckIPLimit(c IConnection) bool
}

const (
	ConnStatusCreated = iota
	ConnStatusConnecting
	ConnStatusActive
	ConnStatusDisconnected
	ConnStatusReconnecting
	ConnStatusClosed
)

type IConnection interface {
	SetID(ID string)
	SetStatus(stat int)
	SetChannels(keys map[string]map[string]any)
	ID() string
	IP() string
	Status() int
	SharedData() map[string]any
	SharedDataForChannel(ch IChannel) map[string]any
	Channels() map[string]map[string]any
	Handle()
	Send(payload any, typ string, masked bool) error
	Close()
	Break(msg string)
	IsChannelMate(ch IChannel) bool
	EnterChannel(ch IChannel, message map[string]any) (bool, string)
	LeaveChannel(ch IChannel)
	LeaveAllChannels()
	CreatedChannelsCount() int
	IncrementCreatedChannels()
	DecrementCreatedChannels()
	SetCreatedChannelsCount(n int)
}

type IChannelRepo interface {
	Init()
	Close()
	CreateChannel(builder IChannelBuilder) (IChannel, string)
	Has(key string) bool
	Get(key string) IChannel
	Remove(key string)
	Channels() map[string]IChannel
	PublicChannels() []IChannel
}

type IChannel interface {
	Server() IWSServer
	Key() string
	SharedData() map[string]any
	IsPublic() bool
	IsProprietary() bool
	CreatorID() string
	MatesData() []MateData
	MateIDs() []string
	Has(c IConnection) bool
	HasID(id string) bool
	AddConnection(conn IConnection)
	Enter(c IConnection, message map[string]any) (bool, string)
	Leave(c IConnection)
	Close(code string)
	SetEventHandler(handler ChannelEventHandler)
	EventHandler() ChannelEventHandler
	SetAuthHandler(handler ChannelAuthHandler)
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
	Creator() IConnection
	SetCreator(c IConnection) IChannelBuilder
	Key() string
	SetKey(key string) IChannelBuilder
	Public() bool
	SetPublic(pub bool) IChannelBuilder
	Proprietary() bool
	SetProprietary(prop bool) IChannelBuilder
	SharedData() map[string]any
	SetSharedData(data map[string]any) IChannelBuilder
	InitData() map[string]any
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

type IRouter interface {
	RegisterResources(routes kernel.HttpResourcesList)
	RegisterResource(route string, cResource kernel.CHttpResource)
	Handle(route string, params map[string]any) kernel.IHttpResponse
}

type MateData struct {
	ID   string         `json:"id"`
	Data map[string]any `json:"data"`
}

type IMessage interface {
	Server() IWSServer
	SetData(d any) IMessage
	Data() any
	AddData(d map[string]any) IMessage
	SetDataForConnection(r IConnection, d any) IMessage
	AddDataForConnection(r IConnection, d map[string]any) IMessage
	SetReceiverIds(ids []string) IMessage
	SetReceiver(r IConnection) IMessage
	SetReceivers(rr []IConnection) IMessage
	AddReceiver(r IConnection) IMessage
	AddReceivers(rr []IConnection) IMessage
	ExceptReceiver(r IConnection) IMessage
	ExceptReceivers(rr []IConnection) IMessage
	ReceiverIDs() []string
	ValidateConnectionID(id string) bool
	PrepareDataForConnection(connID string) any
}

type IChannelMessage interface {
	IMessage
	SetSender(id string) IChannelMessage
	ReturnToSender(val bool) IChannelMessage
	SetPrivate(val bool) IChannelMessage
}

// IChannelEvent is a channel message triggered by Channel.trigger() on the
// client - unlike a plain IChannelMessage, application code can intercept
// it server-side (see IChannel.SetEventHandler) to inspect/mutate it, narrow
// its receivers, or Stop() it entirely before it's relayed to anyone.
type IChannelEvent interface {
	IChannelMessage
	Name() string
	Initiator() IConnection
	Stop()
	IsStopped() bool
}

// ChannelEventHandler is the application-supplied hook set via
// IChannel.SetEventHandler - it runs server-side for every IChannelEvent
// triggered in that channel, before (unless Stop() is called) the event is
// relayed via SendMessage. A nil handler (the default) means events are
// just relayed unchanged, same as a plain channel message.
type ChannelEventHandler func(event IChannelEvent)
