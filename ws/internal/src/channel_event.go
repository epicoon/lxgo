package src

import (
	"github.com/epicoon/lxgo/ws"
)

/** @interface ws.IChannelEvent */
type ChannelEvent struct {
	*ChannelMessage
	name    string
	stopped bool
}

var _ ws.IChannelEvent = (*ChannelEvent)(nil)

/** @constructor */
func NewChannelEvent(name string, channel ws.IChannel, initiator ws.IConnection) ws.IChannelEvent {
	return &ChannelEvent{
		ChannelMessage: &ChannelMessage{
			Message: newMessageBase(channel.Server()),
			channel: channel,
			sender:  initiator.ID(),
		},
		name: name,
	}
}

func (e *ChannelEvent) Name() string {
	return e.name
}

func (e *ChannelEvent) Initiator() ws.IConnection {
	return e.server.Connections().Get(e.sender)
}

func (e *ChannelEvent) Stop() {
	e.stopped = true
}

func (e *ChannelEvent) IsStopped() bool {
	return e.stopped
}

func (e *ChannelEvent) PrepareDataForConnection(connID string) any {
	data := e.ChannelMessage.PrepareDataForConnection(connID)

	// ChannelMessage.PrepareDataForConnection already tags this "message" -
	// an event is relayed the same way, just tagged differently and
	// carrying its name.
	mp, _ := data.(map[string]any)
	mp["__lxws_channel__"] = "event"
	mp["event"] = e.name
	return mp
}
