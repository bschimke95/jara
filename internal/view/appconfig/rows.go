package appconfig

import (
	"charm.land/bubbles/v2/table"

	"github.com/bschimke95/jara/internal/model"
)

func columns() []table.Column {
	return []table.Column{
		{Title: "KEY", Width: 25},
		{Title: "VALUE", Width: 20},
		{Title: "DEFAULT", Width: 20},
		{Title: "SOURCE", Width: 10},
		{Title: "TYPE", Width: 8},
	}
}

func rows(entries []model.ConfigEntry) []table.Row {
	result := make([]table.Row, 0, len(entries))
	for _, e := range entries {
		result = append(result, table.Row{
			e.Key,
			e.Value,
			e.Default,
			e.Source,
			e.Type,
		})
	}
	return result
}
