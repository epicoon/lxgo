package migrator

import (
	"fmt"
	"strconv"

	"github.com/epicoon/lxgo/cmd"
)

/** @interface cmd.ICommand */
type MigratorCommand struct {
	*cmd.Command
}

/** @type cmd.FConstructor */
func NewCommand(_ ...cmd.ICommandOptions) cmd.ICommand {
	return cmd.Prepare(&MigratorCommand{Command: cmd.NewCommand()})
}

func (c *MigratorCommand) Config() *cmd.Config {
	return &cmd.Config{
		Description: "Command to manage migrations",
		Actions: cmd.ActionsConfig{
			"create": cmd.ActionConfig{
				Description: "Create new migration",
				Executor:    create,
				Params: cmd.ParamsConfig{
					"name": cmd.ParamConfig{
						Description: "New migration name",
						Type:        cmd.ParamTypeString,
						Required:    true,
					},
				},
			},
			"show": cmd.ActionConfig{
				Description: "Show migrations state",
				Executor:    show,
				Params: cmd.ParamsConfig{
					"count": cmd.ParamConfig{
						Description: "Migrations count to show. If not defined all migartions will be shown",
						Type:        cmd.ParamTypeInt,
						Required:    false,
						Default:     0,
						HideDefault: true,
					},
				},
			},
			"check": cmd.ActionConfig{
				Description: "Show unapplied migrations",
				Executor:    check,
			},
			"up": cmd.ActionConfig{
				Description: "Apply migrations",
				Executor:    up,
			},
			"down": cmd.ActionConfig{
				Description: "Cancel migrations",
				Executor:    down,
				Params: cmd.ParamsConfig{
					"count": cmd.ParamConfig{
						Description: "Migrations count to cancel. If not defined the last migartion will be cancelled",
						Type:        cmd.ParamTypeInt,
						Required:    false,
						Default:     0,
						HideDefault: true,
					},
				},
			},
		},
	}
}

/** @type cmd.FAction */
func create(c cmd.ICommand) error {
	params := c.Params()
	name, ok := params["name"]
	if !ok {
		fmt.Println("Please, enter the --name parameter for new migration")
		return nil
	}

	nameStr, ok := name.(string)
	if !ok {
		fmt.Println("The --name parameter must be a string")
		return nil
	}

	err := Create(nameStr)
	if err != nil {
		fmt.Printf("Can not create migration file: %s\n", err)
		return nil
	}

	return nil
}

/** @type cmd.FAction */
func show(c cmd.ICommand) error {
	params := c.Params()
	cntStr, ok := params["count"]
	if !ok {
		cntStr = "0"
	}

	cnt, err := strconv.Atoi(fmt.Sprintf("%v", cntStr))
	if err != nil {
		fmt.Printf("Invalid count parameter: %v. Integer required\n", cntStr)
		return nil
	}

	mm, err := Show(cnt)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return nil
	}

	if len(mm) == 0 {
		fmt.Println("There are no migrations")
		return nil
	}

	fmt.Println("Migrations:")
	for _, m := range mm {
		var pref string
		if m.isApplied() {
			pref = "+"
		} else {
			pref = "-"
		}
		fmt.Printf("%v %s\n", pref, m.file)
	}

	return nil
}

/** @type cmd.FAction */
func check(c cmd.ICommand) error {
	mm, err := Check()
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return nil
	}

	if len(mm) == 0 {
		fmt.Println("All migrations are allied")
		return nil
	}

	fmt.Println("Migrations:")
	for _, m := range mm {
		fmt.Printf("- %s\n", m.file)
	}

	return nil
}

/** @type cmd.FAction */
func up(c cmd.ICommand) error {
	Up()
	return nil
}

/** @type cmd.FAction */
func down(c cmd.ICommand) error {
	params := c.Params()
	cntStr, ok := params["count"]
	if !ok {
		cntStr = "0"
	}

	cnt, err := strconv.Atoi(fmt.Sprintf("%v", cntStr))
	if err != nil {
		fmt.Printf("Invalid count parameter: %v. Integer required\n", cntStr)
		return nil
	}

	Down(cnt)

	return nil
}
