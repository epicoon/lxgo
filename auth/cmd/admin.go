package cmd

import (
	"fmt"

	cvn "github.com/epicoon/lxgo/auth/internal/conventions"
	"github.com/epicoon/lxgo/auth/internal/core"
	"github.com/epicoon/lxgo/auth/internal/models"
	"github.com/epicoon/lxgo/cmd"
)

// AdminCommand manages service administrators - people who operate the auth
// service itself, not to be confused with "client" (an OAuth relying party,
// see ClientCommand). See .claude/tasks/0062.md.
type AdminCommand struct {
	*cmd.Command
	App cvn.IApp
}

func NewAdminCommand(_ ...cmd.ICommandOptions) cmd.ICommand {
	return cmd.Prepare(&AdminCommand{Command: cmd.NewCommand()})
}

func (c *AdminCommand) Config() *cmd.Config {
	return &cmd.Config{
		Description: "Command to manage service administrators",
		Actions: cmd.ActionsConfig{
			"new": cmd.ActionConfig{
				Description: "Bootstrap a new admin - creates the underlying user too, use it to create the first superadmin",
				Executor:    newAdmin,
				Params: cmd.ParamsConfig{
					"login": cmd.ParamConfig{
						Description: "Login for the new admin's user account",
						Type:        cmd.ParamTypeString,
						Required:    true,
					},
					"password": cmd.ParamConfig{
						Description: "Password for the new admin's user account",
						Type:        cmd.ParamTypeString,
						Required:    true,
					},
					"role": cmd.ParamConfig{
						Description: "Admin role name: superadmin or admin",
						Type:        cmd.ParamTypeString,
						Required:    false,
						Default:     "superadmin",
					},
				},
			},
		},
	}
}

/** @type cmd.FAction */
func newAdmin(c cmd.ICommand) error {
	app, err := core.PrepareApp("config.yaml")
	if err != nil {
		return err
	}

	roleName := c.Param("role").(string)
	role := new(models.Role)
	result := app.Gorm().Where("name = ?", roleName).Find(role)
	if result.RowsAffected == 0 {
		return fmt.Errorf("can not find role '%s'. Available are: superadmin, admin", roleName)
	}

	login := c.Param("login").(string)
	password := c.Param("password").(string)

	user, err := app.UsersRepo().Create(login, password)
	if err != nil {
		return fmt.Errorf("can not create user: %s", err)
	}

	if _, err := app.AdminsRepo().Create(user.ID, role.ID); err != nil {
		return fmt.Errorf("can not create admin: %s", err)
	}

	fmt.Printf("New admin has been created. User ID: %d; Login: %s; Role: %s\n", user.ID, login, role.Name)

	return nil
}
