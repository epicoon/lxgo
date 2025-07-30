package migrator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	cGET_APPLIED_ONLY = iota
	cGET_UNAPPLIED_ONLY
	cGET_ALL
)

type migration struct {
	file      string
	name      string
	timestamp string
	extension string
	applied   bool
}

func NewMigration(filename string) (*migration, error) {
	base := filepath.Base(filename)
	ext := filepath.Ext(base)
	if ext == "" {
		return nil, fmt.Errorf("invalid filename '%s': no extension found", filename)
	}

	nameWithoutExt := strings.TrimSuffix(base, ext)
	parts := strings.SplitN(nameWithoutExt, "_", 2)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid filename '%s': expected format 'timestamp_name.ext'", filename)
	}

	return &migration{
		file:      filename,
		timestamp: parts[0],
		name:      parts[1],
		extension: ext,
	}, nil
}

func (m *migration) String() string {
	return m.file
}

func (m *migration) getTimestamp() string {
	return m.timestamp
}

func (m *migration) getName() string {
	return m.name
}

func (m *migration) setApplied(applied bool) {
	m.applied = applied
}

func (m *migration) isApplied() bool {
	return m.applied
}

func getMigrations(mode int) ([]*migration, error) {
	files, err := os.ReadDir(m.path)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", m.path, err)
	}

	var migrations []*migration
	appliedData, err := m.getAppliedData()
	if err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}

	for _, file := range files {
		if file.Type().IsRegular() {
			mig, err := NewMigration(file.Name())
			if err != nil {
				return nil, fmt.Errorf("error parsing filename '%s': %w", file.Name(), err)
			}
			mig.setApplied(appliedData.checkMigration(mig))
			switch mode {
			case cGET_ALL:
				migrations = append(migrations, mig)
			case cGET_APPLIED_ONLY:
				if mig.isApplied() {
					migrations = append(migrations, mig)
				}
			case cGET_UNAPPLIED_ONLY:
				if !mig.isApplied() {
					migrations = append(migrations, mig)
				}
			}
		}
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].timestamp < migrations[j].timestamp
	})

	return migrations, nil
}
