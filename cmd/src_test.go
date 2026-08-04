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

// TestValidate_RequiredParamMissing_InteractiveFlag_PromptsInsteadOfErroring
// checks the "interactive" flag path: a missing required parameter is
// read from stdin instead of failing validate outright, and the flag being
// absent still fails fast (no accidental stdin block in the common case).
func TestValidate_RequiredParamMissing_InteractiveFlag_PromptsInsteadOfErroring(t *testing.T) {
	conf := &Config{Params: ParamsConfig{"name": ParamConfig{Type: ParamTypeString, Required: true}}}
	c := newValidateCommand(conf, "", map[string]any{"interactive": true})

	withStdinInput(t, "Alice\n")
	if err := validate(c); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if got := c.Param("name"); got != "Alice" {
		t.Fatalf("name = %#v, want the interactively-entered Alice", got)
	}
}

// TestValidate_RequiredParamMissing_ParamConfigInteractive_PromptsWithoutFlag
// checks the ParamConfig.Interactive path: a missing required parameter is
// prompted for even though the caller never passed --interactive.
func TestValidate_RequiredParamMissing_ParamConfigInteractive_PromptsWithoutFlag(t *testing.T) {
	conf := &Config{Params: ParamsConfig{"name": ParamConfig{Type: ParamTypeString, Required: true, Interactive: true}}}
	c := newValidateCommand(conf, "", map[string]any{})

	withStdinInput(t, "Alice\n")
	if err := validate(c); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if got := c.Param("name"); got != "Alice" {
		t.Fatalf("name = %#v, want the interactively-entered Alice", got)
	}
}

// TestValidate_EnumParam_SuppliedValue_DoesNotCallFTypeDetails checks that
// an already-supplied enum value skips FTypeDetails entirely - an
// expensive lookup (e.g. a filesystem scan) behind it should never run
// just to type-check a value the caller already gave.
func TestValidate_EnumParam_SuppliedValue_DoesNotCallFTypeDetails(t *testing.T) {
	called := false
	conf := &Config{Params: ParamsConfig{"path": ParamConfig{
		Type:     ParamTypeEnum,
		Required: true,
		FTypeDetails: func(ICommand) (any, error) {
			called = true
			return []string{"a", "b"}, nil
		},
	}}}
	c := newValidateCommand(conf, "", map[string]any{"path": "a"})

	if err := validate(c); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if called {
		t.Fatal("FTypeDetails must not be called when the parameter was already supplied")
	}
}

// TestValidate_EnumParam_MissingRequiredNotInteractive_DoesNotCallFTypeDetails
// checks the same for the plain-error path: a missing required enum
// parameter that isn't going interactive fails fast, without ever
// resolving its options.
func TestValidate_EnumParam_MissingRequiredNotInteractive_DoesNotCallFTypeDetails(t *testing.T) {
	called := false
	conf := &Config{Params: ParamsConfig{"path": ParamConfig{
		Type:     ParamTypeEnum,
		Required: true,
		FTypeDetails: func(ICommand) (any, error) {
			called = true
			return []string{"a", "b"}, nil
		},
	}}}
	c := newValidateCommand(conf, "", map[string]any{})

	if err := validate(c); err == nil {
		t.Fatal("expected an error for a missing required parameter")
	}
	if called {
		t.Fatal("FTypeDetails must not be called unless actually going interactive")
	}
}

// TestValidate_EnumParam_MissingRequiredInteractive_PromptsUsingFTypeDetails
// is the one case where FTypeDetails IS expected to run: missing,
// Required, and Interactive together.
func TestValidate_EnumParam_MissingRequiredInteractive_PromptsUsingFTypeDetails(t *testing.T) {
	conf := &Config{Params: ParamsConfig{"path": ParamConfig{
		Type:        ParamTypeEnum,
		Required:    true,
		Interactive: true,
		FTypeDetails: func(ICommand) (any, error) {
			return []string{"first", "second"}, nil
		},
	}}}
	c := newValidateCommand(conf, "", map[string]any{})

	withStdinInput(t, "2\n")
	if err := validate(c); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if got := c.Param("path"); got != "second" {
		t.Fatalf("path = %#v, want second", got)
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
