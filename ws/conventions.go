package ws

import "github.com/epicoon/lxgo/kernel"

const APP_COMPONENT_KEY = "lxgo_ws"

type IWSServer interface {
	kernel.IAppComponent
	MaxRequestsPerMinute() int
	MaxConnectionsPerIp() int
	ReconnectionAllowed() bool
	ReconnectionDuration() int
	DefaultChannelKey() string
	DefaultChannelData() map[string]any

	Connections() IConnRepo
	Channels() IChannelRepo
	Router() IRouter
	CreateMessage() IMessage

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
	EnterChannel(ch IChannel, message map[string]any) bool
	LeaveChannel(ch IChannel)
	LeaveAllChannels()
}

type IChannelRepo interface {
	Init()
	CreateChannel(key string, data map[string]any) IChannel
	Has(key string) bool
	Get(key string) IChannel
	Remove(key string)
}

type IChannel interface {
	Server() IWSServer
	Key() string
	SharedData() map[string]any
	MatesData() []MateData
	MateIDs() []string
	Has(c IConnection) bool
	HasID(id string) bool
	AddConnection(conn IConnection)
	Enter(c IConnection, message map[string]any) bool
	Leave(c IConnection)
}

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
