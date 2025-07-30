package cmd

import (
	"fmt"

	"github.com/epicoon/lxgo/auth/internal/core"
	"github.com/epicoon/lxgo/cmd"
)

/** @interface cmd.ICommand */
type MainCommand struct {
	*cmd.Command
}

/** @constructor cmd.FConstructor */
func NewMainCommand(_ ...cmd.ICommandOptions) cmd.ICommand {
	return &MainCommand{Command: cmd.NewCommand()}
}

func (c *MainCommand) Exec() error {
	app, err := core.PrepareApp("config.yaml")
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return nil
	}

	app.Run()
	app.Final()
	return nil
}
