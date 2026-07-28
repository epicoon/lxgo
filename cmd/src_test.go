package cmd

import "testing"

func TestGetOptions_Present(t *testing.T) {
	type opts struct{ Name string }
	got := GetOptions[opts]([]ICommandOptions{opts{Name: "hi"}})
	if got.Name != "hi" {
		t.Fatalf("got = %#v", got)
	}
}

func TestGetOptions_Empty(t *testing.T) {
	type opts struct{ Name string }
	got := GetOptions[opts](nil)
	if got.Name != "" {
		t.Fatalf("got = %#v, want the zero value", got)
	}
}

func TestGetOptions_WrongType(t *testing.T) {
	type opts struct{ Name string }
	got := GetOptions[opts]([]ICommandOptions{42})
	if got.Name != "" {
		t.Fatalf("got = %#v, want the zero value on a type mismatch", got)
	}
}

func TestPrepare_RegistersActionsFromConfig(t *testing.T) {
	called := false
	c := &testCommandWithConfig{
		Command: NewCommand(),
		conf: &Config{
			Actions: ActionsConfig{
				"greet": ActionConfig{Executor: func(ICommand) error { called = true; return nil }},
			},
		},
	}

	Prepare(c)

	c.SetAction("greet")
	action := c.ActiveAction()
	if action == nil {
		t.Fatal("expected Prepare to register the 'greet' action")
	}
	if err := action(c); err != nil {
		t.Fatalf("action: %v", err)
	}
	if !called {
		t.Fatal("expected the registered executor to run")
	}
}

func TestPrepare_NilConfig_NoOp(t *testing.T) {
	c := NewCommand()
	got := Prepare(c)
	if got != ICommand(c) {
		t.Fatal("expected Prepare to return the same command unchanged")
	}
	if len(c.Actions()) != 0 {
		t.Fatalf("expected no actions registered, got %#v", c.Actions())
	}
}

type testCommandWithConfig struct {
	*Command
	conf *Config
}

func (c *testCommandWithConfig) Config() *Config { return c.conf }

func newValidateCommand(conf *Config, action string, params map[string]any) ICommand {
	c := &testCommandWithConfig{Command: NewCommand(), conf: conf}
	c.SetAction(action)
	c.SetParams(params)
	return c
}

func TestValidate_NilConfig_NoOp(t *testing.T) {
	c := NewCommand()
	if err := validate(c); err != nil {
		t.Fatalf("validate: %v", err)
	}
}

func TestValidate_RequiredParamMissing(t *testing.T) {
	conf := &Config{Params: ParamsConfig{"name": ParamConfig{Type: ParamTypeString, Required: true}}}
	c := newValidateCommand(conf, "", map[string]any{})

	if err := validate(c); err == nil {
		t.Fatal("expected an error for a missing required parameter")
	}
}

func TestValidate_OptionalParamMissing_AppliesDefault(t *testing.T) {
	conf := &Config{Params: ParamsConfig{"count": ParamConfig{Type: ParamTypeInt, Default: 5}}}
	c := newValidateCommand(conf, "", map[string]any{})

	if err := validate(c); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if got := c.Param("count"); got != 5 {
		t.Fatalf("count = %#v, want the default 5", got)
	}
}

func TestValidate_IntParam_CoercesStringFromCLI(t *testing.T) {
	conf := &Config{Params: ParamsConfig{"count": ParamConfig{Type: ParamTypeInt}}}
	c := newValidateCommand(conf, "", map[string]any{"count": "42"})

	if err := validate(c); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if got := c.Param("count"); got != 42 {
		t.Fatalf("count = %#v (%T), want int 42", got, got)
	}
}

func TestValidate_IntParam_RejectsNonNumericString(t *testing.T) {
	conf := &Config{Params: ParamsConfig{"count": ParamConfig{Type: ParamTypeInt}}}
	c := newValidateCommand(conf, "", map[string]any{"count": "abc"})

	if err := validate(c); err == nil {
		t.Fatal("expected an error for a non-numeric int parameter")
	}
}

func TestValidate_BoolParam_CoercesStringFromCLI(t *testing.T) {
	conf := &Config{Params: ParamsConfig{"verbose": ParamConfig{Type: ParamTypeBool}}}
	c := newValidateCommand(conf, "", map[string]any{"verbose": "true"})

	if err := validate(c); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if got := c.Param("verbose"); got != true {
		t.Fatalf("verbose = %#v, want true", got)
	}
}

func TestValidate_BoolParam_BareFlagAlreadyBool(t *testing.T) {
	conf := &Config{Params: ParamsConfig{"verbose": ParamConfig{Type: ParamTypeBool}}}
	c := newValidateCommand(conf, "", map[string]any{"verbose": true})

	if err := validate(c); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if got := c.Param("verbose"); got != true {
		t.Fatalf("verbose = %#v, want true", got)
	}
}

func TestValidate_StringParam_WrongType(t *testing.T) {
	conf := &Config{Params: ParamsConfig{"name": ParamConfig{Type: ParamTypeString}}}
	c := newValidateCommand(conf, "", map[string]any{"name": 5})

	if err := validate(c); err == nil {
		t.Fatal("expected an error for a non-string value on a string parameter")
	}
}

// TestValidate_UsesActionParams_NotCommandParams confirms validate looks at
// the active action's own Params, not the command-level ones, once an
// action is set.
func TestValidate_UsesActionParams_NotCommandParams(t *testing.T) {
	conf := &Config{
		Params: ParamsConfig{"onlyOnCommand": ParamConfig{Type: ParamTypeString, Required: true}},
		Actions: ActionsConfig{
			"create": ActionConfig{
				Params: ParamsConfig{"name": ParamConfig{Type: ParamTypeString, Required: true}},
			},
		},
	}
	c := newValidateCommand(conf, "create", map[string]any{"name": "x"})

	if err := validate(c); err != nil {
		t.Fatalf("validate: %v (command-level required param shouldn't apply to an action call)", err)
	}
}
