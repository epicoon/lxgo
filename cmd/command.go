package cmd

/** @interface ICommand */

// Command is the base ICommand implementation - embed it in your own
// command struct and override at least Config or Exec.
type Command struct {
	name    string
	action  string
	params  map[string]any
	context map[string]any
	actions ActionsList
}

/** @constructor */

// NewCommand constructs an empty Command.
func NewCommand() *Command {
	return &Command{
		actions: make(ActionsList, 0),
	}
}

// Config returns nil - override this to declare parameters/actions.
func (c *Command) Config() *Config {
	return nil
}

// SetName sets the command's name.
func (c *Command) SetName(name string) {
	c.name = name
}

// Name returns the command's name.
func (c *Command) Name() string {
	return c.name
}

// SetAction sets the action to run.
func (c *Command) SetAction(action string) {
	c.action = action
}

// Action returns the action to run, or "" if the command was called with no action.
func (c *Command) Action() string {
	return c.action
}

// Actions returns the command's registered actions.
func (c *Command) Actions() ActionsList {
	return c.actions
}

// ActiveAction returns the FAction registered for the current Action, or nil if there isn't one.
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

// SetParams replaces all of the command's parameters.
func (c *Command) SetParams(params map[string]any) {
	c.params = params
}

// SetParam sets a single parameter.
func (c *Command) SetParam(key string, val any) {
	if c.params == nil {
		c.params = make(map[string]any, 1)
	}
	c.params[key] = val
}

// Params returns all of the command's parameters.
func (c *Command) Params() map[string]any {
	return c.params
}

// HasParam reports whether key is set.
func (c *Command) HasParam(key string) bool {
	_, exists := c.params[key]
	return exists
}

// Param returns the value of key, or nil if it isn't set.
func (c *Command) Param(key string) any {
	val, ok := c.params[key]
	if !ok {
		return nil
	}
	return val
}

// Flag reports whether key was passed as a boolean flag - equivalent to HasParam.
func (c *Command) Flag(key string) bool {
	return c.HasParam(key)
}

// SetContext stores a value in the command's execution context.
func (c *Command) SetContext(key string, value any) {
	if c.context == nil {
		c.context = make(map[string]any)
	}
	c.context[key] = value
}

// Context returns a value from the command's execution context.
func (c *Command) Context(key string) (any, bool) {
	val, exists := c.context[key]
	return val, exists
}

// RegisterActions adds actions to the command, keyed by name.
func (c *Command) RegisterActions(list ActionsList) {
	for key, val := range list {
		c.actions[key] = val
	}
}

/** @abstract */

// BeforeExec is a no-op - override it to run logic before Exec/the active action.
func (c *Command) BeforeExec() error {
	// pass
	return nil
}

/** @abstract */

// Exec returns ErrNotImplementedExec, falling through to action-based
// dispatch - override it for a command with no actions.
func (c *Command) Exec() error {
	// pass
	return ErrNotImplementedExec
}
