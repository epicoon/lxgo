package src

import (
	"maps"
	"slices"
	"sync"

	"github.com/epicoon/lxgo/ws"
)

/** @interface ws.IChannel */
type Channel struct {
	server     ws.IWSServer
	key        string
	sharedData map[string]any

	mu          sync.RWMutex
	connections map[string]bool
}

var _ ws.IChannel = (*Channel)(nil)

/** @constructor */
func NewChannel(s ws.IWSServer, key string, data map[string]any) ws.IChannel {
	return &Channel{
		server:      s,
		key:         key,
		sharedData:  data,
		connections: map[string]bool{},
	}
}

func (c *Channel) Server() ws.IWSServer {
	return c.server
}

func (c *Channel) Key() string {
	return c.key
}

func (c *Channel) SharedData() map[string]any {
	return c.sharedData
}

func (c *Channel) MateIDs() []string {
	return slices.Collect(maps.Keys(c.connections))
}

func (c *Channel) MatesData() []ws.MateData {
	c.mu.Lock()
	defer c.mu.Unlock()
	res := []ws.MateData{}
	for key := range c.connections {
		con := c.server.Connections().Get(key)
		res = append(res, ws.MateData{
			ID:   key,
			Data: con.SharedDataForChannel(c),
		})
	}
	return res
}

func (c *Channel) Has(con ws.IConnection) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, exists := c.connections[con.ID()]
	return exists
}

func (c *Channel) HasID(id string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, exists := c.connections[id]
	return exists
}

func (c *Channel) AddConnection(conn ws.IConnection) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.connections[conn.ID()] = true

	var event string
	if conn.Status() == ws.ConnStatusReconnecting {
		event = "mateReconnected"
	} else {
		event = "mateEntered"
	}

	for id := range c.connections {
		if id == conn.ID() {
			continue
		}
		iCon := c.server.Connections().Get(id)
		iCon.Send(map[string]any{
			"__lxws_channel_event__": event,
			"channel":                c.Key(),
			"id":                     conn.ID(),
			"data":                   conn.SharedDataForChannel(c),
		}, "text", false)
	}
}

func (c *Channel) Enter(conn ws.IConnection, message map[string]any) bool {
	if c.Has(conn) {
		return true
	}

	//TODO auth

	c.AddConnection(conn)

	return true
}

func (c *Channel) Leave(conn ws.IConnection) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.connections, conn.ID())

	var event string
	if conn.Status() == ws.ConnStatusDisconnected {
		event = "mateDisconnected"
	} else {
		event = "mateLeft"
	}

	for key := range c.connections {
		iCon := c.server.Connections().Get(key)
		iCon.Send(map[string]any{
			"__lxws_channel_event__": event,
			"channel":                c.Key(),
			"id":                     conn.ID(),
		}, "text", false)
	}
}
