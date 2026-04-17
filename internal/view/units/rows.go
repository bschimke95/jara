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
func UnitToCompactRow(u model.Unit, s *color.Styles) table.Row {
	var name string
	if u.Leader {
		name = lipgloss.NewStyle().Foreground(s.HintKeyColor).Render("★") + " " + u.Name
	} else {
		name = "  " + u.Name
	}
	workload := s.StatusText(u.WorkloadStatus)
	agent := s.StatusText(u.AgentStatus)
	return table.Row{name, workload, agent, u.WorkloadMessage}
}

// unitToDetailRow builds the full-column row used by the standalone units view.
func unitToDetailRow(u model.Unit, s *color.Styles) table.Row {
	var name string
	if u.Leader {
		name = lipgloss.NewStyle().Foreground(s.HintKeyColor).Render("★") + " " + u.Name
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
		s.StatusText(u.WorkloadStatus),
		s.StatusText(u.AgentStatus),
		u.Machine,
		u.PublicAddress,
		ports,
		u.WorkloadMessage,
	}
}

// CompactRowsForApp returns compact unit rows for a specific application (model panel).
func CompactRowsForApp(app model.Application, s *color.Styles) []table.Row {
	var rows []table.Row
	for _, unit := range app.Units {
		rows = append(rows, UnitToCompactRow(unit, s))
		for _, sub := range unit.Subordinates {
			rows = append(rows, UnitToCompactRow(sub, s))
		}
	}
	return rows
}

// DetailRows returns full unit rows for all applications (standalone units view).
func DetailRows(apps map[string]model.Application, s *color.Styles) []table.Row {
	names := ui.SortedKeys(apps)
	var rows []table.Row
	for _, name := range names {
		app := apps[name]
		for _, unit := range app.Units {
			rows = append(rows, unitToDetailRow(unit, s))
			for _, sub := range unit.Subordinates {
				rows = append(rows, unitToDetailRow(sub, s))
			}
		}
	}
	return rows
}

// DetailRowsForApp returns full unit rows for one application (standalone units view).
func DetailRowsForApp(app model.Application, s *color.Styles) []table.Row {
	var rows []table.Row
	for _, unit := range app.Units {
		rows = append(rows, unitToDetailRow(unit, s))
		for _, sub := range unit.Subordinates {
			rows = append(rows, unitToDetailRow(sub, s))
		}
	}
	return rows
}

// PendingCompactRows returns compact placeholder rows for the model-overview units pane.
// Only scale-up (delta > 0) generates pseudo rows; scale-down is handled by
// real status updates from Juju as units transition to dying/terminating.
func PendingCompactRows(appName string, currentUnits []model.Unit, delta int, s *color.Styles) []table.Row {
	if delta <= 0 {
		return nil
	}
	rows := make([]table.Row, 0, delta)
	nextIdx := len(currentUnits)
	for range delta {
		name := s.Pending.Render(fmt.Sprintf("  %s/%d", appName, nextIdx))
		rows = append(rows, table.Row{name, s.Pending.Render("allocating"), s.Pending.Render("allocating"), s.Pending.Render("installing agent")})
		nextIdx++
	}
	return rows
}

// PendingDetailRows returns full-column placeholder rows for the standalone units view.
// PendingDetailRows returns full-column placeholder rows for the standalone units view.
// Only scale-up (delta > 0) generates pseudo rows; scale-down is handled by
// real status updates from Juju as units transition to dying/terminating.
func PendingDetailRows(appName string, currentUnits []model.Unit, delta int, s *color.Styles) []table.Row {
	if delta <= 0 {
		return nil
	}
	rows := make([]table.Row, 0, delta)
	nextIdx := len(currentUnits)
	for range delta {
		name := s.Pending.Render(fmt.Sprintf("  %s/%d", appName, nextIdx))
		rows = append(rows, table.Row{name, s.Pending.Render("allocating"), s.Pending.Render("allocating"), "", "", "", s.Pending.Render("waiting for unit…")})
		nextIdx++
	}
	return rows
}
