package modelview

import (
	"fmt"

	"charm.land/bubbles/v2/table"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/ui"
)

func applicationColumns() []table.Column {
	return []table.Column{
		{Title: "NAME", Width: 20},
		{Title: "STATUS", Width: 14},
		{Title: "SCALE", Width: 6},
		{Title: "MESSAGE", Width: 30},
	}
}

func applicationRows(apps map[string]model.Application, s *color.Styles) []table.Row {
	names := ui.SortedKeys(apps)
	result := make([]table.Row, 0, len(names))
	for _, name := range names {
		app := apps[name]
		result = append(result, table.Row{
			app.Name,
			s.StatusText(app.Status),
			fmt.Sprintf("%d", app.Scale),
			app.StatusMessage,
		})
	}
	return result
}
