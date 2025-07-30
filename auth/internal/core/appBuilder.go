package core

import (
	"fmt"

	cvn "github.com/epicoon/lxgo/auth/internal/conventions"
	lxApp "github.com/epicoon/lxgo/kernel/app"
	"github.com/epicoon/lxgo/kernel/config"
)

func PrepareApp(configPath string) (cvn.IApp, error) {
	app := NewApp()

	conf, err := config.Load(app.Pathfinder().GetAbsPath(configPath))
	if err != nil {
		return nil, fmt.Errorf("can not read application config. Cause: %s", err)
	}

	if err := lxApp.InitApp(app, conf); err != nil {
		return nil, fmt.Errorf("can not init application: %s", err)
	}

	if err := app.Connection().Connect(); err != nil {
		return nil, fmt.Errorf("can not connect to DB. Cause: %s", err)
	}

	if err := setComponents(app); err != nil {
		return nil, err
	}

	InitRoutes(app.Router())

	return app, nil
}
