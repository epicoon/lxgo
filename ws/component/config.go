package component

import (
	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/app"
)

type WSServerConfig struct {
	*app.ComponentConfig
	//TODO use Protocol
	Protocol string
	Host     string
	Port     int
	// TODO use AllowedOrigins
	AllowedOrigins []string
	DefaultChannel struct {
		Key        string
		SharedData map[string]any
	}
	MaxRequestsPerMinute int
	MaxConnectionsPerIp  int
	ReconnectionAllowed  bool
	ReconnectionDuration int
	LifecycleLog         bool
	LifecycleError       bool
}

/** @constructor kernel.CAppComponentConfig */
func NewWSServerConfig() kernel.IAppComponentConfig {
	return &WSServerConfig{
		ComponentConfig: app.NewComponentConfigStruct(),
	}
}
