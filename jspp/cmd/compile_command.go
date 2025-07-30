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

type CompileCommandOptions struct {
	App kernel.IApp
}

/** @interface cmd.ICommand */
type CompileCommand struct {
	*cmd.Command
	app kernel.IApp
}

/** @type cmd.FConstructor */
func NewCompileCommand(opt ...cmd.ICommandOptions) cmd.ICommand {
	options := cmd.GetOptions[CompileCommandOptions](opt)
	if options.App == nil {
		panic("CompileCommand option 'App' is not defined")
	}

	c := &CompileCommand{
		Command: cmd.NewCommand(),
		app:     options.App,
	}
	c.RegisterActions(cmd.ActionsList{
		"build-core":        buildCore,
		"build-maps":        buildMaps,
		"build-modules-map": buildModulesMap,
		"build-plugins-map": buildPluginsMap,
	})
	return c
}

/** @function cmd.FAction */
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

/** @function cmd.FAction */
func buildMaps(com cmd.ICommand) error {
	return build(com, utils.MapBuilderOptions{
		Modules: true,
		Plugins: true,
	})
}

/** @function cmd.FAction */
func buildModulesMap(com cmd.ICommand) error {
	return build(com, utils.MapBuilderOptions{
		Modules: true,
	})
}

/** @function cmd.FAction */
func buildPluginsMap(com cmd.ICommand) error {
	return build(com, utils.MapBuilderOptions{
		Plugins: true,
	})
}

func build(com cmd.ICommand, op utils.MapBuilderOptions) error {
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

	if c.Flag("preview") {
		if op.Modules {
			src := utils.GetModulesSrcList(pp)
			fmt.Println("Modules src directories:")
			for _, val := range src {
				fmt.Printf("- %s\n", val)
			}
		}
		if op.Plugins {
			src := utils.GetPluginsSrcList(pp)
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
