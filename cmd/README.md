# Package for console commands creating in lxgo/kernel applications

> Actual version: `v0.1.0-alpha.6`. [Details](https://github.com/epicoon/lxgo/tree/master/cmd/CHANGE_LOG.md)

> You can use it if your application is based on [lxgo/kernel](https://github.com/epicoon/lxgo/tree/master/kernel)

## Prepare your application

To split console calls you need to do next steps:

1. Create main command and move `func main()` code from `main.go`:
```go
package main

import (
	"github.com/epicoon/lxgo/cmd"
)

type MainCommand struct {
	*cmd.Command
}

func NewMainCommand(_ ...cmd.ICommandOptions) cmd.ICommand {
	return &MainCommand{Command: cmd.NewCommand()}
}

func (c *MainCommand) Exec() error {
	// Init and run your app - code from "main.go"

	return nil
}
```

2. Init commands executor in your `main.go` file:
```go
package main

import (
	"github.com/epicoon/lxgo/cmd"
)

func main() {
	cmd.Init(cmd.CommandsList{
		// Empty command key corresponds to `go run .` call
		"": NewMainCommand,
	})
	cmd.Run()
}
```

3. After that `go run .` will do the same as before


## Create your console command

1. Write command code:
```go
package main

import (
	"fmt"

	"github.com/epicoon/lxgo/cmd"
)

type MyCommand struct {
	*cmd.Command
}

func NewMyCommand(_ ...cmd.ICommandOptions) cmd.ICommand {
	return cmd.Prepare(&MyCommand{Command: cmd.NewCommand()})
}

func (c *MyCommand) Config() *cmd.Config {
	return &cmd.Config{
		Description: "My command to say hi and bye :)",
		Actions: cmd.ActionsConfig{
			"hi": cmd.ActionConfig{
				Description: "Say hi!",
				Executor:    actionHi,
				Params: cmd.ParamsConfig{
					"name": cmd.ParamConfig{
						Description: "The name of the one we greet",
						Type:        cmd.ParamTypeString,
						Required:    false,
						Default:     "anonymous",
					},
				},
			},
			"bye": cmd.ActionConfig{
				Description: "Say bye",
				Executor:    actionBye,
			},
		},
	}
}

// Corresponds to `go run . {command_key}:hi --name="Al"` call
func actionHi(c cmd.ICommand) error {
	name := c.Param("name").(string)
	fmt.Println("Hi, "+name+"!")
	return nil
}

// Corresponds to `go run . {command_key}:bye` call
func actionBye(c cmd.ICommand) error {
	fmt.Println("Bye")
	return nil
}
```

2. Add your command to `CommandsList`:
```go
package main

import (
	"github.com/epicoon/lxgo/cmd"
)

func main() {
	cmd.Init(cmd.CommandsList{
		"": NewMainCommand,
		"my-command": NewMyCommand,
	})
	cmd.Run()
}
```

3. After that you can call:
    - `go run . my-command:hi`
    - `go run . my-command:hi --name="Al"`
    - `go run . my-command:bye`
    - `go run . my-command --help`
    - `go run . my-command:hi --help`

## Example for alternative way to write command code without `Config`
> The example is given for better understanding. The best practice is to write code using `Config`
```go
package main

import (
	"errors"
	"fmt"

	"github.com/epicoon/lxgo/cmd"
)

type MyCommand struct {
	*cmd.Command
}

func NewMyCommand(_ ...cmd.ICommandOptions) cmd.ICommand {
	c := &MyCommand{Command: cmd.NewCommand()}
	c.RegisterActions(cmd.ActionsList{
		"hi": actionHi,
		"bye": actionBye,
	})
	return c
}

// Corresponds to `go run . {command_key}:hi --name="Al"` call
func actionHi(c cmd.ICommand) error {
	name := "anonymous"
	if c.HasParam("name") {
		param, ok := c.Param("name").(string)
		if !ok {
			return errors.New("parameter 'name' must be string")
		}
		name = param
	}
	fmt.Println("Hi, "+name+"!")
	return nil
}

// Corresponds to `go run . {command_key}:bye` call
func actionBye(c cmd.ICommand) error {
	fmt.Println("Bye")
	return nil
}
```


## `cmd.ICommandOptions`

Every command constructor has the same signature —
`func(opt ...cmd.ICommandOptions) cmd.ICommand` — so that any of them can be
plugged into `cmd.CommandsList` uniformly. But different commands often need
different construction-time data (e.g. a reference to the app instance), and
Go doesn't let `CommandsList`'s constructor type vary per entry. `ICommandOptions`
is how this package works around that: it's just `interface{}` — any struct
automatically satisfies it — used purely as a generic "some options, typed
by whoever's constructing this specific command" placeholder.

To use it:
1. Define your own options struct — no need to implement anything, any type
   satisfies `ICommandOptions`.
2. In your command's constructor, pull it out with the generic helper
   `cmd.GetOptions[YourOptions](opt)`, which type-asserts the first passed
   option to `YourOptions` (returning the zero value if none was passed or
   it's the wrong type).

A real example — `lxgo-jspp`'s compile command needs the app instance to do
anything useful:
```go
// lxgo-jspp/cmd/compile_command.go
type CompileCommandOptions struct {
	App kernel.IApp
}

func NewCompileCommand(opt ...cmd.ICommandOptions) cmd.ICommand {
	options := cmd.GetOptions[CompileCommandOptions](opt)
	if options.App == nil {
		panic("CompileCommand option 'App' is not defined")
	}
	// ...
}
```
and the caller passes the options explicitly when wrapping it into their own
constructor (see [Start using `lxgo-jspp`](https://github.com/epicoon/lxgo/tree/master/jspp/doc/start.md)
for the full context):
```go
func NewJSPPCommand(_ ...cmd.ICommandOptions) cmd.ICommand {
	// Create your app ...
	return jsppCmd.NewCompileCommand(jsppCmd.CompileCommandOptions{
		App: app,
	})
}
```
A command that doesn't need any construction-time data (like every example
above in this README) simply ignores the parameter (`_ ...cmd.ICommandOptions`).


## License

Apache License 2.0 — see [LICENSE](./LICENSE).
