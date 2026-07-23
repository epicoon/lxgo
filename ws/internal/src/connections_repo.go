package src

import (
	"sync"
	"time"

	"github.com/epicoon/lxgo/ws"
)

type reqCounter struct {
	last time.Time
	cnt  int
}

type tombstone struct {
	conn            ws.IConnection
	channels        map[string]map[string]any
	createdChannels int
	expiresAt       time.Time
}

type ConnRepo struct {
	server ws.IWSServer

	mu      sync.RWMutex
	reqMu   sync.Mutex
	tombsMu sync.Mutex
	wg      sync.WaitGroup
	quit    chan struct{}

	conns      map[string]ws.IConnection
	tombstones map[string]*tombstone
	reqStat    map[string]*reqCounter
	IPs        map[string]int
}

func NewConnRepo(s ws.IWSServer) ws.IConnRepo {
	r := &ConnRepo{
		server:     s,
		conns:      map[string]ws.IConnection{},
		tombstones: map[string]*tombstone{},
		reqStat:    map[string]*reqCounter{},
		IPs:        map[string]int{},
		quit:       make(chan struct{}),
	}
	r.wg.Add(1)
	go r.sweeper()
	return r
}

func (r *ConnRepo) Close() {
	close(r.quit)
	r.wg.Wait()
}

func (r *ConnRepo) Add(c ws.IConnection) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.conns[c.ID()] = c
	r.IPs[c.IP()]++

	r.tombsMu.Lock()
	delete(r.tombstones, c.ID())
	r.tombsMu.Unlock()
}

func (r *ConnRepo) RemoveImmediate(c ws.IConnection) {
	r.mu.Lock()
	delete(r.conns, c.ID())
	r.dropIP(c)
	r.mu.Unlock()

	r.tombsMu.Lock()
	delete(r.tombstones, c.ID())
	r.tombsMu.Unlock()

	r.reqMu.Lock()
	delete(r.reqStat, c.ID())
	r.reqMu.Unlock()
}

func (r *ConnRepo) MarkDisconnected(c ws.IConnection) {
	r.mu.Lock()
	if cur, ok := r.conns[c.ID()]; ok && cur == c {
		delete(r.conns, c.ID())
		r.dropIP(c)
		r.reqMu.Lock()
		delete(r.reqStat, c.ID())
		r.reqMu.Unlock()
	}
	r.mu.Unlock()

	duration := time.Duration(r.server.ReconnectionDuration()) * time.Millisecond
	r.tombsMu.Lock()
	r.tombstones[c.ID()] = &tombstone{
		conn:            c,
		channels:        c.Channels(),
		createdChannels: c.CreatedChannelsCount(),
		expiresAt:       time.Now().Add(duration),
	}
	r.tombsMu.Unlock()
}

func (r *ConnRepo) Reconnect(conn ws.IConnection, ID string) bool {
	r.tombsMu.Lock()
	defer r.tombsMu.Unlock()

	tConn := r.tombstones[ID]
	if tConn == nil || tConn.conn.IP() != conn.IP() {
		return false
	}

	delete(r.tombstones, ID)

	r.reqMu.Lock()
	defer r.reqMu.Unlock()
	r.reqStat[ID] = r.reqStat[conn.ID()]
	delete(r.reqStat, conn.ID())

	r.mu.RLock()
	defer r.mu.RUnlock()

	r.conns[ID] = r.conns[conn.ID()]
	delete(r.conns, conn.ID())

	conn.SetID(tConn.conn.ID())
	conn.SetStatus(ws.ConnStatusReconnecting)

	chs := map[string]map[string]any{}
	for key, val := range tConn.channels {
		ch := r.server.Channels().Get(key)
		if ch != nil {
			chs[key] = val
			ch.AddConnection(conn)
		}
	}
	conn.SetChannels(chs)
	conn.SetCreatedChannelsCount(tConn.createdChannels)
	return true
}

func (r *ConnRepo) Has(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.conns[id]
	return exists
}

func (r *ConnRepo) Get(id string) ws.IConnection {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.conns[id]
}

func (r *ConnRepo) GetAll() map[string]ws.IConnection {
	return r.conns
}

func (r *ConnRepo) CheckRequestLimit(c ws.IConnection) bool {
	if r.server.MaxRequestsPerMinute() == 0 {
		return true
	}

	now := time.Now()
	windowStart := now.Add(-1 * time.Minute)

	r.reqMu.Lock()
	defer r.reqMu.Unlock()

	history, exists := r.reqStat[c.ID()]
	if !exists || history.last.Before(windowStart) {
		r.reqStat[c.ID()] = &reqCounter{
			last: now,
			cnt:  1,
		}
		history = r.reqStat[c.ID()]
	} else {
		history.cnt++
	}

	return history.cnt <= r.server.MaxRequestsPerMinute()
}

func (r *ConnRepo) CheckIPLimit(c ws.IConnection) bool {
	if r.server.MaxConnectionsPerIp() == 0 {
		return true
	}
	return r.IPs[c.IP()] < r.server.MaxConnectionsPerIp()
}

func (r *ConnRepo) sweeper() {
	defer r.wg.Done()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			now := time.Now()
			r.tombsMu.Lock()
			for id, t := range r.tombstones {
				if now.After(t.expiresAt) {
					r.mu.Lock()
					delete(r.conns, id)
					r.mu.Unlock()

					r.reqMu.Lock()
					delete(r.reqStat, id)
					r.reqMu.Unlock()

					conn := r.tombstones[id].conn
					conn.SetChannels(r.tombstones[id].channels)
					conn.Close()
					delete(r.tombstones, id)
				}
			}
			r.tombsMu.Unlock()
		case <-r.quit:
			return
		}
	}
}

func (r *ConnRepo) dropIP(c ws.IConnection) {
	r.IPs[c.IP()]--
	if r.IPs[c.IP()] == 0 {
		delete(r.IPs, c.IP())
	}
}
