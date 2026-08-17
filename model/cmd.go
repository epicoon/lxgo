package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/epicoon/lxgo/cmd"
	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/migrator"
)

// CommandOptions is NewCommand's cmd.ICommandOptions - App is required.
type CommandOptions struct {
	// App is the application ModelManager is registered on.
	App kernel.IApp
}

/** @interface cmd.ICommand */

// Command is the model:<action> console command - status/migrate against
// the ModelManager registered on App (see AppComponent) - see NewCommand.
type Command struct {
	*cmd.Command
	app kernel.IApp
}

var _ cmd.ICommand = (*Command)(nil)

/** @constructor cmd.CCommand */

// NewCommand constructs a Command - panics if CommandOptions.App isn't given.
func NewCommand(opt ...cmd.ICommandOptions) cmd.ICommand {
	options := cmd.GetOptions[CommandOptions](opt)
	if options.App == nil {
		panic("model.Command option 'App' is not defined")
	}

	return cmd.Prepare(&Command{
		Command: cmd.NewCommand(),
		app:     options.App,
	})
}

// Config declares the "db-status"/"db-migrate"/"db-audit"/"codegen-status"/
// "codegen-generate"/"codegen-repos"/"actualize" actions - see cmd.ICommand.
func (c *Command) Config() *cmd.Config {
	return &cmd.Config{
		Description: "Command to manage model schemas, migrations, and generated Go model/repository code",
		Actions: cmd.ActionsConfig{
			"db-status": cmd.ActionConfig{
				Description: "Show the diff between schema files and the database",
				Executor:    dbStatus,
			},
			"db-migrate": cmd.ActionConfig{
				Description: "Generate a migration from the current diff",
				Executor:    dbMigrate,
				Params: cmd.ParamsConfig{
					"name": cmd.ParamConfig{
						Description: "New migration name",
						Type:        cmd.ParamTypeString,
						Required:    true,
					},
					"apply": cmd.ParamConfig{
						Description: "Apply the generated migration immediately",
						Type:        cmd.ParamTypeBool,
					},
				},
			},
			"db-audit": cmd.ActionConfig{
				Description: "Check the service tables for stale records no longer matching the database",
				Executor:    dbAudit,
			},
			"codegen-status": cmd.ActionConfig{
				Description: "Show which generated model files (and scaffolded repository files) are missing or stale",
				Executor:    codegenStatus,
			},
			"codegen-generate": cmd.ActionConfig{
				Description: "Generate (or regenerate) Go model files for every Target with Models configured",
				Executor:    codegenGenerate,
			},
			"codegen-repos": cmd.ActionConfig{
				Description: "Scaffold a Go repository file for every Target with Repos configured, skipping any that already exist",
				Executor:    codegenRepos,
			},
			"actualize": cmd.ActionConfig{
				Description: "Check the schema diff and generated code status, show a plan, and on confirmation regenerate models, generate and apply a migration, and scaffold repositories",
				Executor:    actualize,
				Params: cmd.ParamsConfig{
					"name": cmd.ParamConfig{
						Description: "New migration name (prompted for interactively if a migration is needed and this isn't passed)",
						Type:        cmd.ParamTypeString,
					},
					"yes": cmd.ParamConfig{
						Description: "Skip the confirmation dialog and apply the plan immediately",
						Type:        cmd.ParamTypeBool,
					},
				},
			},
		},
	}
}

/** @handler cmd.FAction */
func dbStatus(com cmd.ICommand) error {
	c := com.(*Command)
	mm, err := AppComponent(c.app)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return nil
	}

	schemas, err := mm.LoadModelSchemas()
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return nil
	}

	diffs, err := CompareSchemas(mm.DB(), schemas)
	if err != nil {
		if errors.Is(err, ErrUnappliedMigrations) {
			fmt.Println("There are unapplied migrations - apply them first (see migrator:up)")
			return nil
		}
		fmt.Printf("Error: %s\n", err)
		return nil
	}

	if len(diffs) == 0 {
		fmt.Println("Schema files and the database are in sync")
		return nil
	}

	fmt.Println("Diff:")
	for _, d := range diffs {
		printModelDiff(d)
	}
	return nil
}

func printModelDiff(d ModelDiff) {
	if d.NeedsTable {
		fmt.Printf("- %s: needs table\n", d.Name)
		return
	}

	fmt.Printf("- %s:\n", d.Name)
	for _, name := range d.Fields.Added {
		fmt.Printf("    + %s\n", name)
	}
	for _, name := range d.Fields.Deleted {
		fmt.Printf("    - %s\n", name)
	}
	for _, name := range d.Fields.Changed {
		fmt.Printf("    ~ %s\n", name)
	}
	for _, r := range d.Fields.Renamed {
		how := "heuristic"
		if r.Explicit {
			how = "explicit"
		}
		fmt.Printf("    -> %s to %s (%s)\n", r.From, r.To, how)
	}
	for _, name := range d.AddTimestamps {
		fmt.Printf("    + %s\n", name)
	}
}

/** @handler cmd.FAction */
func dbMigrate(com cmd.ICommand) error {
	c := com.(*Command)

	name, ok := c.Params()["name"].(string)
	if !ok || name == "" {
		fmt.Println("Please, enter the --name parameter for the new migration")
		return nil
	}

	mm, err := AppComponent(c.app)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return nil
	}

	schemas, err := mm.LoadModelSchemas()
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return nil
	}

	actions, err := GenerateMigration(mm.DB(), schemas, name)
	if err != nil {
		if errors.Is(err, ErrUnappliedMigrations) {
			fmt.Println("There are unapplied migrations - apply them first (see migrator:up)")
			return nil
		}
		fmt.Printf("Error: %s\n", err)
		return nil
	}
	if len(actions) == 0 {
		fmt.Println("Nothing to migrate")
		return nil
	}

	fmt.Printf("Migration '%s' created with %d action(s)\n", name, len(actions))

	if c.Flag("apply") {
		migrator.Up()
	}

	return nil
}

/** @handler cmd.FAction */
func dbAudit(com cmd.ICommand) error {
	c := com.(*Command)
	mm, err := AppComponent(c.app)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return nil
	}

	relOrphans, err := AuditRelationFks(mm.DB())
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return nil
	}
	typeOrphans, err := AuditColumnTypes(mm.DB())
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return nil
	}

	if len(relOrphans) == 0 && len(typeOrphans) == 0 {
		fmt.Println("Service tables are clean - no stale records found")
		return nil
	}

	if len(relOrphans) > 0 {
		fmt.Printf("Stale relation records (%s):\n", systemRelationsTable)
		for _, o := range relOrphans {
			fmt.Printf("- %s: %s.%s -> %s.%s (%s), no matching foreign key found\n",
				o.FkName, o.HomeModel, o.HomeAttribute, o.RelatedModel, o.RelatedAttribute, o.Type)
		}
	}
	if len(typeOrphans) > 0 {
		fmt.Printf("Stale column type records (%s):\n", systemTypesTable)
		for _, o := range typeOrphans {
			fmt.Printf("- %s.%s: declared as %s, column no longer exists\n", o.TableName, o.ColumnName, o.Type)
		}
	}
	return nil
}

// actualize is "model:actualize" - checks the schema diff and generated
// code status, prints a plan, and on confirmation applies it: regenerate
// models, generate and apply a migration (if the diff is non-empty), then
// scaffold missing repositories. An already-pending set of unapplied
// migrations (ErrUnappliedMigrations) is applied first, before the diff
// used for the plan is computed, so the plan always reflects the database
// state actualize itself is about to leave things in.
/** @handler cmd.FAction */
func actualize(com cmd.ICommand) error {
	c := com.(*Command)
	mm, err := AppComponent(c.app)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return nil
	}

	schemas, err := mm.LoadModelSchemas()
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return nil
	}

	diffs, err := CompareSchemas(mm.DB(), schemas)
	if errors.Is(err, ErrUnappliedMigrations) {
		fmt.Println("There are already pending migrations - applying them first")
		migrator.Up()
		diffs, err = CompareSchemas(mm.DB(), schemas)
	}
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return nil
	}

	staleModels, missingRepos := mm.codegenPlan(schemas)
	if len(diffs) == 0 && len(staleModels) == 0 && len(missingRepos) == 0 {
		fmt.Println("Nothing to actualize")
		return nil
	}

	fmt.Println("Plan:")
	if len(diffs) == 0 {
		fmt.Println("Schema files and the database are in sync")
	} else {
		fmt.Println("Diff:")
		for _, d := range diffs {
			printModelDiff(d)
		}
	}
	if len(staleModels) > 0 {
		fmt.Printf("Models to (re)generate: %s\n", strings.Join(staleModels, ", "))
	}
	if len(missingRepos) > 0 {
		fmt.Printf("Repositories to scaffold: %s\n", strings.Join(missingRepos, ", "))
	}

	if !c.Flag("yes") {
		idx, err := cmd.PromptSelect("Apply?", []string{"Apply", "Cancel"})
		if err != nil || idx != 0 {
			fmt.Println("Aborted")
			return nil
		}
	}

	if err := codegenGenerate(c); err != nil {
		return err
	}

	if len(diffs) > 0 {
		name, _ := c.Params()["name"].(string)
		if name == "" {
			if c.Flag("yes") {
				fmt.Println("Please, pass --name for the new migration when using --yes")
				return nil
			}
			name, err = cmd.PromptString("Migration name")
			if err != nil {
				fmt.Printf("Error: %s\n", err)
				return nil
			}
		}
		if name == "" {
			fmt.Println("Please, enter a migration name")
			return nil
		}

		actions, err := GenerateMigration(mm.DB(), schemas, name)
		if err != nil {
			fmt.Printf("Error: %s\n", err)
			return nil
		}
		fmt.Printf("Migration '%s' created with %d action(s)\n", name, len(actions))
		migrator.Up()
	}

	return codegenRepos(c)
}
