// Package cmd provides console commands for managing a running lxgo/kernel
// application over its manage socket (ManageCommand) and for generating its
// API documentation (see gen_api_doc.go).
package cmd

import (
	"fmt"
	"net"

	"github.com/epicoon/lxgo/cmd"
)

// ManageCommandOptions is ManageCommand's cmd.ICommandOptions - pass the
// same socket path the target app is configured with (its "ManageSocket" config key).
type ManageCommandOptions struct {
	SocketPath string
}

/** @interface cmd.ICommand */

// ManageCommand talks to a running application's manage socket - status/
// refresh-config/inject-config/trigger - see NewManageCommand.
type ManageCommand struct {
	*cmd.Command
	SocketPath string
}

var _ cmd.ICommand = (*ManageCommand)(nil)

/** @constructor cmd.CCommand */

// NewManageCommand constructs a ManageCommand - pass its socket path via
// ManageCommandOptions.
func NewManageCommand(opt ...cmd.ICommandOptions) cmd.ICommand {
	options := cmd.GetOptions[ManageCommandOptions](opt)
	return cmd.Prepare(&ManageCommand{
		Command:    cmd.NewCommand(),
		SocketPath: options.SocketPath,
	})
}

// Config declares the "status", "refresh-config", "inject-config" and
// "trigger" actions - see cmd.ICommand.
func (c *ManageCommand) Config() *cmd.Config {
	return &cmd.Config{
		Description: "Command for local app managing by socket file defined in the config param 'ManageSocket'",
		Actions: cmd.ActionsConfig{
			"status": cmd.ActionConfig{
				Description: "Action to be sure the command is ok. Answeres 'ok'",
				Executor:    status,
			},
			"refresh-config": cmd.ActionConfig{
				Description: "Refresh app config without restart",
				Executor:    refreshConfig,
				Params: cmd.ParamsConfig{
					"t": cmd.ParamConfig{
						Description: "Before refresh config test it if config is invalid",
						Type:        cmd.ParamTypeBool,
						Required:    false,
						Default:     false,
					},
				},
			},
			"inject-config": cmd.ActionConfig{
				Description: "Change app config params without restart",
				Executor:    injectConfig,
				Params: cmd.ParamsConfig{
					"t": cmd.ParamConfig{
						Description: "Before change config test it if config is invalid",
						Type:        cmd.ParamTypeBool,
						Required:    false,
						Default:     false,
					},
					"params": cmd.ParamConfig{
						Description: "List of parameters to be refreshed, example: --params=\"number:123,name:'some string'\"",
						Type:        cmd.ParamTypeString,
						Required:    false,
					},
					"add": cmd.ParamConfig{
						Description: "Add parameters to an array, example: --add=\"arrayName:[newElem1,newELem2],arrayName2:[newElem1,newELem2]\"",
						Type:        cmd.ParamTypeString,
						Required:    false,
					},
					"remove": cmd.ParamConfig{
						Description: "Remove parameters from an array, example: --remove=\"arrayName:[newElem1,newELem2]\"",
						Type:        cmd.ParamTypeString,
						Required:    false,
					},
				},
			},
			"trigger": cmd.ActionConfig{
				Description: "Trigger a custom app event (app.Events().Trigger)",
				Executor:    trigger,
				Params: cmd.ParamsConfig{
					"event": cmd.ParamConfig{
						Description: "Event name",
						Type:        cmd.ParamTypeString,
						Required:    true,
					},
					"params": cmd.ParamConfig{
						Description: "Event payload, example: --params=\"number:123,name:'some string'\"",
						Type:        cmd.ParamTypeString,
						Required:    false,
					},
				},
			},
		},
	}
}

// BeforeExec announces which socket the command is about to talk to - see cmd.ICommand.
func (c *ManageCommand) BeforeExec() error {
	fmt.Println("Send message to socket '" + c.SocketPath + "'...")
	return nil
}

/** @handler cmd.FAction */
func status(c cmd.ICommand) error {
	sendToSocket(c.(*ManageCommand).SocketPath, "status")
	return nil
}

/** @handler cmd.FAction */
func refreshConfig(c cmd.ICommand) error {
	sendToSocket(c.(*ManageCommand).SocketPath, prepareMsg("reconf", c.Params()))
	return nil
}

/** @handler cmd.FAction */
func injectConfig(c cmd.ICommand) error {
	sendToSocket(c.(*ManageCommand).SocketPath, prepareMsg("inconf", c.Params()))
	return nil
}

/** @handler cmd.FAction */
func trigger(c cmd.ICommand) error {
	sendToSocket(c.(*ManageCommand).SocketPath, prepareMsg("trigger", c.Params()))
	return nil
}

func prepareMsg(command string, params map[string]any) string {
	msg := command
	for key, val := range params {
		msg += "&&" + key + "=" + fmt.Sprintf("%v", val)
	}
	return msg
}

func sendToSocket(socketPath, msg string) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		fmt.Printf("connect error: %v\n", err)
		return
	}
	defer conn.Close()

	_, _ = conn.Write([]byte(msg + "\n"))
	buf := make([]byte, 1024)
	n, _ := conn.Read(buf)
	fmt.Println(string(buf[:n]))
}
