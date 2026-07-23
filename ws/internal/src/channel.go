package src

import (
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/epicoon/lxgo/ws"
)

/** @interface ws.IChannel */
type Channel struct {
	server      ws.IWSServer
	key         string
	sharedData  map[string]any
	public      bool
	proprietary bool
	creatorID   string

	mu          sync.RWMutex
	connections map[string]bool
	emptySince  time.Time
	closed      bool

	eventHandler ws.ChannelEventHandler
	authHandler  ws.ChannelAuthHandler
}

var _ ws.IChannel = (*Channel)(nil)

/** @constructor */
func NewChannel(s ws.IWSServer, key string, data map[string]any, public, proprietary bool, creatorID string) ws.IChannel {
	return &Channel{
		server:      s,
		key:         key,
		sharedData:  data,
		public:      public,
		proprietary: proprietary,
		creatorID:   creatorID,
		connections: map[string]bool{},
		emptySince:  time.Now(),
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

func (c *Channel) IsPublic() bool {
	return c.public
}

func (c *Channel) IsProprietary() bool {
	return c.proprietary
}

func (c *Channel) CreatorID() string {
	return c.creatorID
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
	c.refreshEmptySince()

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
			"__lxws_channel__": event,
			"channel":          c.Key(),
			"id":               conn.ID(),
			"data":             conn.SharedDataForChannel(c),
		}, "text", false)
	}
}

func (c *Channel) Enter(conn ws.IConnection, message map[string]any) (bool, string) {
	if c.Has(conn) {
		return true, ""
	}

	if handler := c.authHandler; handler != nil {
		if ok, reason := handler(conn, message); !ok {
			if reason == "" {
				reason = "access denied"
			}
			return false, reason
		}
	}

	c.AddConnection(conn)

	return true, ""
}

func (c *Channel) SetEventHandler(handler ws.ChannelEventHandler) {
	c.eventHandler = handler
}

func (c *Channel) EventHandler() ws.ChannelEventHandler {
	return c.eventHandler
}

func (c *Channel) SetAuthHandler(handler ws.ChannelAuthHandler) {
	c.authHandler = handler
}

func (c *Channel) AuthHandler() ws.ChannelAuthHandler {
	return c.authHandler
}

// Leave removes conn from the channel - unless conn is this proprietary
// channel's creator genuinely leaving (not merely a temporary disconnect
// that might still reconnect - see conn.Status() check below), in which
// case the whole channel closes instead (see Close). A proprietary
// channel's creator departing is definitionally its closing - there's no
// separate "close channel" action
func (c *Channel) Leave(conn ws.IConnection) {
	c.mu.Lock()
	delete(c.connections, conn.ID())
	c.refreshEmptySince()
	shouldClose := c.proprietary && c.creatorID == conn.ID() && conn.Status() != ws.ConnStatusDisconnected
	c.mu.Unlock()

	if shouldClose {
		c.Close(ws.ChannelCloseCodeCreatorGone)
		return
	}

	var event string
	if conn.Status() == ws.ConnStatusDisconnected {
		event = "mateDisconnected"
	} else {
		event = "mateLeft"
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	for key := range c.connections {
		iCon := c.server.Connections().Get(key)
		iCon.Send(map[string]any{
			"__lxws_channel__": event,
			"channel":          c.Key(),
			"id":               conn.ID(),
		}, "text", false)
	}
}

// Close force-closes the channel: kicks every remaining member (sent a
// "closed" channel message carrying code, distinct from a per-member
// "mateLeft"/"mateDisconnected"), credits back the creator's
// MaxChannelsPerConnection quota if it's still connected, and removes the
// channel from the repo. code is relayed as-is to every kicked member so the
// client can show a situation-appropriate message - see
// ChannelCloseCodeServer/ChannelCloseCodeCreatorGone for this package's own
// codes, or pass any other string for an application-defined reason.
// Used both when a proprietary channel's creator leaves (see Leave, always
// ChannelCloseCodeCreatorGone), when ChannelRepo's sweeper auto-closes a
// non-proprietary channel that's been empty past EmptyChannelTTL (always
// ChannelCloseCodeServer), and by application code closing a channel it no
// longer needs. A no-op if the channel was already closed.
func (c *Channel) Close(code string) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	remaining := slices.Collect(maps.Keys(c.connections))
	c.connections = map[string]bool{}
	c.mu.Unlock()

	for _, id := range remaining {
		if conn := c.server.Connections().Get(id); conn != nil {
			delete(conn.Channels(), c.key)
			conn.Send(map[string]any{
				"__lxws_channel__": "closed",
				"channel":          c.key,
				"code":             code,
			}, "text", false)
		}
	}

	if c.creatorID != "" {
		if creator := c.server.Connections().Get(c.creatorID); creator != nil {
			creator.DecrementCreatedChannels()
		}
	}

	c.server.Channels().Remove(c.key)
}

// refreshEmptySince updates emptySince based on current membership - call
// with c.mu already held. A zero connections count starts (or keeps) the
// "empty since" clock; any member present clears it.
func (c *Channel) refreshEmptySince() {
	if len(c.connections) == 0 {
		if c.emptySince.IsZero() {
			c.emptySince = time.Now()
		}
	} else {
		c.emptySince = time.Time{}
	}
}

// isAutoCloseDue reports whether this channel is eligible for
// ChannelRepo's empty-channel sweep - proprietary channels never are (they
// close via their creator's departure, see Leave).
func (c *Channel) isAutoCloseDue(ttl time.Duration) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.proprietary || len(c.connections) > 0 || c.emptySince.IsZero() {
		return false
	}
	return time.Since(c.emptySince) >= ttl
}
