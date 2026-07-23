package src

import (
	"fmt"
	"maps"
	"sync"
	"time"

	"github.com/epicoon/lxgo/ws"
)

/** @interface ws.IChannelRepo */
type ChannelRepo struct {
	server ws.IWSServer

	mu       sync.RWMutex
	channels map[string]ws.IChannel

	wg   sync.WaitGroup
	quit chan struct{}
}

var _ ws.IChannelRepo = (*ChannelRepo)(nil)

/** @constructor */
func NewChannelRepo(s ws.IWSServer) *ChannelRepo {
	r := &ChannelRepo{
		server:   s,
		channels: map[string]ws.IChannel{},
		quit:     make(chan struct{}),
	}
	r.wg.Add(1)
	go r.sweeper()
	return r
}

func (r *ChannelRepo) Close() {
	close(r.quit)
	r.wg.Wait()
}

func (r *ChannelRepo) Init() {
	if r.server.DefaultChannelKey() != "" {
		// The DefaultChannel is a startup-time technical bootstrap, not a
		// creation request from anyone - it always skips ChannelValidator,
		// but still goes through ChannelCreatedHandler like any other
		// channel (see ChannelCreatedHandler's doc-comment). Never
		// proprietary/auto-closing - see insert() and the sweeper's
		// defaultKey exclusion.
		channel := r.insert(r.server.DefaultChannelKey(), r.server.DefaultChannelData(), false)
		if handler := r.server.ChannelCreatedHandler(); handler != nil {
			handler(channel, map[string]any{})
		}
	}
}

// CreateChannel constructs and registers a new channel per builder - an
// empty builder.Key() means "generate one" (the common case for
// client-initiated creation, see IChannelBuilder). If a ChannelValidator is
// set (IWSServer.SetChannelValidator), it's consulted before the channel is
// considered created; a false result rolls the registration back and
// returns the validator's reason. If builder.Creator() is a real connection,
// the configured per-connection limit (MaxChannelsPerConnection) is enforced
// first. On success, ChannelCreatedHandler (if set) runs before the channel
// is returned.
func (r *ChannelRepo) CreateChannel(builder ws.IChannelBuilder) (ws.IChannel, string) {
	if creator := builder.Creator(); creator != nil {
		if max := r.server.MaxChannelsPerConnection(); max > 0 && creator.CreatedChannelsCount() >= max {
			return nil, "channel creation limit exceeded"
		}
	}

	key := builder.Key()
	r.mu.Lock()
	if key == "" {
		for {
			key = RandHash()
			if _, exists := r.channels[key]; !exists {
				break
			}
		}
		builder.SetKey(key)
	} else if _, exists := r.channels[key]; exists {
		r.mu.Unlock()
		return nil, fmt.Sprintf("channel key '%s' already exists", key)
	}
	creatorID := ""
	if creator := builder.Creator(); creator != nil {
		creatorID = creator.ID()
	}
	channel := NewChannel(r.server, key, builder.SharedData(), builder.Public(), builder.Proprietary(), creatorID)
	r.channels[key] = channel
	r.mu.Unlock()

	if validator := r.server.ChannelValidator(); validator != nil {
		if ok, reason := validator(builder); !ok {
			r.mu.Lock()
			delete(r.channels, key)
			r.mu.Unlock()
			if reason == "" {
				reason = "channel creation denied"
			}
			return nil, reason
		}
	}

	if creator := builder.Creator(); creator != nil {
		creator.IncrementCreatedChannels()
	}

	if handler := r.server.ChannelCreatedHandler(); handler != nil {
		handler(channel, builder.InitData())
	}

	// A public channel is already sent to every new connection via
	// connect()/reconnect()'s "channels" field - but connections already on
	// the server at creation time would never otherwise learn about it.
	// Announce it to everyone except the creator, who already gets the full
	// ack (with membership too) from Connection.createChannel.
	if channel.IsPublic() {
		r.announcePublicChannel(channel, builder.Creator())
	}

	return channel, ""
}

// announcePublicChannel notifies every other connection currently on the
// server that a new public channel exists - same "channel": {key, data}
// shape Connection.createChannel sends the creator, just without
// "connections" (this connection isn't a member, only learned it exists -
// still requires enterChannel to join).
func (r *ChannelRepo) announcePublicChannel(channel ws.IChannel, creator ws.IConnection) {
	data := map[string]any{
		"__lxws_action__": "createChannel",
		"channel": map[string]any{
			"key":  channel.Key(),
			"data": channel.SharedData(),
		},
	}
	for id, conn := range r.server.Connections().GetAll() {
		if creator != nil && id == creator.ID() {
			continue
		}
		if err := conn.Send(data, "text", false); err != nil {
			r.server.LifecycleError("public channel announce send error for %s: %v", id, err)
		}
	}
}

// insert registers a channel directly, bypassing ChannelValidator and the
// creation limit entirely - only used for the DefaultChannel bootstrap (see
// Init()), which isn't a creation request from anyone (so no creator,
// never proprietary).
func (r *ChannelRepo) insert(key string, data map[string]any, public bool) ws.IChannel {
	r.mu.Lock()
	defer r.mu.Unlock()
	channel := NewChannel(r.server, key, data, public, false, "")
	r.channels[key] = channel
	return channel
}

func (r *ChannelRepo) Has(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, exists := r.channels[key]
	return exists
}

func (r *ChannelRepo) Get(key string) ws.IChannel {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.channels[key]
}

func (r *ChannelRepo) Remove(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.channels, key)
}

func (r *ChannelRepo) Channels() map[string]ws.IChannel {
	r.mu.RLock()
	defer r.mu.RUnlock()
	channels := make(map[string]ws.IChannel, len(r.channels))
	maps.Copy(channels, r.channels)
	return channels
}

func (r *ChannelRepo) PublicChannels() []ws.IChannel {
	r.mu.RLock()
	defer r.mu.RUnlock()
	channels := []ws.IChannel{}
	for _, ch := range r.channels {
		if ch.IsPublic() {
			channels = append(channels, ch)
		}
	}
	return channels
}

// sweeper auto-closes non-proprietary channels (other than DefaultChannel)
// that have been empty for at least EmptyChannelTTL - disabled entirely
// when that config is 0/unset (see IWSServer.EmptyChannelTTL), same "0 =
// no limit/no restriction" convention used elsewhere in this package.
// Proprietary channels are never touched here - they close via their
// creator's departure (see Channel.Leave).
func (r *ChannelRepo) sweeper() {
	defer r.wg.Done()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			ttlSeconds := r.server.EmptyChannelTTL()
			if ttlSeconds <= 0 {
				continue
			}
			ttl := time.Duration(ttlSeconds) * time.Second
			defaultKey := r.server.DefaultChannelKey()

			r.mu.RLock()
			due := make([]ws.IChannel, 0)
			for key, ch := range r.channels {
				if key == defaultKey {
					continue
				}
				// isAutoCloseDue is deliberately not part of IChannel - it's
				// pure internal sweep-eligibility bookkeeping, not something
				// a package consumer should ever need to ask a channel.
				// Close() itself, right below, is reached through the plain
				// IChannel interface.
				if concrete, ok := ch.(*Channel); ok {
                    if concrete.isAutoCloseDue(ttl) {
                        due = append(due, ch)
                    }
				}
			}
			r.mu.RUnlock()

			for _, ch := range due {
				ch.Close(ws.ChannelCloseCodeServer)
			}
		case <-r.quit:
			return
		}
	}
}
