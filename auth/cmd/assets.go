package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/epicoon/lxgo/cmd"
	jsppComponent "github.com/epicoon/lxgo/jspp/component"
	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/apptest"
)

/** @interface cmd.ICommand */

// AssetsCommand builds the static JS bundles under client/js/* -
// each app's src/App.js is compiled through lxgo-jspp's compiler, which
// inlines whatever it lx.import(...)s, instead of a JS toolchain
// (webpack/babel/npm).
type AssetsCommand struct {
	*cmd.Command
}

var _ cmd.ICommand = (*AssetsCommand)(nil)

/** @constructor cmd.CCommand */

// NewAssetsCommand constructs an AssetsCommand with a "build" action.
func NewAssetsCommand(_ ...cmd.ICommandOptions) cmd.ICommand {
	c := &AssetsCommand{Command: cmd.NewCommand()}
	c.RegisterActions(cmd.ActionsList{
		"build": buildAssets,
	})
	return c
}

// clientApps lists the client/js/* entry points to build, relative to
// client/js.
var clientApps = []string{"form", "client"}

/** @handler cmd.FAction */
func buildAssets(_ cmd.ICommand) error {
	sysPath, err := os.MkdirTemp("", "lxgo-auth-assets-*")
	if err != nil {
		return fmt.Errorf("can not create temp dir: %v", err)
	}
	defer os.RemoveAll(sysPath)

	// A throwaway jspp preprocessor just for its compiler - this build step
	// doesn't need a named-modules map (every lx.import(...) here is
	// path-based), so SysPath/MapsPath just need to point somewhere writable.
	app, err := apptest.New(kernel.Dict{
		"Components": kernel.Dict{
			"JSPreprocessor": kernel.Dict{
				"SysPath":  sysPath,
				"MapsPath": sysPath,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("can not build jspp preprocessor: %v", err)
	}
	if err := jsppComponent.SetAppComponent(app, "Components.JSPreprocessor"); err != nil {
		return fmt.Errorf("can not register jspp component: %v", err)
	}
	pp, err := jsppComponent.AppComponent(app)
	if err != nil {
		return fmt.Errorf("can not get jspp component: %v", err)
	}

	for _, appName := range clientApps {
		entryPath := filepath.Join("client", "js", appName, "src", "App.js")
		distDir := filepath.Join("client", "js", appName, "dist")
		distPath := filepath.Join(distDir, "bundle.js")

		code, err := pp.CompilerBuilder().
			SetClientContext().
			SetUnwrapped().
			SetFilePath(entryPath).
			Compiler().
			Run()
		if err != nil {
			return fmt.Errorf("can not compile '%s': %v", entryPath, err)
		}

		if err := os.MkdirAll(distDir, 0755); err != nil {
			return err
		}
		if err := os.WriteFile(distPath, []byte(code), 0644); err != nil {
			return err
		}

		fmt.Printf("built %s\n", distPath)
	}

	fmt.Println("Done")
	return nil
}
