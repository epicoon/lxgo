package cmd

import (
	"errors"
	"fmt"
	"strconv"
)

// Init registers the application's commands, keyed by name (used as
// "command" or "command:action" on the command line) - call this once
// before Run, typically with the empty-string key for the default command.
func Init(cmds CommandsList) {
	m.list = cmds
}

// Run parses os.Args, resolves the matching command/action from the
// CommandsList passed to Init, and executes it - call this from main.
func Run() {
	m.prepare()
	cc, err := m.defineConstructor()
	if err != nil {
		fmt.Printf("Can not exec: %s\n", err)
		return
	}

	help := false
	_, exists := m.params["help"]
	if exists {
		delete(m.params, "help")
		help = true
	}

	command := cc()
	command.SetName(m.cmdName)
	command.SetAction(m.subName)
	command.SetParams(m.params)

	if help {
		if m.subName == "" {
			printInfo(command)
		} else {
			printActionInfo(command, m.subName)
		}
		return
	}

	err = command.BeforeExec()
	if err != nil {
		fmt.Printf("Can not exec: %s\n", err)
		return
	}

	err = command.Exec()
	if errors.Is(ErrNotImplementedExec, err) {
		if command.Action() == "" {
			printInfo(command)
			return
		}

		action := command.ActiveAction()
		if action == nil {
			fmt.Printf("Action handler is undefined: %s\n", command.Action())
			return
		}

		if err := validate(command); err != nil {
			fmt.Printf("Can not execute the command: %v\n", err)
			return
		}

		err := action(command)
		if err != nil {
			fmt.Printf("Error occurred while action executing: %s\n", err)
			return
		}
		return
	}
	if err != nil {
		fmt.Printf("Error occurred while action executing: %s\n", err)
	}
}

// GetOptions type-asserts the first element of opt to T, returning T's zero
// value if opt is empty or its first element isn't a T - use this in a
// command constructor to pull out its ICommandOptions.
func GetOptions[T any](opt []ICommandOptions) T {
	if len(opt) > 0 && opt[0] != nil {
		res, ok := opt[0].(T)
		if !ok {
			//TODO log to fmt
			return *new(T)
		}
		return res
	}
	return *new(T)
}

// Prepare registers c's Config.Actions executors onto c via RegisterActions
// - call this after constructing a command that declares actions through Config.
func Prepare(c ICommand) ICommand {
	conf := c.Config()
	if conf == nil {
		return c
	}

	actionsLen := len(conf.Actions)
	if actionsLen > 0 {
		al := make(ActionsList, actionsLen)
		for key, val := range conf.Actions {
			if val.Executor != nil {
				al[key] = val.Executor
			}
		}
		if len(al) > 0 {
			c.RegisterActions(al)
		}
	}

	return c
}

func validate(c ICommand) error {
	conf := c.Config()
	if conf == nil {
		return nil
	}

	var paramsConf ParamsConfig
	if c.Action() == "" {
		paramsConf = conf.Params
	} else {
		actions := conf.Actions
		ac, exists := actions[c.Action()]
		if exists {
			paramsConf = ac.Params
		}
	}
	if len(paramsConf) == 0 {
		return nil
	}

	for pName, pConf := range paramsConf {
		if c.HasParam(pName) {
			val := c.Param(pName)
			var ok bool
			switch pConf.Type {
			case ParamTypeString:
				_, ok = val.(string)
			case ParamTypeInt:
				_, ok = val.(int)
				if !ok {
					s, ok2 := val.(string)
					if ok2 {
						i, err := strconv.Atoi(s)
						if err == nil {
							c.SetParam(pName, i)
							ok = true
						}
					}
				}
			case ParamTypeBool:
				switch val {
				case "false":
					c.SetParam(pName, false)
				case "true":
					c.SetParam(pName, true)
				}
				val = c.Param(pName)
				_, ok = val.(bool)
			case ParamTypeEnum:
				if pConf.ElemType == ParamTypeInt {
					_, ok = val.(int)
					if !ok {
						s, ok2 := val.(string)
						if ok2 {
							i, err := strconv.Atoi(s)
							if err == nil {
								c.SetParam(pName, i)
								ok = true
							}
						}
					}
				} else {
					_, ok = val.(string)
				}
			}
			if !ok {
				return fmt.Errorf("expected type for 'parameter' - %s, received - %T", pConf.Type, val)
			}
		}

		if !c.HasParam(pName) {
			if pConf.Required {
				if pConf.Interactive || c.Flag("interactive") {
					val, err := promptForParam(c, pName, pConf)
					if err != nil {
						return fmt.Errorf("can not read parameter '%s' interactively: %w", pName, err)
					}
					c.SetParam(pName, val)
					continue
				}
				return fmt.Errorf("parameter '%s' required. Expected type - %s", pName, pConf.Type)
			}

			var def any
			var ok bool
			switch pConf.Type {
			case ParamTypeString:
				def, ok = pConf.Default.(string)
			case ParamTypeInt:
				def, ok = pConf.Default.(int)
			case ParamTypeBool:
				def, ok = pConf.Default.(bool)
			}
			if ok {
				c.SetParam(pName, def)
			}
		}
	}

	return nil
}

func printInfo(c ICommand) {
	conf := c.Config()

	if conf != nil {
		fmt.Printf("Description: %s\n", conf.Description)
	}

	fmt.Printf("Available actions for command '%s':\n", c.Name())
	for action := range c.Actions() {
		fmt.Printf("  - %s\n", action)
	}

	if conf == nil {
		return
	}

	al := conf.Actions
	if len(al) > 0 {
		fmt.Println("Details:")
		for an := range al {
			printActionInfo(c, an)
		}
	}
}

func printActionInfo(c ICommand, action string) {
	conf := c.Config()
	if conf == nil {
		if checkActionExists(c, action) {
			fmt.Println("Action exists but information not found")
		} else {
			fmt.Println("Action does not exist")
		}
		return
	}

	al := conf.Actions
	ac, exists := al[action]
	if !exists {
		if checkActionExists(c, action) {
			fmt.Println("Action exists but information not found")
		} else {
			fmt.Println("Action does not exist")
		}
		return
	}

	fmt.Printf("Action '%s':\n", action)

	if ac.Description != "" {
		fmt.Printf("  Description: %s\n", ac.Description)
	}

	if len(ac.Params) > 0 {
		fmt.Println("  Params:")
		for pn, pc := range ac.Params {
			fmt.Printf("    - %s:\n", pn)

			if pc.Description != "" {
				fmt.Printf("        Description: %s\n", pc.Description)
			}

			fmt.Printf("        Type: %s\n", pc.Type)

			if pc.Required {
				fmt.Println("        Required: true")
			} else {
				fmt.Println("        Required: false")
				if !pc.HideDefault {
					fmt.Printf("        Default value: %v\n", pc.Default)
				}
			}
		}
	}
}

func checkActionExists(c ICommand, action string) bool {
	aa := c.Actions()
	_, exists := aa[action]
	return exists
}
