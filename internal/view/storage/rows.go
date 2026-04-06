package storage

import (
	"charm.land/bubbles/v2/table"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
)

// Columns returns the storage table column definitions.
func Columns() []table.Column {
	return []table.Column{
		{Title: "ID", Width: 16},
		{Title: "Kind", Width: 12},
		{Title: "Owner", Width: 20},
		{Title: "Status", Width: 12},
		{Title: "Persistent", Width: 10},
		{Title: "Pool/Location", Width: 30},
	}
}

// Rows converts storage instances into table rows.
func Rows(instances []model.StorageInstance, styles *color.Styles) []table.Row {
	rows := make([]table.Row, 0, len(instances))
	for _, si := range instances {
		persistent := "no"
		if si.Persistent {
			persistent = "yes"
		}
		status := styles.StatusText(si.Status)
		rows = append(rows, table.Row{
			si.ID,
			si.Kind,
			si.Owner,
			status,
			persistent,
			si.Pool,
		})
	}
	return rows
}
