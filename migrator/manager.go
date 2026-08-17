package migrator

import (
	"database/sql"
	"fmt"
)

type manager struct {
	db             *sql.DB
	migrationsPath string
	seedsPath      string
}

var m = new(manager)

func (m *manager) getAppliedData() (*appliedData, error) {
	exists, err := m.isTableExist()
	if err != nil {
		return nil, err
	}
	if !exists {
		return newAppliedData([]*appliedDataItem{}), nil
	}

	rows, err := m.db.Query("SELECT time, name FROM " + cQUALIFIED_TABLE_NAME)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch applied migrations: %s", err)
	}
	defer rows.Close()

	var items []*appliedDataItem
	for rows.Next() {
		var time, name string
		if err := rows.Scan(&time, &name); err != nil {
			return nil, fmt.Errorf("failed to scan row: %s", err)
		}
		items = append(items, &appliedDataItem{time: time, name: name})
	}

	return newAppliedData(items), nil
}

func (m *manager) isTableExist() (bool, error) {
	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM information_schema.tables
		WHERE table_schema = '%s' AND table_name = '%s'`, cSCHEMA_NAME, cTABLE_NAME)
	var count int
	err := m.db.QueryRow(query).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if table exists: %s", err)
	}

	return count > 0, nil
}

func (m *manager) createTable() error {
	exists, err := m.isTableExist()
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	if _, err := m.db.Exec("CREATE SCHEMA IF NOT EXISTS " + cSCHEMA_NAME); err != nil {
		return fmt.Errorf("failed to create schema %s: %w", cSCHEMA_NAME, err)
	}

	query := fmt.Sprintf(`
		CREATE TABLE %s (
			time VARCHAR(18) NOT NULL,
			name VARCHAR(255) NOT NULL,
			PRIMARY KEY (time, name));`, cQUALIFIED_TABLE_NAME)
	_, err = m.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create table %s: %w", cQUALIFIED_TABLE_NAME, err)
	}

	return nil
}
