package migrator

type appliedDataItem struct {
	time string
	name string
}

type appliedData struct {
	items []*appliedDataItem
}

func newAppliedData(items []*appliedDataItem) *appliedData {
	return &appliedData{items: items}
}

func (d *appliedData) checkMigration(m *migration) bool {
	for _, item := range d.items {
		if item.time == m.getTimestamp() && item.name == m.getName() {
			return true
		}
	}

	return false
}
