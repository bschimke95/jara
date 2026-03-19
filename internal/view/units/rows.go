package units

import (
	"fmt"

	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/ui"
)

// CompactColumns defines the compact columns used inside the model overview panel.
func CompactColumns() []table.Column {
	return []table.Column{
		{Title: "UNIT", Width: 24},
		{Title: "WORKLOAD", Width: 12},
		{Title: "AGENT", Width: 12},
		{Title: "MESSAGE", Width: 28},
	}
}

// DetailColumns defines the full columns used by the standalone units view.
func DetailColumns() []table.Column {
	return []table.Column{
		{Title: "UNIT", Width: 24},
		{Title: "WORKLOAD", Width: 12},
		{Title: "AGENT", Width: 12},
		{Title: "MACHINE", Width: 8},
		{Title: "ADDRESS", Width: 16},
		{Title: "PORTS", Width: 16},
		{Title: "MESSAGE", Width: 30},
	}
}

// UnitToCompactRow builds the compact 4-column row used in the model overview panel.
func UnitToCompactRow(u model.Unit) table.Row {
	var name string
	if u.Leader {
		name = lipgloss.NewStyle().Foreground(color.HintKey).Render("★") + " " + u.Name
	} else {
		name = "  " + u.Name
	}
	workload := color.StatusText(u.WorkloadStatus)
	agent := color.StatusText(u.AgentStatus)
	return table.Row{name, workload, agent, u.WorkloadMessage}
}

// unitToDetailRow builds the full-column row used by the standalone units view.
func unitToDetailRow(u model.Unit) table.Row {
	var name string
	if u.Leader {
		name = lipgloss.NewStyle().Foreground(color.HintKey).Render("★") + " " + u.Name
	} else {
		name = "  " + u.Name
	}
	ports := ""
	if len(u.Ports) > 0 {
		for i, p := range u.Ports {
			if i > 0 {
				ports += ","
			}
			ports += p
		}
	}
	return table.Row{
		name,
		color.StatusText(u.WorkloadStatus),
		color.StatusText(u.AgentStatus),
		u.Machine,
		u.PublicAddress,
		ports,
		u.WorkloadMessage,
	}
}

// CompactRowsForApp returns compact unit rows for a specific application (model panel).
func CompactRowsForApp(app model.Application) []table.Row {
	var rows []table.Row
	for _, unit := range app.Units {
		rows = append(rows, UnitToCompactRow(unit))
		for _, sub := range unit.Subordinates {
			rows = append(rows, UnitToCompactRow(sub))
		}
	}
	return rows
}

// DetailRows returns full unit rows for all applications (standalone units view).
func DetailRows(apps map[string]model.Application) []table.Row {
	names := ui.SortedKeys(apps)
	var rows []table.Row
	for _, name := range names {
		app := apps[name]
		for _, unit := range app.Units {
			rows = append(rows, unitToDetailRow(unit))
			for _, sub := range unit.Subordinates {
				rows = append(rows, unitToDetailRow(sub))
			}
		}
	}
	return rows
}

// DetailRowsForApp returns full unit rows for one application (standalone units view).
func DetailRowsForApp(app model.Application) []table.Row {
	var rows []table.Row
	for _, unit := range app.Units {
		rows = append(rows, unitToDetailRow(unit))
		for _, sub := range unit.Subordinates {
			rows = append(rows, unitToDetailRow(sub))
		}
	}
	return rows
}

var pendingStyle = lipgloss.NewStyle().Foreground(color.Muted).Italic(true)

// PendingCompactRows returns compact placeholder rows for the model-overview units pane.
func PendingCompactRows(appName string, currentUnits []model.Unit, delta int) []table.Row {
	var rows []table.Row
	if delta > 0 {
		nextIdx := len(currentUnits)
		for range delta {
			name := pendingStyle.Render(fmt.Sprintf("  %s/%d", appName, nextIdx))
			rows = append(rows, table.Row{name, pendingStyle.Render("allocating"), pendingStyle.Render("allocating"), pendingStyle.Render("installing agent")})
			nextIdx++
		}
	} else if delta < 0 {
		n := -delta
		start := len(currentUnits) - n
		if start < 0 {
			start = 0
		}
		for _, u := range currentUnits[start:] {
			name := pendingStyle.Render("  " + u.Name + " (removing)")
			rows = append(rows, table.Row{name, pendingStyle.Render("terminating"), pendingStyle.Render("terminating"), ""})
		}
	}
	return rows
}

// PendingDetailRows returns full-column placeholder rows for the standalone units view.
func PendingDetailRows(appName string, currentUnits []model.Unit, delta int) []table.Row {
	var rows []table.Row
	if delta > 0 {
		nextIdx := len(currentUnits)
		for range delta {
			name := pendingStyle.Render(fmt.Sprintf("  %s/%d", appName, nextIdx))
			rows = append(rows, table.Row{name, pendingStyle.Render("allocating"), pendingStyle.Render("allocating"), "", "", "", pendingStyle.Render("waiting for unit…")})
			nextIdx++
		}
	} else if delta < 0 {
		n := -delta
		start := len(currentUnits) - n
		if start < 0 {
			start = 0
		}
		for _, u := range currentUnits[start:] {
			name := pendingStyle.Render("  " + u.Name + " (removing)")
			rows = append(rows, table.Row{name, pendingStyle.Render("terminating"), pendingStyle.Render("terminating"), u.Machine, u.PublicAddress, "", pendingStyle.Render("waiting for removal…")})
		}
	}
	return rows
}
