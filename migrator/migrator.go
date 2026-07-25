// Package migrator manages DB schema migrations and data seeds for
// lxgo/kernel applications - migrations are YAML files with `up`/`down` SQL,
// tracked in a dedicated table; see NewCommand for the ready-made console
// command wrapping Create/Show/Check/Up/Down/UpSeeds.
package migrator

import (
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const cTABLE_NAME = "_lxgo_migrator"

var template = `name: %s
type: query

up: | # TODO SQL to up migration

down: | # TODO SQL to down migration
`

// Config configures the package-level migrator state - see Init.
type Config struct {
	// DB is the database connection migrations/seeds run against.
	DB *sql.DB
	// MigrationsPath is the directory migration YAML files are read from/written to.
	MigrationsPath string
	// SeedsPath is the directory seed YAML files are read from.
	SeedsPath string
}

// Init sets up the package-level migrator state from conf - call this once
// before using any other function in the package.
func Init(conf Config) {
	m.db = conf.DB
	m.migrationsPath = conf.MigrationsPath
	m.seedsPath = conf.SeedsPath
}

// SetDB overrides the database connection set by Init.
func SetDB(db *sql.DB) {
	m.db = db
}

// SetMigrationsPath overrides the migrations directory set by Init.
func SetMigrationsPath(migrationsPath string) {
	m.migrationsPath = migrationsPath
}

// Create writes a new migration YAML file (timestamped, named after name)
// into the configured migrations directory, with an up/down template ready to fill in.
func Create(name string) error {
	timestamp := time.Now().UTC().Format("20060102150405.000")
	filename := fmt.Sprintf("%s_%s.yaml", timestamp, name)
	if m.migrationsPath != "" {
		migrationsPath := strings.TrimSuffix(m.migrationsPath, "/")
		filename = filepath.Join(migrationsPath, filename)
	}
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("can not create file: %w", err)
	}
	defer file.Close()

	content := fmt.Sprintf(template, name)
	_, err = file.WriteString(content)
	if err != nil {
		return fmt.Errorf("can not write to file: %w", err)
	}

	fmt.Printf("Migration '%s' has been created\n", filename)
	return nil
}

// Check returns every migration that hasn't been applied yet.
func Check() ([]*migration, error) {
	list, err := getMigrations(cGET_UNAPPLIED_ONLY)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// Show returns the last count migrations (applied or not), oldest first;
// count == 0 returns all of them.
func Show(count int) ([]*migration, error) {
	list, err := getMigrations(cGET_ALL)
	if err != nil {
		return nil, err
	}

	if count > len(list) || count == 0 {
		count = len(list)
	}
	return list[len(list)-count:], nil
}

// Up applies every unapplied migration, in one transaction, printing
// progress and errors to stdout.
func Up() {
	mm, err := Check()
	if err != nil {
		fmt.Printf("Migrations up failed. Cause: %s\n", err)
		return
	}

	if len(mm) == 0 {
		fmt.Println("All migrations are up to date.")
		return
	}

	err = m.createTable()
	if err != nil {
		fmt.Printf("Migrations up failed. Cause: %s\n", err)
		return
	}

	tx, err := m.db.Begin()
	if err != nil {
		fmt.Printf("Migrations up failed. Cause: %s\n", err)
		return
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	for _, mig := range mm {
		err = upMigration(tx, mig)
		if err != nil {
			fmt.Printf("Migrations up failed. Cause: %s\n", err)
			return
		}
	}

	fmt.Println("All migrations applied successfully.")
}

// Down rolls back the last steps applied migrations, in one transaction,
// printing progress and errors to stdout; steps == 0 rolls back just the
// last one.
func Down(steps int) {
	appliedMigrations, err := getMigrations(cGET_APPLIED_ONLY)
	if err != nil {
		fmt.Printf("Migrations down failed. Cause: %s\n", err)
		return
	}

	if len(appliedMigrations) == 0 {
		fmt.Println("No migrations to roll back.")
		return
	}

	if steps > len(appliedMigrations) {
		steps = len(appliedMigrations)
	} else if steps == 0 {
		steps = 1
	}

	tx, err := m.db.Begin()
	if err != nil {
		fmt.Printf("Migrations down failed. Cause: %s\n", err)
		return
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	count := len(appliedMigrations)
	for i := 1; i <= steps; i++ {
		mig := appliedMigrations[count-i]

		err = downMigration(tx, mig)
		if err != nil {
			fmt.Printf("Migrations down failed. Cause: %s\n", err)
			return
		}
	}

	fmt.Println("Selected migrations rolled back successfully.")
}

// UpSeeds applies every seed YAML file in the configured seeds directory -
// each file's basename is the target table, each entry an inserted row -
// printing progress and errors to stdout.
func UpSeeds() {
	if m.seedsPath == "" {
		fmt.Println("Seeds path not set.")
		return
	}

	files, err := os.ReadDir(m.seedsPath)
	if err != nil {
		fmt.Printf("Seeds failed. Cause: %s\n", err)
		return
	}

	tx, err := m.db.Begin()
	if err != nil {
		fmt.Printf("Seeds failed. Cause: %s\n", err)
		return
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if !strings.HasSuffix(file.Name(), ".yaml") {
			continue
		}

		err = applySeed(tx, file)
		if err != nil {
			fmt.Printf("Seed failed. Cause: %s\n", err)
			return
		}
	}

	fmt.Println("Seeds applied successfully.")
}

func upMigration(tx *sql.Tx, mig *migration) error {
	content, err := os.ReadFile(filepath.Join(m.migrationsPath, mig.file))
	if err != nil {
		return fmt.Errorf("failed to read migration file '%s': %s", mig.file, err)
	}

	var data struct {
		Up any `yaml:"up"`
	}
	err = yaml.Unmarshal(content, &data)
	if err != nil {
		return fmt.Errorf("failed to parse migration file '%s': %s", mig.file, err)
	}

	var upCommands []string
	switch v := data.Up.(type) {
	case string:
		upCommands = append(upCommands, v)
	case []any:
		for _, cmd := range v {
			cmdStr, ok := cmd.(string)
			if !ok {
				return fmt.Errorf("invalid command type in 'up' section of '%s'", mig.file)
			}
			upCommands = append(upCommands, cmdStr)
		}
	default:
		return fmt.Errorf("'up' section of '%s' must be a string or an array of strings", mig.file)
	}

	for _, cmd := range upCommands {
		_, err = tx.Exec(cmd)
		if err != nil {
			return fmt.Errorf("failed to execute migration '%s': %s. The SQL: %q", mig.file, err, cmd)
		}
	}

	_, err = tx.Exec(`INSERT INTO `+cTABLE_NAME+` (time, name) VALUES ($1, $2)`, mig.timestamp, mig.name)
	if err != nil {
		return fmt.Errorf("failed to update migrations table for '%s': %s", mig.file, err)
	}

	fmt.Printf("Migration '%s' applied successfully.\n", mig.file)
	return nil
}

func downMigration(tx *sql.Tx, mig *migration) error {
	content, err := os.ReadFile(filepath.Join(m.migrationsPath, mig.file))
	if err != nil {
		return fmt.Errorf("failed to read migration file '%s': %s", mig.file, err)
	}

	var data struct {
		Down any `yaml:"down"`
	}
	err = yaml.Unmarshal(content, &data)
	if err != nil {
		return fmt.Errorf("failed to parse migration file '%s': %s", mig.file, err)
	}

	var downCommands []string
	switch v := data.Down.(type) {
	case string:
		downCommands = append(downCommands, v)
	case []any:
		for _, cmd := range v {
			cmdStr, ok := cmd.(string)
			if !ok {
				return fmt.Errorf("invalid command type in 'up' section of '%s'", mig.file)
			}
			downCommands = append(downCommands, cmdStr)
		}
	default:
		return fmt.Errorf("'up' section of '%s' must be a string or an array of strings", mig.file)
	}

	for _, cmd := range downCommands {
		_, err = tx.Exec(cmd)
		if err != nil {
			return fmt.Errorf("failed to execute migration '%s': %s. The SQL: %q", mig.file, err, cmd)
		}
	}

	_, err = tx.Exec(`DELETE FROM `+cTABLE_NAME+` WHERE time = $1 AND name = $2`, mig.timestamp, mig.name)
	if err != nil {
		return fmt.Errorf("failed to update migrations table for '%s': %s", mig.file, err)
	}

	fmt.Printf("Migration '%s' rolled back successfully.\n", mig.file)
	return nil
}

func applySeed(tx *sql.Tx, file fs.DirEntry) error {
	table := strings.TrimSuffix(file.Name(), ".yaml")

	path := filepath.Join(m.seedsPath, file.Name())

	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read seed file '%s': %s", file.Name(), err)
	}

	var rows []map[string]any

	err = yaml.Unmarshal(content, &rows)
	if err != nil {
		return fmt.Errorf("failed to parse seed file '%s': %s", file.Name(), err)
	}

	for _, row := range rows {
		err = insertRow(tx, table, row)
		if err != nil {
			return err
		}
	}

	fmt.Printf("Seed '%s' applied.\n", file.Name())

	return nil
}

func insertRow(tx *sql.Tx, table string, row map[string]any) error {
	cols := []string{}
	placeholders := []string{}
	values := []any{}

	i := 1
	for k, v := range row {
		cols = append(cols, k)
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		values = append(values, v)
		i++
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		table,
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
	)

	_, err := tx.Exec(query, values...)
	if err != nil {
		return fmt.Errorf("seed insert failed for table '%s': %s", table, err)
	}

	return nil
}
