package cmd

import (
	"errors"
	"fmt"
	"strconv"

	cvn "github.com/epicoon/lxgo/auth/internal/conventions"
	"github.com/epicoon/lxgo/auth/internal/core"
	"github.com/epicoon/lxgo/auth/internal/models"
	"github.com/epicoon/lxgo/cmd"
	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/conv"
	"github.com/epicoon/lxgo/kernel/utils"
)

type ClientCommand struct {
	*cmd.Command
	App cvn.IApp
}

func NewClientCommand(_ ...cmd.ICommandOptions) cmd.ICommand {
	c := &ClientCommand{Command: &cmd.Command{}}
	c.RegisterActions(cmd.ActionsList{
		"new":        newClient,
		"new-secret": newClientSecret,
		"show":       showClient,
		"del":        delClient,
	})
	return c
}

/** @type cmd.FAction */
func newClient(c cmd.ICommand) error {
	app, err := core.PrepareApp("config.yaml")
	if err != nil {
		return err
	}
	gorm := app.Gorm()

	var roleName string
	if c.HasParam("role") {
		param, ok := c.Param("role").(string)
		if !ok {
			return errors.New("parameter 'role' must be string")
		}
		roleName = param
	} else {
		roleName = "client"
	}

	role := new(models.Role)
	result := gorm.Where("name = ?", roleName).Find(role)
	if result.RowsAffected == 0 {
		return fmt.Errorf("can not find role '%s'. Available are: superadmin, admin, viewer, client", roleName)
	}

	client := &models.Client{
		RoleID:               role.ID,
		AccessTokenLifetime:  900,
		RefreshTokenLifetime: 604800,
	}

	if c.HasParam("seret") {
		param, ok := c.Param("secret").(string)
		if !ok {
			return errors.New("paraneter 'secret' must be string")
		}
		client.Secret = param
	}

	secret, err := app.ClientsRepo().Create(client)
	if err != nil {
		return fmt.Errorf("can cot create new client: %s", err)
	}

	fmt.Printf("New client has been created. ID: %d; Secret: %s; Role: %s\n", client.ID, secret, role.Name)

	return nil
}

/** @type cmd.FAction */
func delClient(c cmd.ICommand) error {
	app, err := core.PrepareApp("config.yaml")
	if err != nil {
		return err
	}

	var id int
	if c.HasParam("id") {
		paramStr, ok := c.Param("id").(string)
		if !ok {
			return errors.New("paraneter 'id' must be int")
		}
		param, err := strconv.Atoi(paramStr)
		if err != nil {
			return fmt.Errorf("paraneter 'id' must be int: %s", err)
		}
		id = param
	}
	if id == 0 {
		return errors.New("enter 'id' parameter")
	}

	result := app.Gorm().Delete(&models.Client{}, id)
	if result.Error != nil {
		return fmt.Errorf("can not delete client: %s", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("client with ID %d not found", id)
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

	paramsSrc := c.Params()
	params := &struct {
		Id     uint   `dict:"id"`
		Secret string `dict:"secret"`
	}{}
	err = conv.DictToStruct((*kernel.Dict)(&paramsSrc), params)
	if err != nil {
		return fmt.Errorf("can not show client data: %s", err)
	}

	if params.Id == 0 || params.Secret == "" {
		return errors.New("enter 'id' and 'secret' parameter")
	}

	client, err := app.ClientsRepo().FindOne(params.Id, params.Secret)
	if err != nil {
		return fmt.Errorf("can not show client data: %s", err)
	}

	fmt.Printf("Client data:\n- id: %d\n- role: %s\n- created: %s\n- updated: %s\n",
		client.ID, client.Role.Name, client.CreatedAt.Format("2006-01-02 15:04:05"), client.UpdatedAt.Format("2006-01-02 15:04:05"))

	return nil
}

/** @type cmd.FAction */
func newClientSecret(c cmd.ICommand) error {
	app, err := core.PrepareApp("config.yaml")
	if err != nil {
		return err
	}

	paramsSrc := c.Params()
	params := &struct {
		Id int `dict:"id"`
	}{}
	err = conv.DictToStruct((*kernel.Dict)(&paramsSrc), params)
	if err != nil {
		return fmt.Errorf("can not show client data: %s", err)
	}
	if params.Id == 0 {
		return errors.New("enter 'id' parameter")
	}

	client := &models.Client{}
	result := app.Gorm().Find(&client, params.Id)
	if result.Error != nil {
		return fmt.Errorf("can not find client id=%d: %s", params.Id, err)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("client id=%d not found", params.Id)
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
