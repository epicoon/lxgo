package cmd

import (
	"fmt"
	"net"

	"github.com/epicoon/lxgo/cmd"
)

type ManageCommandOptions struct {
	SocketPath string
}

/** @interface cmd.ICommand */
type ManageCommand struct {
	*cmd.Command
	SocketPath string
}

/** @type cmd.FConstructor */
func NewManageCommand(opt ...cmd.ICommandOptions) cmd.ICommand {
	options := cmd.GetOptions[ManageCommandOptions](opt)
	return cmd.Prepare(&ManageCommand{
		Command:    cmd.NewCommand(),
		SocketPath: options.SocketPath,
	})
}

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
				Description: "Trigger custom event (!NOT IMPLEMENTED YET!)",
				Executor:    trigger,
				Params: cmd.ParamsConfig{
					"event": cmd.ParamConfig{
						Description: "Event name",
						Type:        cmd.ParamTypeString,
						Required:    true,
					},
				},
			},
		},
	}
}

func (c *ManageCommand) BeforeExec() error {
	fmt.Println("Send message to socket '" + c.SocketPath + "'...")
	return nil
}

func status(c cmd.ICommand) error {
	sendToSocket(c.(*ManageCommand).SocketPath, "status")
	return nil
}

func refreshConfig(c cmd.ICommand) error {
	sendToSocket(c.(*ManageCommand).SocketPath, prepareMsg("reconf", c.Params()))
	return nil
}

func injectConfig(c cmd.ICommand) error {
	sendToSocket(c.(*ManageCommand).SocketPath, prepareMsg("inconf", c.Params()))
	return nil
}

func trigger(c cmd.ICommand) error {
	e := c.Param("event")

	//TODO
	_ = e
	fmt.Println("Not implemented yet")

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
