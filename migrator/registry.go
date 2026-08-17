package migrator

import (
	"database/sql"
	"fmt"

	"gopkg.in/yaml.v3"
)

type migrationTypeHandlers struct {
	apply  FMigrationApply
	invert FMigrationApply
}

var migrationTypes = map[string]migrationTypeHandlers{}

// RegisterMigrationType plugs a custom migration type into upMigration/
// downMigration, alongside the built-in "query" type (a plain SQL list
// under Up/Down) - migration files declaring Type: name are dispatched to
// apply (Up) or invert (Down) instead of being read as SQL. name must not
// already be registered.
func RegisterMigrationType(name string, apply, invert FMigrationApply) error {
	if _, exists := migrationTypes[name]; exists {
		return fmt.Errorf("migration type '%s' is already registered", name)
	}

	migrationTypes[name] = migrationTypeHandlers{apply: apply, invert: invert}
	return nil
}

// migrationAction is what a migration file resolves to for one direction
// (up or down) - either a list of SQL commands (the built-in "query" type)
// or a registered type's handler plus the untouched file content.
type migrationAction struct {
	commands []string
	handler  FMigrationApply
	raw      []byte
}

func (a migrationAction) run(tx *sql.Tx) error {
	if a.handler != nil {
		return a.handler(tx, a.raw)
	}

	for _, cmd := range a.commands {
		if _, err := tx.Exec(cmd); err != nil {
			return fmt.Errorf("%s. The SQL: %q", err, cmd)
		}
	}
	return nil
}

// resolveMigrationAction reads content's "Type" field and decides how to run
// it for the given phase ("Up" or "Down") - a pure function (no DB access),
// split out from upMigration/downMigration so the dispatch logic can be
// tested without a database connection.
func resolveMigrationAction(content []byte, phase, file string) (migrationAction, error) {
	var data map[string]any
	if err := yaml.Unmarshal(content, &data); err != nil {
		return migrationAction{}, fmt.Errorf("failed to parse migration file '%s': %s", file, err)
	}

	migType, _ := data["Type"].(string)
	if migType == "" || migType == "query" {
		commands, err := sqlCommands(data, phase, file)
		if err != nil {
			return migrationAction{}, err
		}
		return migrationAction{commands: commands}, nil
	}

	handlers, ok := migrationTypes[migType]
	if !ok {
		return migrationAction{}, fmt.Errorf("unknown migration type '%s' in '%s'", migType, file)
	}

	handler := handlers.apply
	if phase == "Down" {
		handler = handlers.invert
	}
	return migrationAction{handler: handler, raw: content}, nil
}

// sqlCommands reads section ("Up" or "Down") of a "query"-type migration's
// parsed content as a list of SQL statements - a single string or a list of
// strings.
func sqlCommands(data map[string]any, section, file string) ([]string, error) {
	var commands []string
	switch v := data[section].(type) {
	case string:
		commands = append(commands, v)
	case []any:
		for _, cmd := range v {
			cmdStr, ok := cmd.(string)
			if !ok {
				return nil, fmt.Errorf("invalid command type in '%s' section of '%s'", section, file)
			}
			commands = append(commands, cmdStr)
		}
	default:
		return nil, fmt.Errorf("'%s' section of '%s' must be a string or an array of strings", section, file)
	}
	return commands, nil
}
