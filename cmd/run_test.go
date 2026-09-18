package cmd

import (
	"errors"
	"testing"
)

// runWith runs the command line args against list with a fresh manager,
// returning the exit code run reports.
func runWith(t *testing.T, list CommandsList, args ...string) int {
	t.Helper()
	old := m
	m = &manager{list: list}
	t.Cleanup(func() { m = old })
	return run(args)
}

// actionsCommand is a command declaring one or more actions.
type actionsCommand struct {
	*Command
	conf *Config
}

func (c *actionsCommand) Config() *Config { return c.conf }

func actionsCmd(actions ActionsConfig, params ParamsConfig) CCommand {
	return func(...ICommandOptions) ICommand {
		return Prepare(&actionsCommand{
			Command: NewCommand(),
			conf:    &Config{Actions: actions, Params: params},
		})
	}
}

type execCommand struct {
	*Command
	execErr   error
	beforeErr error
}

func (c *execCommand) Exec() error       { return c.execErr }
func (c *execCommand) BeforeExec() error { return c.beforeErr }

func execCmd(execErr, beforeErr error) CCommand {
	return func(...ICommandOptions) ICommand {
		return &execCommand{Command: NewCommand(), execErr: execErr, beforeErr: beforeErr}
	}
}

func TestRun_ExitCode(t *testing.T) {
	boom := errors.New("boom")
	ok := func(ICommand) error { return nil }
	fail := func(ICommand) error { return boom }

	tests := []struct {
		name string
		list CommandsList
		args []string
		want int
	}{
		{"unknown command", CommandsList{}, []string{"nope"}, 1},
		{"no default command", CommandsList{}, nil, 1},
		{"action succeeds", CommandsList{"c": actionsCmd(ActionsConfig{"go": {Executor: ok}}, nil)}, []string{"c:go"}, 0},
		{"action fails", CommandsList{"c": actionsCmd(ActionsConfig{"go": {Executor: fail}}, nil)}, []string{"c:go"}, 1},
		{"undefined action", CommandsList{"c": actionsCmd(ActionsConfig{"go": {Executor: ok}}, nil)}, []string{"c:missing"}, 1},
		{"no action just lists them", CommandsList{"c": actionsCmd(ActionsConfig{"go": {Executor: ok}}, nil)}, []string{"c"}, 0},
		{"help", CommandsList{"c": actionsCmd(ActionsConfig{"go": {Executor: fail}}, nil)}, []string{"c:go", "--help"}, 0},
		{"exec succeeds", CommandsList{"": execCmd(nil, nil)}, nil, 0},
		{"exec fails", CommandsList{"": execCmd(boom, nil)}, nil, 1},
		{"before-exec fails", CommandsList{"": execCmd(nil, boom)}, nil, 1},
		{
			"required parameter missing",
			CommandsList{"c": actionsCmd(
				ActionsConfig{"go": {Executor: ok, Params: ParamsConfig{"name": {Type: ParamTypeString, Required: true}}}}, nil)},
			[]string{"c:go"},
			1,
		},
		{
			"required parameter given",
			CommandsList{"c": actionsCmd(
				ActionsConfig{"go": {Executor: ok, Params: ParamsConfig{"name": {Type: ParamTypeString, Required: true}}}}, nil)},
			[]string{"c:go", "--name=x"},
			0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := runWith(t, tc.list, tc.args...); got != tc.want {
				t.Fatalf("run(%v) = %d, want %d", tc.args, got, tc.want)
			}
		})
	}
}
