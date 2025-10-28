# Package for console commands creating in lxgo/kernel applications

> Actual version: `v0.1.0-alpha.4`. [Details](https://github.com/epicoon/lxgo/tree/master/cmd/CHANGE_LOG.md)

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
		Description: "My command to say hi and by :)",
		Actions: cmd.ActionsConfig{
			"hi": cmd.ActionConfig{
				Description: "Say hi!",
				Executor:    actionHi,
				Params: cmd.ParamsConfig{
					"name": cmd.ParamConfig{
						Description: "The name of the one we greet",
						Type:        cmd.ParamTypeString,
						Required:    false,
						Default:     "anonimus",
					},
				},
			},
			"by": cmd.ActionConfig{
				Description: "Say by",
				Executor:    actionBy,
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

// Corresponds to `go run . {command_key}:by` call
func actionBy(c cmd.ICommand) error {
	fmt.Println("By")
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
    - `go run . my-command:by`
    - `go run . my-command --help`
    - `go run . my-command:hi --help`

## Example for alternative way to write command code without `Config`
> The example is given for better understanding. The best practice is to write code using `Config`
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
	c := &MyCommand{Command: cmd.NewCommand()}
	c.RegisterActions(cmd.ActionsList{
		"hi": actionHi,
		"by": actionBy,
	})
	return c
}

// Corresponds to `go run . {command_key}:hi --name="Al"` call
func actionHi(c cmd.ICommand) error {
	name := "anonimus"
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

// Corresponds to `go run . {command_key}:by` call
func actionBy(c cmd.ICommand) error {
	fmt.Println("By")
	return nil
}
```


//TODO:
    - Explain cmd.ICommandOptions
