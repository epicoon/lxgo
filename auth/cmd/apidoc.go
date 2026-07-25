package cmd

import (
	"github.com/epicoon/lxgo/auth/internal/core"
	"github.com/epicoon/lxgo/cmd"
	apidoc "github.com/epicoon/lxgo/kernel/cmd"
	lxHttp "github.com/epicoon/lxgo/kernel/http"
)

/** @constructor cmd.CCommand */

// NewApiDocCommand builds the apidoc:gen command, which regenerates
// ApiDoc.md from the application's actually registered routes/forms.
func NewApiDocCommand(_ ...cmd.ICommandOptions) cmd.ICommand {
	r := lxHttp.NewRouter(nil)
	core.InitRoutes(r)
	return apidoc.NewApiDocCommand(apidoc.ApiDocCommandOptions{
		Router: r,
		Output: "ApiDoc.md",
	})
}
