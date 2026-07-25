package component

import (
	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/app"
)

// WSServerConfig is WSServer's config - see "Setup component" in the README
// for the full config.yaml shape this binds to.
type WSServerConfig struct {
	*app.ComponentConfig
	// Host is the address the WS server listens on.
	Host string
	// Port is the port the WS server listens on.
	Port int
	// AllowedOrigins is the Origin allowlist for the WS handshake -
	// empty/unset means no restriction.
	AllowedOrigins []string
	// DefaultChannel, if Key is set, is created automatically at startup and
	// every connection joins it automatically on connect/reconnect.
	DefaultChannel struct {
		Key        string
		SharedData map[string]any
	}
	// MaxRequestsPerMinute is the per-connection rate limit (0/unset means
	// no limit).
	MaxRequestsPerMinute int
	// MaxConnectionsPerIp is the per-IP connection limit (0/unset means no
	// limit).
	MaxConnectionsPerIp int
	// MaxChannelsPerConnection is the per-connection cap on client-created
	// channels (0/unset means no limit).
	MaxChannelsPerConnection int
	// EmptyChannelTTL is, in seconds, how long a non-proprietary channel may
	// sit empty before it's auto-closed (0/unset disables this entirely).
	EmptyChannelTTL int
	// ReconnectionAllowed enables the reconnect flow for disconnected
	// connections.
	ReconnectionAllowed bool
	// ReconnectionDuration is, in milliseconds, how long a disconnected
	// connection stays eligible to reconnect before it's permanently gone.
	ReconnectionDuration int
	// LifecycleLog enables IWSServer.LifecycleLog's output.
	LifecycleLog bool
	// LifecycleError enables IWSServer.LifecycleError's output.
	LifecycleError bool
}

/** @constructor kernel.CAppComponentConfig */

// NewWSServerConfig returns an empty WSServerConfig - see
// WSServer.CConfig, not normally called directly.
func NewWSServerConfig() kernel.IAppComponentConfig {
	return &WSServerConfig{
		ComponentConfig: app.NewComponentConfigStruct(),
	}
}
