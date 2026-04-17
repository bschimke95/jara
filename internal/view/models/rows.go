package models

import (
	"charm.land/bubbles/v2/table"

	"github.com/bschimke95/jara/internal/model"
)

func columns() []table.Column {
	return []table.Column{
		{Title: "NAME", Width: 30},
		{Title: "OWNER", Width: 16},
		{Title: "TYPE", Width: 8},
		{Title: "STATUS", Width: 12},
		{Title: "UUID", Width: 36},
	}
}

func modelRows(mdls []model.ModelSummary) []table.Row {
	rows := make([]table.Row, 0, len(mdls))
	for _, m := range mdls {
		name := m.ShortName
		if m.Current {
			name += " *"
		}
		rows = append(rows, table.Row{name, m.Owner, m.Type, m.Status, m.UUID})
	}
	return rows
}
