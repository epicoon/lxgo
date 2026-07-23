package cmd

import (
	"fmt"

	cvn "github.com/epicoon/lxgo/auth/internal/conventions"
	"github.com/epicoon/lxgo/auth/internal/core"
	"github.com/epicoon/lxgo/auth/internal/models"
	"github.com/epicoon/lxgo/cmd"
	"github.com/epicoon/lxgo/kernel/utils"
)

type ClientCommand struct {
	*cmd.Command
	App cvn.IApp
}

func NewClientCommand(_ ...cmd.ICommandOptions) cmd.ICommand {
	return cmd.Prepare(&ClientCommand{Command: cmd.NewCommand()})
}

func (c *ClientCommand) Config() *cmd.Config {
	return &cmd.Config{
		Description: "Command to manage OAuth2 clients",
		Actions: cmd.ActionsConfig{
			"new": cmd.ActionConfig{
				Description: "Create a new client",
				Executor:    newClient,
				Params: cmd.ParamsConfig{
					"redirect-uri": cmd.ParamConfig{
						Description: "URI to redirect the user back to after authorization - must match exactly what the client sends to /auth",
						Type:        cmd.ParamTypeString,
						Required:    true,
					},
					"secret": cmd.ParamConfig{
						Description: "Client secret. If not given, a random one is generated and printed once",
						Type:        cmd.ParamTypeString,
						Required:    false,
					},
				},
			},
			"new-secret": cmd.ActionConfig{
				Description: "Regenerate a client's secret",
				Executor:    newClientSecret,
				Params: cmd.ParamsConfig{
					"id": cmd.ParamConfig{
						Description: "Client ID",
						Type:        cmd.ParamTypeInt,
						Required:    true,
					},
				},
			},
			"show": cmd.ActionConfig{
				Description: "Show client data",
				Executor:    showClient,
				Params: cmd.ParamsConfig{
					"id": cmd.ParamConfig{
						Description: "Client ID",
						Type:        cmd.ParamTypeInt,
						Required:    true,
					},
					"secret": cmd.ParamConfig{
						Description: "Client secret",
						Type:        cmd.ParamTypeString,
						Required:    true,
					},
				},
			},
			"del": cmd.ActionConfig{
				Description: "Delete a client",
				Executor:    delClient,
				Params: cmd.ParamsConfig{
					"id": cmd.ParamConfig{
						Description: "Client ID",
						Type:        cmd.ParamTypeInt,
						Required:    true,
					},
				},
			},
		},
	}
}

/** @type cmd.FAction */
func newClient(c cmd.ICommand) error {
	app, err := core.PrepareApp("config.yaml")
	if err != nil {
		return err
	}

	client := &models.Client{
		AccessTokenLifetime:  models.DefaultAccessTokenLifetime,
		RefreshTokenLifetime: models.DefaultRefreshTokenLifetime,
		RedirectUri:          c.Param("redirect-uri").(string),
	}
	if c.HasParam("secret") {
		client.Secret = c.Param("secret").(string)
	}

	secret, err := app.ClientsRepo().Create(client)
	if err != nil {
		return fmt.Errorf("can not create new client: %s", err)
	}

	fmt.Printf("New client has been created. ID: %d; Secret: %s\n", client.ID, secret)

	return nil
}

/** @type cmd.FAction */
func delClient(c cmd.ICommand) error {
	app, err := core.PrepareApp("config.yaml")
	if err != nil {
		return err
	}

	id := c.Param("id").(int)

	if err := app.ClientsRepo().Delete(uint(id)); err != nil {
		return fmt.Errorf("can not delete client: %s", err)
	}

	fmt.Println("Client has been deleted")
	return nil
}

/** @type cmd.FAction */
func showClient(c cmd.ICommand) error {
	app, err := core.PrepareApp("config.yaml")
	if err != nil {
		return err
	}

	id := uint(c.Param("id").(int))
	secret := c.Param("secret").(string)

	client, err := app.ClientsRepo().FindOne(id, secret)
	if err != nil {
		return fmt.Errorf("can not show client data: %s", err)
	}

	fmt.Printf("Client data:\n- id: %d\n- redirect_uri: %s\n- created: %s\n- updated: %s\n",
		client.ID, client.RedirectUri, client.CreatedAt.Format("2006-01-02 15:04:05"), client.UpdatedAt.Format("2006-01-02 15:04:05"))

	return nil
}

/** @type cmd.FAction */
func newClientSecret(c cmd.ICommand) error {
	app, err := core.PrepareApp("config.yaml")
	if err != nil {
		return err
	}

	id := c.Param("id").(int)

	client := &models.Client{}
	result := app.Gorm().Find(&client, id)
	if result.Error != nil {
		return fmt.Errorf("can not find client id=%d: %s", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("client id=%d not found", id)
	}

	secret := client.GenSecret(16)
	secretHash, err := utils.GenHash(secret)
	if err != nil {
		return fmt.Errorf("can not create secret: %s", err)
	}
	client.Secret = secretHash
	app.Gorm().Save(client)

	fmt.Printf("New secret: %s\n", secret)

	return nil
}
