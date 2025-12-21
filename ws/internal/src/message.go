package src

import (
	"maps"
	"slices"

	"github.com/epicoon/lxgo/ws"
)

/** @interface ws.IMessage */
type Message struct {
	server ws.IWSServer

	data               any
	dataForConnections map[string]any
	receivers          map[string]bool
	except             map[string]bool
}

var _ ws.IMessage = (*Message)(nil)

/** @condtructor */
func newMessageBase(s ws.IWSServer) *Message {
	return &Message{
		server:             s,
		dataForConnections: make(map[string]any),
		receivers:          make(map[string]bool),
		except:             make(map[string]bool),
	}
}

/** @condtructor */
func NewMessage(s ws.IWSServer) ws.IMessage {
	return newMessageBase(s)
}

func (m *Message) Server() ws.IWSServer {
	return m.server
}

func (m *Message) SetData(d any) ws.IMessage {
	m.data = d
	return m
}

func (m *Message) AddData(d map[string]any) ws.IMessage {
	currD, ok := m.data.(map[string]any)
	if ok {
		maps.Copy(currD, d)
		m.data = currD
	} else {
		d["__data__"] = m.data
		m.data = d
	}
	return m
}

func (m *Message) SetDataForConnection(r ws.IConnection, d any) ws.IMessage {
	m.dataForConnections[r.ID()] = d
	return m
}

func (m *Message) AddDataForConnection(r ws.IConnection, d map[string]any) ws.IMessage {
	if _, exists := m.dataForConnections[r.ID()]; !exists {
		m.dataForConnections[r.ID()] = d
		return m
	}

	currD, ok := m.dataForConnections[r.ID()].(map[string]any)
	if ok {
		maps.Copy(currD, d)
		m.dataForConnections[r.ID()] = currD
	} else {
		d["__data__"] = m.dataForConnections[r.ID()]
		m.dataForConnections[r.ID()] = d
	}
	return m
}

func (m *Message) SetReceiverIds(ids []string) ws.IMessage {
	m.receivers = make(map[string]bool, len(ids))
	for _, id := range ids {
		m.receivers[id] = true
	}
	return m
}

func (m *Message) SetReceiver(r ws.IConnection) ws.IMessage {
	m.receivers = make(map[string]bool, 1)
	m.receivers[r.ID()] = true
	return m
}

func (m *Message) SetReceivers(rr []ws.IConnection) ws.IMessage {
	m.receivers = make(map[string]bool, len(rr))
	for _, r := range rr {
		m.receivers[r.ID()] = true
	}
	return m
}

func (m *Message) AddReceiver(r ws.IConnection) ws.IMessage {
	m.receivers[r.ID()] = true
	return m
}

func (m *Message) AddReceivers(rr []ws.IConnection) ws.IMessage {
	for _, r := range rr {
		m.receivers[r.ID()] = true
	}
	return m
}

func (m *Message) ExceptReceiver(r ws.IConnection) ws.IMessage {
	m.except[r.ID()] = true
	return m
}

func (m *Message) ExceptReceivers(rr []ws.IConnection) ws.IMessage {
	for _, r := range rr {
		m.except[r.ID()] = true
	}
	return m
}

func (m *Message) ReceiverIDs() []string {
	if len(m.receivers) > 0 {
		return slices.Collect(maps.Keys(m.receivers))
	}

	return slices.Collect(maps.Keys(m.server.Connections().GetAll()))
}

func (m *Message) ValidateConnectionID(id string) bool {
	return !m.except[id] && m.server.Connections().Has(id)
}

func (m *Message) PrepareDataForConnection(connID string) any {
	var data any
	if connData, exists := m.dataForConnections[connID]; !exists {
		data = m.data
	} else {
		if m.data == nil {
			data = connData
		} else {
			temp := map[string]any{}
			mData, ok := m.data.(map[string]any)
			if ok {
				maps.Copy(temp, mData)
			} else {
				temp["__data__"] = m.data
			}
			mConnData, ok := connData.(map[string]any)
			if ok {
				maps.Copy(temp, mConnData)
			} else {
				temp["__cData__"] = connData
			}
			data = temp
		}
	}

	return map[string]any{
		"__lxws_message__": data,
	}
}
