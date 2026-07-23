package component

import (
	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/app"
)

type WSServerConfig struct {
	*app.ComponentConfig
	Host           string
	Port           int
	AllowedOrigins []string
	DefaultChannel struct {
		Key        string
		SharedData map[string]any
	}
	MaxRequestsPerMinute     int
	MaxConnectionsPerIp      int
	MaxChannelsPerConnection int
	EmptyChannelTTL          int
	ReconnectionAllowed      bool
	ReconnectionDuration     int
	LifecycleLog             bool
	LifecycleError           bool
}

/** @constructor kernel.CAppComponentConfig */
func NewWSServerConfig() kernel.IAppComponentConfig {
	return &WSServerConfig{
		ComponentConfig: app.NewComponentConfigStruct(),
	}
}
