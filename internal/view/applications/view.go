// Package applications implements the self-contained applications table view.
package applications

import (
	"fmt"

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
	"github.com/bschimke95/jara/internal/view/deploymodal"
)

// New creates a new applications view.
func New(keys ui.KeyMap, styles *color.Styles) *View {
	cols := columns()
	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	t.SetStyles(ui.StyledTableHighlightOnly(styles))
	return &View{table: t, keys: keys, styles: styles}
}

func (a *View) SetSize(width, height int) {
	a.width = width
	a.height = height
	a.table.SetWidth(width)
	a.table.SetHeight(height)
	a.table.SetColumns(ui.ScaleColumns(columns(), width))
}

// SetStatus implements view.StatusReceiver.
func (a *View) SetStatus(status *model.FullStatus) {
	a.status = status
	a.rebuildRows()
}

// SetFilter implements view.Filterable.
func (a *View) SetFilter(filter string) {
	a.filterStr = filter
	a.rebuildRows()
}

func (a *View) rebuildRows() {
	if a.status == nil {
		return
	}
	allRows := rows(a.status.Applications, a.styles)
	a.table.SetRows(view.FilterRows(allRows, 0, a.filterStr, a.styles.SearchHighlight))
}

// SetCharmSuggestions stores external charm suggestions for deploy modal.
func (a *View) SetCharmSuggestions(names []string) {
	a.charmhubSuggestions = append([]string(nil), names...)
}

// KeyHints returns the view-specific key hints for the header.
func (a *View) KeyHints() []view.KeyHint {
	return []view.KeyHint{
		{Key: view.BindingKey(a.keys.Enter), Desc: "units"},
		{Key: view.BindingKey(a.keys.Inspect), Desc: "info"},
		{Key: view.BindingKey(a.keys.RunAction), Desc: "action"},
		{Key: view.BindingKey(a.keys.ConfigNav), Desc: "config"},
		{Key: view.BindingKey(a.keys.Deploy), Desc: "deploy"},
		{Key: view.BindingKey(a.keys.LogsJump), Desc: "logs (app)"},
	}
}

// CopySelection implements view.Copyable.
func (a *View) CopySelection() string {
	return view.CopySelectedRow(a.table)
}

func (a *View) Init() tea.Cmd { return nil }

func (a *View) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Action modal takes priority when open.
	if a.actionModalOpen {
		switch msg.(type) {
		case actionmodal.CloseMsg:
			a.actionModalOpen = false
			a.actionModal = nil
			return a, nil
		default:
			var cmd tea.Cmd
			newModel, cmd := a.actionModal.Update(msg)
			a.actionModal = newModel.(*actionmodal.Modal)
			return a, cmd
		}
	}

	if a.deployModalOpen {
		switch msg := msg.(type) {
		case deploymodal.AppliedMsg:
			a.deployModalOpen = false
			return a, func() tea.Msg {
				return view.DeployRequestMsg{ModelName: msg.ModelName, Options: msg.Options}
			}
		case deploymodal.ClosedMsg:
			a.deployModalOpen = false
			return a, nil
		default:
			updated, cmd := a.deployModal.Update(msg)
			if dm, ok := updated.(*deploymodal.Modal); ok {
				a.deployModal = *dm
			}
			return a, cmd
		}
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if key.Matches(msg, a.keys.Deploy) {
			a.deployModal = deploymodal.New("", a.keys, a.styles, a.charmSuggestions(), a.applicationSuggestions())
			a.deployModal.SetSize(a.width, a.height)
			a.deployModalOpen = true
			return a, a.deployModal.BeginCharmEdit()
		}
		if key.Matches(msg, a.keys.Enter) {
			if row := a.table.SelectedRow(); row != nil {
				return a, func() tea.Msg {
					return view.NavigateMsg{Target: nav.UnitsView, Context: row[0]}
				}
			}
		}
		if key.Matches(msg, a.keys.LogsJump) {
			var filter *model.DebugLogFilter
			if row := a.table.SelectedRow(); row != nil {
				f := model.DebugLogFilter{Applications: []string{row[0]}}
				filter = &f
			}
			return a, func() tea.Msg {
				return view.NavigateMsg{Target: nav.DebugLogView, Filter: filter}
			}
		}
		if key.Matches(msg, a.keys.ConfigNav) {
			if row := a.table.SelectedRow(); row != nil {
				return a, func() tea.Msg {
					return view.NavigateMsg{Target: nav.AppConfigView, Context: row[0]}
				}
			}
		}
		if key.Matches(msg, a.keys.RunAction) {
			if row := a.table.SelectedRow(); row != nil && a.status != nil {
				appName := ansi.Strip(row[0])
				if app, ok := a.status.Applications[appName]; ok {
					unitNames := view.UnitNamesLeaderFirst(app)
					if len(unitNames) > 0 {
						m := actionmodal.NewWithUnits(appName, unitNames, a.keys, a.styles)
						m.SetSize(a.width, a.height)
						a.actionModal = m
						a.actionModalOpen = true
						return a, m.Init()
					}
				}
			}
		}
	}
	var cmd tea.Cmd
	a.table, cmd = a.table.Update(msg)
	return a, cmd
}

func (a *View) View() tea.View {
	background := a.tableView()
	if a.actionModalOpen && a.actionModal != nil {
		return tea.NewView(a.actionModal.Render(background))
	}
	if a.deployModalOpen {
		return tea.NewView(a.deployModal.Render(background))
	}
	return tea.NewView(background)
}

func (a *View) tableView() string {
	cursor := a.table.Cursor()
	rows := a.table.Rows()
	if cursor >= 0 && cursor < len(rows) {
		original := rows[cursor]
		stripped := make(table.Row, len(original))
		for i, cell := range original {
			stripped[i] = ansi.Strip(cell)
		}
		if len(stripped) > 1 {
			stripped[1] = a.styles.StatusText(stripped[1])
		}
		rows[cursor] = stripped
		a.table.SetRows(rows)
		defer func() {
			rows[cursor] = original
			a.table.SetRows(rows)
		}()
	}
	return a.table.View()
}

func (a *View) charmSuggestions() []string {
	out := append([]string(nil), a.charmhubSuggestions...)
	if a.status == nil {
		return out
	}
	for _, appName := range ui.SortedKeys(a.status.Applications) {
		app := a.status.Applications[appName]
		if app.Charm != "" {
			out = append(out, app.Charm)
		}
	}
	return out
}

func (a *View) applicationSuggestions() []string {
	if a.status == nil {
		return nil
	}
	out := make([]string, 0, len(a.status.Applications))
	out = append(out, ui.SortedKeys(a.status.Applications)...)
	return out
}

func (a *View) Enter(_ view.NavigateContext) (tea.Cmd, error) { return nil, nil }
func (a *View) Leave() tea.Cmd                                { return nil }

// InspectSelection implements view.Inspectable.
func (a *View) InspectSelection() *view.InspectData {
	row := a.table.SelectedRow()
	if row == nil || a.status == nil {
		return nil
	}
	name := ansi.Strip(row[0])
	app, ok := a.status.Applications[name]
	if !ok {
		return nil
	}
	since := ""
	if app.Since != nil {
		since = app.Since.Format("2006-01-02 15:04:05")
	}
	fields := []view.InspectField{
		{Label: "Name", Value: app.Name},
		{Label: "Status", Value: app.Status},
		{Label: "Status Message", Value: app.StatusMessage},
		{Label: "Charm", Value: app.Charm},
		{Label: "Channel", Value: app.CharmChannel},
		{Label: "Revision", Value: fmt.Sprintf("%d", app.CharmRev)},
		{Label: "Scale", Value: fmt.Sprintf("%d", app.Scale)},
		{Label: "Exposed", Value: fmt.Sprintf("%v", app.Exposed)},
		{Label: "Workload Version", Value: app.WorkloadVersion},
		{Label: "Base", Value: app.Base},
		{Label: "Since", Value: since},
	}
	// Append per-unit detail so the full workload/agent messages are visible.
	for _, u := range app.Units {
		fields = appendUnitFields(fields, u)
		for _, sub := range u.Subordinates {
			fields = appendUnitFields(fields, sub)
		}
	}
	return &view.InspectData{
		Title:  name,
		Fields: fields,
	}
}

func appendUnitFields(fields []view.InspectField, u model.Unit) []view.InspectField {
	leader := ""
	if u.Leader {
		leader = "*"
	}
	fields = append(fields, view.InspectField{
		Label: fmt.Sprintf("── %s %s", u.Name, leader),
		Value: fmt.Sprintf("%s: %s | agent: %s: %s",
			u.WorkloadStatus, u.WorkloadMessage,
			u.AgentStatus, u.AgentMessage),
	})
	return fields
}
