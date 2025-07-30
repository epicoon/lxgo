package migrator

import (
	"database/sql"
	"fmt"
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

func Init(db *sql.DB, migrationsPath string) {
	m.db = db
	m.path = migrationsPath
}

func SetDB(db *sql.DB) {
	m.db = db
}

func SetMigrationsPath(migrationsPath string) {
	m.path = migrationsPath
}

func Create(name string) error {
	timestamp := time.Now().UTC().Format("20060102150405.000")
	filename := fmt.Sprintf("%s_%s.yaml", timestamp, name)
	if m.path != "" {
		migrationsPath := strings.TrimSuffix(m.path, "/")
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

func Check() ([]*migration, error) {
	list, err := getMigrations(cGET_UNAPPLIED_ONLY)
	if err != nil {
		return nil, err
	}

	return list, nil
}

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

func Up() {
	mm, err := Check()
	if err != nil {
		fmt.Printf("Migrations up failed. Cause: %s\n", err)
	}

	if len(mm) == 0 {
		fmt.Println("All migrations are up to date.")
		return
	}

	err = m.createTable()
	if err != nil {
		fmt.Printf("Migrations up failed. Cause: %s\n", err)
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

func upMigration(tx *sql.Tx, mig *migration) error {
	content, err := os.ReadFile(filepath.Join(m.path, mig.file))
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
	content, err := os.ReadFile(filepath.Join(m.path, mig.file))
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
