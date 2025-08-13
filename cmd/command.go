package cmd

import (
	"errors"
)

type FConstructor func(opt ...ICommandOptions) ICommand
type FAction func(c ICommand) error
type ActionsList map[string]FAction
type ICommandOptions interface{}

type Config struct {
	Description string
	Params      ParamsConfig
	Actions     ActionsConfig
}

type ParamType string

const (
	ParamTypeString ParamType = "string"
	ParamTypeInt    ParamType = "int"
	ParamTypeBool   ParamType = "bool"
)

type ParamsConfig map[string]ParamConfig

type ParamConfig struct {
	Description string
	Type        ParamType
	Required    bool
	Default     any
	HideDefault bool
}

type ActionsConfig map[string]ActionConfig

type ActionConfig struct {
	Description string
	Executor    FAction
	Params      ParamsConfig
}

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
	Config() *Config
	SetName(name string)
	Name() string
	SetAction(action string)
	Action() string
	Actions() ActionsList
	ActiveAction() FAction
	SetParams(params map[string]any)
	SetParam(key string, val any)
	Params() map[string]any
	HasParam(key string) bool
	Param(key string) any
	Flag(key string) bool
	SetContext(key string, value any)
	Context(key string) (any, bool)
	RegisterActions(ActionsList)
	BeforeExec() error
	Exec() error
}

var ErrNotImplementedExec = errors.New("no exec")

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
	return &Command{
		actions: make(ActionsList, 0),
	}
}

func (c *Command) Config() *Config {
	return nil
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

func (c *Command) Actions() ActionsList {
	return c.actions
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

func (c *Command) SetParam(key string, val any) {
	if c.params == nil {
		c.params = make(map[string]any, 1)
	}
	c.params[key] = val
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
	for key, val := range list {
		c.actions[key] = val
	}
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
