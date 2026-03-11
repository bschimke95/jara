package render

import (
	"fmt"
	"sort"

	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"
	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
)

// ModelColumns defines the columns for the models table.
func ModelColumns() []table.Column {
	return []table.Column{
		{Title: "NAME", Width: 30},
		{Title: "OWNER", Width: 16},
		{Title: "TYPE", Width: 8},
		{Title: "UUID", Width: 36},
	}
}

// ModelRows converts a slice of model summaries to table rows.
func ModelRows(models []model.ModelSummary) []table.Row {
	rows := make([]table.Row, 0, len(models))
	for _, m := range models {
		name := m.ShortName
		if m.Current {
			name += " *"
		}
		rows = append(rows, table.Row{name, m.Owner, m.Type, m.UUID})
	}
	return rows
}

// ControllerColumns defines the columns for the controller table.
func ControllerColumns() []table.Column {
	return []table.Column{
		{Title: "NAME", Width: 18},
		{Title: "CLOUD", Width: 14},
		{Title: "REGION", Width: 16},
		{Title: "VERSION", Width: 10},
		{Title: "STATUS", Width: 12},
		{Title: "HA", Width: 5},
		{Title: "MODELS", Width: 7},
		{Title: "MACHINES", Width: 9},
		{Title: "ACCESS", Width: 12},
		{Title: "ADDRESS", Width: 22},
	}
}

// ControllerRows converts a slice of controllers to table rows.
func ControllerRows(controllers []model.Controller) []table.Row {
	rows := make([]table.Row, 0, len(controllers))
	for _, c := range controllers {
		rows = append(rows, table.Row{
			c.Name,
			c.Cloud,
			c.Region,
			c.Version,
			c.Status,
			c.HA,
			fmt.Sprintf("%d", c.Models),
			fmt.Sprintf("%d", c.Machines),
			c.Access,
			c.Addr,
		})
	}
	return rows
}

// ApplicationColumns defines the columns for the application table.
func ApplicationColumns() []table.Column {
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

// ApplicationRows converts a map of applications to sorted table rows.
func ApplicationRows(apps map[string]model.Application) []table.Row {
	names := sortedKeys(apps)
	rows := make([]table.Row, 0, len(names))
	for _, name := range names {
		app := apps[name]
		exposed := "no"
		if app.Exposed {
			exposed = "yes"
		}
		rows = append(rows, table.Row{
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
	return rows
}

// UnitColumns defines the compact columns used inside the model overview panel.
func UnitColumns() []table.Column {
	return []table.Column{
		{Title: "UNIT", Width: 28},
		{Title: "WORKLOAD", Width: 12},
		{Title: "AGENT", Width: 12},
	}
}

// UnitDetailColumns defines the full columns used by the standalone units view.
func UnitDetailColumns() []table.Column {
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

// UnitRows converts a map of applications to a flat list of unit rows.
func UnitRows(apps map[string]model.Application) []table.Row {
	names := sortedKeys(apps)
	var rows []table.Row
	for _, name := range names {
		app := apps[name]
		for _, unit := range app.Units {
			rows = append(rows, unitToRow(unit))
			for _, sub := range unit.Subordinates {
				rows = append(rows, unitToRow(sub))
			}
		}
	}
	return rows
}

// UnitRowsForApp returns compact unit rows for a specific application (model panel).
func UnitRowsForApp(app model.Application) []table.Row {
	var rows []table.Row
	for _, unit := range app.Units {
		rows = append(rows, unitToRow(unit))
		for _, sub := range unit.Subordinates {
			rows = append(rows, unitToRow(sub))
		}
	}
	return rows
}

// UnitDetailRows returns full unit rows for all applications (standalone units view).
func UnitDetailRows(apps map[string]model.Application) []table.Row {
	names := sortedKeys(apps)
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

// UnitDetailRowsForApp returns full unit rows for one application (standalone units view).
func UnitDetailRowsForApp(app model.Application) []table.Row {
	var rows []table.Row
	for _, unit := range app.Units {
		rows = append(rows, unitToDetailRow(unit))
		for _, sub := range unit.Subordinates {
			rows = append(rows, unitToDetailRow(sub))
		}
	}
	return rows
}

// unitToRow builds the compact 3-column row used in the model overview panel.
// Status values are pre-coloured; leader gets a star prefix.
func unitToRow(u model.Unit) table.Row {
	var name string
	if u.Leader {
		name = lipgloss.NewStyle().Foreground(color.HintKey).Render("★") + " " + u.Name
	} else {
		name = "  " + u.Name
	}
	workload := color.StatusStyle(u.WorkloadStatus).Render(u.WorkloadStatus)
	agent := color.StatusStyle(u.AgentStatus).Render(u.AgentStatus)
	return table.Row{name, workload, agent}
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
		color.StatusStyle(u.WorkloadStatus).Render(u.WorkloadStatus),
		color.StatusStyle(u.AgentStatus).Render(u.AgentStatus),
		u.Machine,
		u.PublicAddress,
		ports,
		u.WorkloadMessage,
	}
}

// MachineColumns defines the columns for the machine table.
func MachineColumns() []table.Column {
	return []table.Column{
		{Title: "ID", Width: 6},
		{Title: "STATUS", Width: 12},
		{Title: "DNS NAME", Width: 30},
		{Title: "INSTANCE ID", Width: 14},
		{Title: "BASE", Width: 16},
		{Title: "HARDWARE", Width: 36},
	}
}

// MachineRows converts a map of machines to sorted table rows.
func MachineRows(machines map[string]model.Machine) []table.Row {
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

// RelationColumns defines the columns for the relation table.
func RelationColumns() []table.Column {
	return []table.Column{
		{Title: "ID", Width: 5},
		{Title: "ENDPOINT 1", Width: 28},
		{Title: "ENDPOINT 2", Width: 28},
		{Title: "INTERFACE", Width: 22},
		{Title: "TYPE", Width: 8},
		{Title: "STATUS", Width: 10},
	}
}

// RelationRows converts a slice of relations to table rows.
func RelationRows(relations []model.Relation) []table.Row {
	rows := make([]table.Row, 0, len(relations))
	for _, r := range relations {
		ep1, ep2 := "", ""
		if len(r.Endpoints) > 0 {
			ep := r.Endpoints[0]
			ep1 = fmt.Sprintf("%s:%s", ep.ApplicationName, ep.Name)
		}
		if len(r.Endpoints) > 1 {
			ep := r.Endpoints[1]
			ep2 = fmt.Sprintf("%s:%s", ep.ApplicationName, ep.Name)
		}
		rows = append(rows, table.Row{
			fmt.Sprintf("%d", r.ID), ep1, ep2, r.Interface, r.Scope, r.Status,
		})
	}
	return rows
}

// RelationRowsForApp returns relation rows involving a specific application.
func RelationRowsForApp(relations []model.Relation, appName string) []table.Row {
	var rows []table.Row
	for _, r := range relations {
		involved := false
		for _, ep := range r.Endpoints {
			if ep.ApplicationName == appName {
				involved = true
				break
			}
		}
		if !involved {
			continue
		}
		ep1, ep2 := "", ""
		if len(r.Endpoints) > 0 {
			ep := r.Endpoints[0]
			ep1 = fmt.Sprintf("%s:%s", ep.ApplicationName, ep.Name)
		}
		if len(r.Endpoints) > 1 {
			ep := r.Endpoints[1]
			ep2 = fmt.Sprintf("%s:%s", ep.ApplicationName, ep.Name)
		}
		rows = append(rows, table.Row{
			fmt.Sprintf("%d", r.ID), ep1, ep2, r.Interface, r.Scope, r.Status,
		})
	}
	return rows
}

// ScaleColumns adjusts column widths proportionally so their total equals availableWidth.
// Each column's cell padding (2 chars) is accounted for.
func ScaleColumns(cols []table.Column, availableWidth int) []table.Column {
	// Calculate the sum of the original (desired) widths.
	var totalDesired int
	for _, c := range cols {
		totalDesired += c.Width
	}
	if totalDesired <= 0 {
		return cols
	}

	// Account for cell padding: each column has ~2 chars of padding from the table style.
	padding := len(cols) * 2
	usable := availableWidth - padding
	if usable < len(cols) {
		usable = len(cols) // at least 1 char per column
	}

	scaled := make([]table.Column, len(cols))
	var assigned int
	for i, c := range cols {
		w := c.Width * usable / totalDesired
		if w < 1 {
			w = 1
		}
		scaled[i] = table.Column{Title: c.Title, Width: w}
		assigned += w
	}
	// Distribute any leftover to the last column.
	if diff := usable - assigned; diff > 0 && len(scaled) > 0 {
		scaled[len(scaled)-1].Width += diff
	}
	return scaled
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
