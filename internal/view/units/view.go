// Package units implements the self-contained units table view.
package units

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/nav"
	"github.com/bschimke95/jara/internal/ui"
	"github.com/bschimke95/jara/internal/view"
	"github.com/bschimke95/jara/internal/view/actionmodal"
)

// New creates a new units view. If appName is non-empty, only that app's units are shown.
func New(appName string, keys ui.KeyMap, styles *color.Styles) *View {
	cols := DetailColumns()
	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	t.SetStyles(ui.StyledTableHighlightOnly(styles))
	return &View{table: t, keys: keys, styles: styles, appName: appName, pendingScale: make(map[string]int)}
}

func (u *View) SetSize(width, height int) {
	u.width = width
	u.height = height
	u.table.SetWidth(width)
	u.table.SetHeight(height)
	u.table.SetColumns(ui.ScaleColumns(DetailColumns(), width))
}

// SetStatus implements view.StatusReceiver.
func (u *View) SetStatus(status *model.FullStatus) {
	u.status = status
	if status == nil {
		return
	}
	// Reconcile pending scale: clear entries where live unit count has caught up.
	for appName, delta := range u.pendingScale {
		app, ok := status.Applications[appName]
		if !ok {
			delete(u.pendingScale, appName)
			continue
		}
		if delta < 0 {
			if len(app.Units) <= app.Scale {
				delete(u.pendingScale, appName)
			}
		} else {
			if len(app.Units) >= app.Scale {
				delete(u.pendingScale, appName)
			}
		}
	}
	u.rebuildRows()
}

// KeyHints returns the view-specific key hints for the header.
func (u *View) KeyHints() []view.KeyHint {
	return []view.KeyHint{
		{Key: view.BindingKey(u.keys.Inspect), Desc: "info"},
		{Key: view.BindingKey(u.keys.RunAction), Desc: "action"},
		{Key: view.BindingKey(u.keys.ScaleUp) + "/" + view.BindingKey(u.keys.ScaleDown), Desc: "scale"},
		{Key: view.BindingKey(u.keys.LogsJump), Desc: "logs (unit)"},
		{Key: view.BindingKey(u.keys.EntitySwitch), Desc: "switch app"},
	}
}

// CopySelection implements view.Copyable.
func (u *View) CopySelection() string {
	return view.CopySelectedRow(u.table)
}

// rebuildRows recomputes the table rows from the current status + pending deltas.
func (u *View) rebuildRows() {
	if u.status == nil {
		return
	}
	var rows []table.Row
	if u.appName != "" {
		if app, ok := u.status.Applications[u.appName]; ok {
			rows = DetailRowsForApp(app, u.styles)
			if delta := u.pendingScale[u.appName]; delta != 0 {
				pending := PendingDetailRows(u.appName, app.Units, delta, u.styles)
				if delta < 0 {
					tail := len(rows) - len(pending)
					if tail < 0 {
						tail = 0
					}
					rows = append(rows[:tail], pending...)
				} else {
					rows = append(rows, pending...)
				}
			}
		}
	} else {
		rows = DetailRows(u.status.Applications, u.styles)
		for appName, delta := range u.pendingScale {
			if delta == 0 {
				continue
			}
			app := u.status.Applications[appName]
			pending := PendingDetailRows(appName, app.Units, delta, u.styles)
			if delta < 0 {
				prefix := "  " + appName + "/"
				end := -1
				for i, r := range rows {
					if strings.Contains(r[0], prefix) {
						end = i + 1
					}
				}
				if end >= 0 {
					start := end - len(pending)
					if start < 0 {
						start = 0
					}
					rows = append(rows[:start], append(pending, rows[end:]...)...)
				}
			} else {
				rows = append(rows, pending...)
			}
		}
	}
	u.table.SetRows(view.FilterRows(rows, 0, u.filterStr, u.styles.SearchHighlight))
}

// SetFilter implements view.Filterable.
func (u *View) SetFilter(filter string) {
	u.filterStr = filter
	u.rebuildRows()
}

func (u *View) Init() tea.Cmd { return nil }

func (u *View) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// When the action modal is open, delegate all messages to it.
	if u.actionModalOpen {
		switch msg := msg.(type) {
		case actionmodal.CloseMsg:
			u.actionModalOpen = false
			u.actionModal = nil
			return u, nil
		default:
			var cmd tea.Cmd
			newModel, cmd := u.actionModal.Update(msg)
			u.actionModal = newModel.(*actionmodal.Modal)
			return u, cmd
		}
	}

	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(msg, u.keys.ScaleUp):
			appName := u.selectedAppName()
			if appName != "" {
				u.pendingScale[appName]++
				u.rebuildRows()
				return u, func() tea.Msg { return view.ScaleRequestMsg{AppName: appName, Delta: 1} }
			}
		case key.Matches(msg, u.keys.ScaleDown):
			appName := u.selectedAppName()
			if appName != "" {
				u.pendingScale[appName]--
				u.rebuildRows()
				return u, func() tea.Msg { return view.ScaleRequestMsg{AppName: appName, Delta: -1} }
			}
		case key.Matches(msg, u.keys.LogsJump):
			var filter *model.DebugLogFilter
			if row := u.table.SelectedRow(); row != nil {
				unitName := strings.TrimSpace(row[0])
				if idx := strings.Index(unitName, " "); idx >= 0 {
					unitName = unitName[idx+1:]
				}
				unitName = strings.TrimSpace(unitName)
				if slash := strings.LastIndex(unitName, "/"); slash >= 0 {
					app := unitName[:slash]
					num := unitName[slash+1:]
					unitName = "unit-" + app + "-" + num
				}
				f := model.DebugLogFilter{IncludeEntities: []string{unitName}}
				filter = &f
			}
			return u, func() tea.Msg {
				return view.NavigateMsg{Target: nav.DebugLogView, Filter: filter}
			}
		case key.Matches(msg, u.keys.RunAction):
			unitName := u.selectedUnitName()
			appName := u.selectedAppName()
			if unitName != "" && appName != "" {
				m := actionmodal.New(unitName, appName, u.keys, u.styles)
				m.SetSize(u.width, u.height)
				u.actionModal = m
				u.actionModalOpen = true
				return u, m.Init()
			}
		}
	}
	var cmd tea.Cmd
	u.table, cmd = u.table.Update(msg)
	return u, cmd
}

// selectedAppName returns the application name for the currently highlighted row.
func (u *View) selectedAppName() string {
	if u.appName != "" {
		return u.appName
	}
	row := u.table.SelectedRow()
	if row == nil {
		return ""
	}
	unitName := strings.TrimSpace(ansi.Strip(row[0]))
	for i := len(unitName) - 1; i >= 0; i-- {
		if unitName[i] == '/' {
			return unitName[:i]
		}
	}
	return ""
}

// selectedUnitName returns the unit name (e.g. "myapp/0") for the highlighted row.
func (u *View) selectedUnitName() string {
	row := u.table.SelectedRow()
	if row == nil {
		return ""
	}
	name := strings.TrimSpace(ansi.Strip(row[0]))
	// Strip leading indicator characters like ★.
	name = strings.TrimLeft(name, "★ ")
	return name
}

func (u *View) View() tea.View {
	cursor := u.table.Cursor()
	rows := u.table.Rows()
	if cursor >= 0 && cursor < len(rows) {
		original := rows[cursor]
		stripped := make(table.Row, len(original))
		for i, cell := range original {
			stripped[i] = ansi.Strip(cell)
		}
		if len(stripped) > 0 {
			if rest, ok := strings.CutPrefix(stripped[0], "★"); ok {
				stripped[0] = color.ForegroundText(u.styles.HintKeyColor, "★") + rest
			}
		}
		if len(stripped) > 1 {
			stripped[1] = u.styles.StatusText(stripped[1])
		}
		if len(stripped) > 2 {
			stripped[2] = u.styles.StatusText(stripped[2])
		}
		rows[cursor] = stripped
		u.table.SetRows(rows)
		defer func() {
			rows[cursor] = original
			u.table.SetRows(rows)
		}()
	}
	tableView := u.table.View()
	if u.actionModalOpen && u.actionModal != nil {
		return tea.NewView(u.actionModal.Render(tableView))
	}
	return tea.NewView(tableView)
}

func (u *View) Enter(ctx view.NavigateContext) (tea.Cmd, error) {
	u.appName = ctx.Context
	u.pendingScale = make(map[string]int)
	cols := DetailColumns()
	u.table = table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	u.table.SetStyles(ui.StyledTableHighlightOnly(u.styles))
	if u.width > 0 {
		u.table.SetWidth(u.width)
		u.table.SetHeight(u.height)
		u.table.SetColumns(ui.ScaleColumns(DetailColumns(), u.width))
	}
	if u.status != nil {
		u.rebuildRows()
	}
	return nil, nil
}

func (u *View) Leave() tea.Cmd { return nil }

// SwitchTitle implements view.EntitySwitchable.
func (u *View) SwitchTitle() string { return "Switch Application" }

// SwitchableEntities implements view.EntitySwitchable.
func (u *View) SwitchableEntities() ([]string, string) {
	if u.status == nil {
		return nil, u.appName
	}
	names := make([]string, 0, len(u.status.Applications))
	for name := range u.status.Applications {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, u.appName
}

// InspectSelection implements view.Inspectable.
func (u *View) InspectSelection() *view.InspectData {
	row := u.table.SelectedRow()
	if row == nil || u.status == nil {
		return nil
	}
	unitName := ansi.Strip(row[0])
	// Find the unit in the status.
	for _, app := range u.status.Applications {
		for _, unit := range app.Units {
			if unit.Name == unitName {
				return unitInspectData(unit)
			}
			for _, sub := range unit.Subordinates {
				if sub.Name == unitName {
					return unitInspectData(sub)
				}
			}
		}
	}
	return nil
}

func unitInspectData(unit model.Unit) *view.InspectData {
	since := ""
	if unit.Since != nil {
		since = unit.Since.Format("2006-01-02 15:04:05")
	}
	return &view.InspectData{
		Title: unit.Name,
		Fields: []view.InspectField{
			{Label: "Name", Value: unit.Name},
			{Label: "Workload Status", Value: unit.WorkloadStatus},
			{Label: "Workload Message", Value: unit.WorkloadMessage},
			{Label: "Agent Status", Value: unit.AgentStatus},
			{Label: "Agent Message", Value: unit.AgentMessage},
			{Label: "Machine", Value: unit.Machine},
			{Label: "Public Address", Value: unit.PublicAddress},
			{Label: "Ports", Value: strings.Join(unit.Ports, ", ")},
			{Label: "Leader", Value: fmt.Sprintf("%v", unit.Leader)},
			{Label: "Since", Value: since},
		},
	}
}
