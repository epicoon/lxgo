package cmd

import (
	"fmt"

	"github.com/epicoon/lxgo/cmd"
	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/app"
	"github.com/epicoon/lxgo/kernel/config"
	"github.com/epicoon/lxgo/migrator"
)

/** @type cmd.FConstructor */
func NewMigratorCommand(_ ...cmd.ICommandOptions) cmd.ICommand {
	conf, err := config.Load("config.yaml")
	if err != nil {
		fmt.Printf("Can not read application config. Cause: %q", err)
		return nil
	}

	if !config.HasParam(conf, "Database") {
		fmt.Println("The application config doesn't contain Database configuration")
		return nil
	}

	dbConf, err := config.GetParam[kernel.Config](conf, "Database")
	if err != nil {
		fmt.Printf("can not read Database config: %s", err)
		return nil
	}
	connection := app.NewConnection()
	connection.SetConfig(&dbConf)
	err = connection.Connect()
	if err != nil {
		fmt.Printf("Can not connect to DB. Cause: %q", err)
		return nil
	}

	migrator.Init(connection.DB(), "migrations")

	return migrator.NewCommand()
}
