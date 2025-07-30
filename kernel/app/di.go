package app

import (
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

func (c *DIContainer) Get(key string) any {
	f, ok := c.list[key]
	if !ok {
		return nil
	}

	result := f()

	return result
}
