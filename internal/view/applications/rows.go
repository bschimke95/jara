package applications

import (
	"fmt"

	"charm.land/bubbles/v2/table"

	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/ui"
)

func columns() []table.Column {
	return []table.Column{
		{Title: "NAME", Width: 20},
		{Title: "STATUS", Width: 14},
		{Title: "CHARM", Width: 22},
		{Title: "CHANNEL", Width: 16},
		{Title: "REV", Width: 5},
		{Title: "SCALE", Width: 6},
		{Title: "EXPOSED", Width: 8},
		{Title: "MESSAGE", Width: 30},
	}
}

func rows(apps map[string]model.Application) []table.Row {
	names := ui.SortedKeys(apps)
	result := make([]table.Row, 0, len(names))
	for _, name := range names {
		app := apps[name]
		exposed := "no"
		if app.Exposed {
			exposed = "yes"
		}
		result = append(result, table.Row{
			app.Name,
			app.Status,
			app.Charm,
			app.CharmChannel,
			fmt.Sprintf("%d", app.CharmRev),
			fmt.Sprintf("%d", app.Scale),
			exposed,
			app.StatusMessage,
		})
	}
	return result
}
