package src

import (
	"github.com/epicoon/lxgo/ws"
)

/** @interface ws.IChannelMessage */
type ChannelMessage struct {
	*Message
	channel        ws.IChannel
	sender         string
	returnToSender bool
	private        bool
}

var _ ws.IChannelMessage = (*ChannelMessage)(nil)

/** @condturctor */
func NewChannelMessage(channel ws.IChannel) ws.IChannelMessage {
	return &ChannelMessage{
		Message: newMessageBase(channel.Server()),
		channel: channel,
	}
}

func (m *ChannelMessage) SetSender(id string) ws.IChannelMessage {
	m.sender = id
	return m
}

func (m *ChannelMessage) ReturnToSender(val bool) ws.IChannelMessage {
	if m.sender == "" {
		return m
	}

	m.returnToSender = val
	if m.returnToSender {
		if len(m.receivers) > 0 {
			m.receivers[m.sender] = true
		}
	} else {
		m.except[m.sender] = true
	}
	return m
}

func (m *ChannelMessage) SetPrivate(val bool) ws.IChannelMessage {
	m.private = val
	return m
}

func (m *ChannelMessage) ReceiverIDs() []string {
	if len(m.receivers) == 0 {
		return m.channel.MateIDs()
	}

	var ids []string
	for id := range m.receivers {
		if m.channel.HasID(id) {
			ids = append(ids, id)
		}
	}
	return ids
}

func (m *ChannelMessage) PrepareDataForConnection(connID string) any {
	data := m.Message.PrepareDataForConnection(connID)

	mp, _ := data.(map[string]any)
	mp["data"] = mp["__lxws_message__"]
	delete(mp, "__lxws_message__")

	mp["__lxws_channel__"] = "message"
	mp["channel"] = m.channel.Key()
	mp["from"] = m.sender
	mp["private"] = m.private
	mp["addressed"] = m.receivers[connID]
	return mp
}
