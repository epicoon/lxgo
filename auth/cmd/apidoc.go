package cmd

import (
	"github.com/epicoon/lxgo/auth/internal/core"
	"github.com/epicoon/lxgo/cmd"
	apidoc "github.com/epicoon/lxgo/kernel/cmd"
	lxHttp "github.com/epicoon/lxgo/kernel/http"
)

/** @type cmd.FConstructor */
func NewApiDocCommand(_ ...cmd.ICommandOptions) cmd.ICommand {
	r := lxHttp.NewRouter(nil)
	core.InitRoutes(r)
	return apidoc.NewApiDocCommand(apidoc.ApiDocCommandOptions{
		Router: r,
		Output: "ApiDoc.md",
	})
}
