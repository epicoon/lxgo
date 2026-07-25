// Package cmd holds the auth service's console commands - the default
// (unnamed) command that runs the server itself, plus client/admin/migrator/
// apidoc management commands. See the "Console commands" section of the
// package README for the full command-line reference.
package cmd

import (
	"fmt"

	"github.com/epicoon/lxgo/auth/internal/core"
	"github.com/epicoon/lxgo/cmd"
)

/** @interface cmd.ICommand */

// MainCommand is the default (unnamed) command - `go run .` with no command
// name - that starts the auth server itself.
type MainCommand struct {
	*cmd.Command
}

var _ cmd.ICommand = (*MainCommand)(nil)

/** @constructor cmd.CCommand */

// NewMainCommand constructs a MainCommand.
func NewMainCommand(_ ...cmd.ICommandOptions) cmd.ICommand {
	return &MainCommand{Command: cmd.NewCommand()}
}

// Exec loads config.yaml, builds and runs the auth application.
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
