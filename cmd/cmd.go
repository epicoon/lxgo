package cmd

import (
	"errors"
	"fmt"
)

type CommandsList map[string]FConstructor

func Init(cmds CommandsList) {
	m.list = cmds
}

func Run() {
	m.prepare()
	cc, err := m.defineConstructor()
	if err != nil {
		fmt.Printf("Can not exec: %s\n", err)
		return
	}

	command := cc()
	command.SetName(m.cmdName)
	command.SetAction(m.subName)
	command.SetParams(m.params)

	err = command.BeforeExec()
	if err != nil {
		fmt.Printf("Can not exec: %s\n", err)
		return
	}

	err = command.Exec()
	if errors.Is(ErrNotImplementedExec, err) {
		if command.Action() == "" {
			command.PrintAvailableActions()
			return
		}
		action := command.ActiveAction()
		if action == nil {
			fmt.Printf("Action handler is undefined: %s\n", command.Action())
			return
		}
		err := action(command)
		if err != nil {
			fmt.Printf("Error occured white action executing: %s\n", err)
			return
		}
		return
	}
	if err != nil {
		fmt.Printf("Error occured white action executing: %s\n", err)
	}
}
