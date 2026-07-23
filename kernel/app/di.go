package app

import (
	"fmt"

	"github.com/epicoon/lxgo/kernel"
)

type DIContainer struct {
	app  kernel.IApp
	list kernel.CAnyList
}

func NewDIConteiner(app kernel.IApp) kernel.IDIContainer {
	return &DIContainer{app: app}
}

func (c *DIContainer) Init(list kernel.CAnyList) {
	c.list = list
}

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

func (c *DIContainer) Get(key string) any {
	f, ok := c.list[key]
	if !ok {
		return nil
	}

	result := f()

	return result
}
