// Package cmd provides the "compile" console command for building/serving
// the jspp preprocessor's static output - core JS, modules map, plugins map.
package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/epicoon/lxgo/cmd"
	jsppComp "github.com/epicoon/lxgo/jspp/component"
	"github.com/epicoon/lxgo/jspp/internal/utils"
	"github.com/epicoon/lxgo/kernel"
)

// CompileCommandOptions is CompileCommand's cmd.ICommandOptions - App is required.
type CompileCommandOptions struct {
	// App is the application the jspp component is registered on.
	App kernel.IApp
}

/** @interface cmd.ICommand */

// CompileCommand builds jspp's static output: the core JS bundle, and the
// modules/plugins maps - see NewCompileCommand.
type CompileCommand struct {
	*cmd.Command
	app kernel.IApp
}

var _ cmd.ICommand = (*CompileCommand)(nil)

/** @constructor cmd.CCommand */

// NewCompileCommand constructs a CompileCommand with "build"/"build-core"/
// "build-maps"/"build-modules-map"/"build-plugins-map"/"scaffold-plugin"
// actions - panics if CompileCommandOptions.App isn't given.
func NewCompileCommand(opt ...cmd.ICommandOptions) cmd.ICommand {
	options := cmd.GetOptions[CompileCommandOptions](opt)
	if options.App == nil {
		panic("CompileCommand option 'App' is not defined")
	}

	return cmd.Prepare(&CompileCommand{
		Command: cmd.NewCommand(),
		app:     options.App,
	})
}

// Config declares all of CompileCommand's actions.
func (c *CompileCommand) Config() *cmd.Config {
	return &cmd.Config{
		Description: "Build jspp's static output, or scaffold a new plugin.",
		Actions: cmd.ActionsConfig{
			"build": cmd.ActionConfig{
				Description: "Build core.js and the modules/plugins maps.",
				Executor:    build,
			},
			"build-core": cmd.ActionConfig{
				Description: "Build core.js.",
				Executor:    buildCore,
			},
			"build-maps": cmd.ActionConfig{
				Description: "Build the modules and plugins maps.",
				Executor:    buildMaps,
			},
			"build-modules-map": cmd.ActionConfig{
				Description: "Build the modules map only.",
				Executor:    buildModulesMap,
			},
			"build-plugins-map": cmd.ActionConfig{
				Description: "Build the plugins map only.",
				Executor:    buildPluginsMap,
			},
			"scaffold-plugin": cmd.ActionConfig{
				Description: "Create a skeleton for a new plugin: lx-plugin.yaml, snippets/_root.js, Plugin.js (add --full for assets/i18n, assets/css, and one GUI node too).",
				Executor:    scaffoldPlugin,
				Params: cmd.ParamsConfig{
					"name": cmd.ParamConfig{
						Description: "The new plugin's name (its directory name and lx-plugin.yaml's 'name')",
						Type:        cmd.ParamTypeString,
						Required:    true,
						Interactive: true,
					},
					"path": cmd.ParamConfig{
						Description:  "The target directory for the new plugin (one of Components.JSPreprocessor.Plugins)",
						Type:         cmd.ParamTypeEnum,
						ElemType:     cmd.ParamTypeString,
						FTypeDetails: pluginPaths,
						Required:     true,
						Interactive:  true,
					},
				},
			},
		},
	}
}

/** @handler cmd.FAction */
func build(com cmd.ICommand) error {
	if err := buildCore(com); err != nil {
		return err
	}
	if err := buildMaps(com); err != nil {
		return err
	}
	return nil
}

/** @handler cmd.FAction */
func buildCore(com cmd.ICommand) error {
	c := com.(*CompileCommand)
	app := c.app
	if app == nil {
		return errors.New("command require access to application through 'app' option")
	}

	_, filename, _, _ := runtime.Caller(0)
	absPath, _ := filepath.Abs(filename)
	parentDir := filepath.Dir(filepath.Dir(absPath))

	pp, _ := jsppComp.AppComponent(app)
	utils.BuildCore(pp, parentDir, c.Flag("src"))

	fmt.Println("Done")
	return nil
}

/** @handler cmd.FAction */
func buildMaps(com cmd.ICommand) error {
	return buildMap(com, utils.MapBuilderOptions{
		Modules: true,
		Plugins: true,
	})
}

/** @handler cmd.FAction */
func buildModulesMap(com cmd.ICommand) error {
	return buildMap(com, utils.MapBuilderOptions{
		Modules: true,
	})
}

/** @handler cmd.FAction */
func buildPluginsMap(com cmd.ICommand) error {
	return buildMap(com, utils.MapBuilderOptions{
		Plugins: true,
	})
}

func buildMap(com cmd.ICommand, op utils.MapBuilderOptions) error {
	c := com.(*CompileCommand)
	app := c.app
	if app == nil {
		return errors.New("command require access to application through 'app' option")
	}
	pp, _ := jsppComp.AppComponent(app)

	root := app.Pathfinder().GetRoot()
	goModPath := filepath.Join(root, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		return errors.New("go.mod not found")
	}

	if c.Flag("p") {
		if op.Modules {
			src, err := utils.GetModulesSrcList(pp)
			if err != nil {
				return err
			}
			fmt.Println("Modules src directories:")
			for _, val := range src {
				fmt.Printf("- %s\n", val)
			}
		}
		if op.Plugins {
			src, err := utils.GetPluginsSrcList(pp)
			if err != nil {
				return err
			}
			fmt.Println("Plugins src directories:")
			for _, val := range src {
				fmt.Printf("- %s\n", val)
			}
		}
		return nil
	}

	if err := utils.BuildMaps(pp, op); err != nil {
		return err
	}

	fmt.Println("Done")
	return nil
}
