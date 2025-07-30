package migrator

import (
	"database/sql"
	"fmt"
)

type manager struct {
	db   *sql.DB
	path string
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

	rows, err := m.db.Query("SELECT time, name FROM " + cTABLE_NAME)
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
		WHERE table_name = '%s'`, cTABLE_NAME)
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

	query := fmt.Sprintf(`
		CREATE TABLE %s (
			time VARCHAR(18) NOT NULL,
			name VARCHAR(255) NOT NULL,
			PRIMARY KEY (time, name));`, cTABLE_NAME)
	_, err = m.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create table %s: %w", cTABLE_NAME, err)
	}

	return nil
}
