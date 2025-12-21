package src

import (
	"sync"

	"github.com/epicoon/lxgo/ws"
)

/** @interface ws.IChannelRepo */
type ChannelRepo struct {
	server ws.IWSServer

	mu       sync.RWMutex
	channels map[string]ws.IChannel
}

var _ ws.IChannelRepo = (*ChannelRepo)(nil)

/** @constructor */
func NewChannelRepo(s ws.IWSServer) *ChannelRepo {
	return &ChannelRepo{
		server:   s,
		channels: map[string]ws.IChannel{},
	}
}

func (r *ChannelRepo) Init() {
	if r.server.DefaultChannelKey() != "" {
		r.CreateChannel(r.server.DefaultChannelKey(), r.server.DefaultChannelData())
	}
}

func (r *ChannelRepo) CreateChannel(key string, data map[string]any) ws.IChannel {
	if r.Has(key) {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	channel := NewChannel(r.server, key, data)
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
