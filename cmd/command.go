package cmd

import (
	"errors"
	"fmt"
)

type FConstructor func(opt ...ICommandOptions) ICommand
type FAction func(c ICommand) error
type ActionsList map[string]FAction
type ICommandOptions interface{}

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

type ICommand interface {
	SetName(name string)
	Name() string
	SetAction(action string)
	Action() string
	ActiveAction() FAction
	SetParams(params map[string]any)
	Params() map[string]any
	HasParam(key string) bool
	Param(key string) any
	Flag(key string) bool
	SetContext(key string, value any)
	Context(key string) (any, bool)
	RegisterActions(ActionsList)
	PrintAvailableActions()
	BeforeExec() error
	Exec() error
}

var ErrNotImplementedExec = errors.New("no exec")

/** @interface ICommand */
type Command struct {
	name    string
	action  string
	params  map[string]any
	context map[string]any
	actions ActionsList
}

/** @constructor */
func NewCommand() *Command {
	return &Command{}
}

func (c *Command) SetName(name string) {
	c.name = name
}

func (c *Command) Name() string {
	return c.name
}

func (c *Command) SetAction(action string) {
	c.action = action
}

func (c *Command) Action() string {
	return c.action
}

func (c *Command) ActiveAction() FAction {
	if c.action == "" {
		return nil
	}
	a, ok := c.actions[c.action]
	if !ok {
		return nil
	}
	return a
}

func (c *Command) SetParams(params map[string]any) {
	c.params = params
}

func (c *Command) Params() map[string]any {
	return c.params
}

func (c *Command) HasParam(key string) bool {
	_, exists := c.params[key]
	return exists
}

func (c *Command) Param(key string) any {
	val, ok := c.params[key]
	if !ok {
		return nil
	}
	return val
}

func (c *Command) Flag(key string) bool {
	return c.HasParam(key)
}

func (c *Command) SetContext(key string, value any) {
	if c.context == nil {
		c.context = make(map[string]any)
	}
	c.context[key] = value
}

func (c *Command) Context(key string) (any, bool) {
	val, exists := c.context[key]
	return val, exists
}

func (c *Command) RegisterActions(list ActionsList) {
	c.actions = list
}

/** @abstract */
func (c *Command) BeforeExec() error {
	// pass
	return nil
}

/** @abstract */
func (c *Command) Exec() error {
	// pass
	return ErrNotImplementedExec
}

func (c *Command) PrintAvailableActions() {
	fmt.Printf("Available actions for command '%s':\n", c.name)
	for action := range c.actions {
		fmt.Printf("  - %s\n", action)
	}
}
