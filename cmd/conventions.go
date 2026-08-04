// Package cmd builds console commands for lxgo/kernel applications: define
// an ICommand (usually by embedding Command), register it under a name in a
// CommandsList passed to Init, and call Run from main to dispatch
// os.Args into the matching command/action.
package cmd

import "errors"

// CommandsList maps command names to their constructors - see Init.
type CommandsList map[string]CCommand

// CCommand constructs an ICommand - the signature every command constructor
// must have, so any of them can be plugged into a CommandsList uniformly.
type CCommand func(opt ...ICommandOptions) ICommand

// FAction implements one command action - see ActionConfig.Executor.
type FAction func(c ICommand) error

// ActionsList maps action names to their FAction implementations - see
// Command.RegisterActions.
type ActionsList map[string]FAction

// ICommandOptions is a generic "some options, typed by whoever's
// constructing this specific command" placeholder, passed to a CCommand.
// Every command constructor has the same signature so it can be plugged
// into a CommandsList uniformly, but different commands often need
// different construction-time data (e.g. a reference to the app instance) -
// ICommandOptions (just interface{}, so any struct satisfies it) is how
// this package works around that. Define your own options struct and pull
// it out with GetOptions.
type ICommandOptions interface{}

// Config describes a command for both validation and the auto-generated
// --help output - see ICommand.Config.
type Config struct {
	// Description is shown in the auto-generated --help output.
	Description string
	// Params describes the command's own parameters (used when it's called
	// with no action).
	Params ParamsConfig
	// Actions describes the command's actions - each key is an action name
	// usable as "command:action".
	Actions ActionsConfig
}

// ParamType is a command/action parameter's expected type - see ParamConfig.Type.
type ParamType string

const (
	// ParamTypeString marks a parameter as a string.
	ParamTypeString ParamType = "string"
	// ParamTypeInt marks a parameter as an int.
	ParamTypeInt ParamType = "int"
	// ParamTypeBool marks a parameter as a bool.
	ParamTypeBool ParamType = "bool"
	// ParamTypeEnum marks a parameter whose value must be one of a fixed
	// set of options - see ParamConfig.TypeDetails/FTypeDetails and ElemType.
	ParamTypeEnum ParamType = "enum"
)

// ParamsConfig maps parameter names to their ParamConfig - see Config.Params, ActionConfig.Params.
type ParamsConfig map[string]ParamConfig

// ParamConfig describes one command/action parameter.
type ParamConfig struct {
	// Description is shown in the auto-generated --help output.
	Description string
	// Type is the parameter's expected type.
	Type ParamType
	// TypeDetails holds Type-specific extra data. For ParamTypeEnum, it's
	// the allowed values - a []string, or a []int when ElemType is
	// ParamTypeInt. Ignored if FTypeDetails is set.
	TypeDetails any
	// FTypeDetails computes TypeDetails lazily - e.g. by scanning the
	// filesystem for a ParamTypeEnum parameter's allowed values. Only
	// called when the parameter is actually about to be prompted for
	// interactively (missing, Required, and going interactive - see
	// Interactive) - never eagerly, and never at all for a parameter
	// that was supplied on the command line or isn't Required, so an
	// expensive lookup only runs when its result is actually needed.
	FTypeDetails func(c ICommand) (any, error)
	// ElemType is a ParamTypeEnum parameter's element type - ParamTypeString
	// (the default) or ParamTypeInt. Ignored for every other Type.
	ElemType ParamType
	// Required fails validation if the parameter isn't passed.
	Required bool
	// Interactive marks a Required parameter as always prompted for on
	// stdin when missing, without needing the caller to pass
	// --interactive - set this by the command author when a parameter is
	// inherently something a human fills in (e.g. a name to scaffold),
	// so the caller doesn't need to know or care that a dialog exists.
	// The --interactive flag still works as a manual override for
	// parameters that don't set this.
	Interactive bool
	// Default is used when the parameter isn't passed and it's not Required.
	Default any
	// HideDefault omits Default from the auto-generated --help output.
	HideDefault bool
}

// ActionsConfig maps action names to their ActionConfig - see Config.Actions.
type ActionsConfig map[string]ActionConfig

// ActionConfig describes one command action.
type ActionConfig struct {
	// Description is shown in the auto-generated --help output.
	Description string
	// Executor runs the action.
	Executor FAction
	// Params describes the action's own parameters.
	Params ParamsConfig
}

// ICommand is a console command, dispatched to by Run - usually implemented
// by embedding Command and defining at least Config or Exec.
type ICommand interface {
	// Config returns the command's Config, or nil if it doesn't need one
	// (no validation, no auto-generated --help details).
	Config() *Config

	// SetName sets the command's name - called by Run before dispatch.
	SetName(name string)

	// Name returns the command's name.
	Name() string

	// SetAction sets the action to run - called by Run before dispatch.
	SetAction(action string)

	// Action returns the action to run, or "" if the command was called
	// with no action.
	Action() string

	// Actions returns the command's registered actions - see RegisterActions.
	Actions() ActionsList

	// ActiveAction returns the FAction registered for Action, or nil if
	// there isn't one.
	ActiveAction() FAction

	// SetParams replaces all of the command's parameters - called by Run
	// with the parsed command-line arguments.
	SetParams(params map[string]any)

	// SetParam sets a single parameter.
	SetParam(key string, val any)

	// Params returns all of the command's parameters.
	Params() map[string]any

	// HasParam reports whether key is set.
	HasParam(key string) bool

	// Param returns the value of key, or nil if it isn't set.
	Param(key string) any

	// Flag reports whether key was passed as a boolean flag (e.g. -v) - equivalent to HasParam.
	Flag(key string) bool

	// SetContext stores a value in the command's execution context, shared
	// across BeforeExec/Exec/action calls.
	SetContext(key string, value any)

	// Context returns a value from the command's execution context.
	Context(key string) (any, bool)

	// RegisterActions adds actions to the command, keyed by name.
	RegisterActions(ActionsList)

	// BeforeExec runs before Exec/the active action - return an error to
	// abort execution.
	BeforeExec() error

	// Exec runs the command itself, for commands with no actions. Return
	// ErrNotImplementedExec (the default via Command) to fall through to
	// action-based dispatch instead.
	Exec() error
}

// ErrNotImplementedExec signals that a command has no Exec of its own and
// should fall through to its active action instead - see Command.Exec.
var ErrNotImplementedExec = errors.New("no exec")
