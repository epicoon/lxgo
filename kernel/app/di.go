package app

import (
	"fmt"

	"github.com/epicoon/lxgo/kernel"
)

/** @interface kernel.IDIContainer */

// DIContainer is the default kernel.IDIContainer implementation.
type DIContainer struct {
	app  kernel.IApp
	list kernel.CAnyList
}

var _ kernel.IDIContainer = (*DIContainer)(nil)

/** @constructor */

// NewDIContainer constructs a DIContainer bound to app.
func NewDIContainer(app kernel.IApp) kernel.IDIContainer {
	return &DIContainer{app: app}
}

// Init registers list, replacing any previously registered factories.
func (c *DIContainer) Init(list kernel.CAnyList) {
	c.list = list
}

// Register adds list's factories to the container, failing if any key is already registered.
func (c *DIContainer) Register(list kernel.CAnyList) error {
	if c.list == nil {
		c.list = make(kernel.CAnyList, len(list))
	}

	for key, constructor := range list {
		_, exists := c.list[key]
		if exists {
			return fmt.Errorf("DI-key '%s' already initialized", key)
		}
		c.list[key] = constructor
	}

	return nil
}

// Get resolves the value registered under key by calling its factory, or
// returns nil if key isn't registered.
func (c *DIContainer) Get(key string) any {
	f, ok := c.list[key]
	if !ok {
		return nil
	}

	result := f()

	return result
}
