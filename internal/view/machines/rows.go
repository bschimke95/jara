package machines

import (
	"sort"

	"charm.land/bubbles/v2/table"

	"github.com/bschimke95/jara/internal/model"
)

func columns() []table.Column {
	return []table.Column{
		{Title: "ID", Width: 6},
		{Title: "STATUS", Width: 12},
		{Title: "DNS NAME", Width: 30},
		{Title: "INSTANCE ID", Width: 14},
		{Title: "BASE", Width: 16},
		{Title: "HARDWARE", Width: 36},
	}
}

func machineRows(machines map[string]model.Machine) []table.Row {
	ids := make([]string, 0, len(machines))
	for id := range machines {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	var rows []table.Row
	for _, id := range ids {
		m := machines[id]
		rows = append(rows, table.Row{
			m.ID, m.Status, m.DNSName, m.InstanceID, m.Base, m.Hardware,
		})
		for _, c := range m.Containers {
			rows = append(rows, table.Row{
				c.ID, c.Status, c.DNSName, c.InstanceID, c.Base, c.Hardware,
			})
		}
	}
	return rows
}
