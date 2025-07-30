package main

import (
	authcmd "github.com/epicoon/lxgo/auth/cmd"
	"github.com/epicoon/lxgo/cmd"
)

func main() {
	cmd.Init(cmd.CommandsList{
		"":         authcmd.NewMainCommand,
		"client":   authcmd.NewClientCommand,
		"migrator": authcmd.NewMigratorCommand,
		"apidoc":   authcmd.NewApiDocCommand,
	})
	cmd.Run()
}
