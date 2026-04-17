// Package appconfig implements the application configuration table view.
package appconfig

import (
	"fmt"
	"sort"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/ui"
	"github.com/bschimke95/jara/internal/view"
)

// FetchAppConfigMsg is emitted by the view on Enter to request config data.
type FetchAppConfigMsg struct {
	AppName string
}

// AppConfigMsg delivers fetched config entries to the view.
type AppConfigMsg struct {
	AppName string
	Entries []model.ConfigEntry
}

// New creates a new application config view.
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

func (v *View) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.table.SetWidth(width)
	v.table.SetHeight(height)
	v.table.SetColumns(ui.ScaleColumns(columns(), width))
}

// CopySelection returns the value of the currently selected config entry.
func (v *View) CopySelection() string {
	row := v.table.SelectedRow()
	if len(row) < 2 {
		return ""
	}
	return row[1] // VALUE column
}

// KeyHints returns the view-specific key hints for the header.
func (v *View) KeyHints() []view.KeyHint {
	return []view.KeyHint{
		{Key: view.BindingKey(v.keys.Inspect), Desc: "info"},
		{Key: view.BindingKey(v.keys.Back), Desc: "back"},
		{Key: view.BindingKey(v.keys.Yank), Desc: "copy value"},
		{Key: view.BindingKey(v.keys.EntitySwitch), Desc: "switch app"},
	}
}

func (v *View) Init() tea.Cmd { return nil }

func (v *View) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if cfgMsg, ok := msg.(AppConfigMsg); ok {
		if cfgMsg.AppName == v.appName {
			v.entries = cfgMsg.Entries
			v.rebuildRows()
		}
		return v, nil
	}
	var cmd tea.Cmd
	v.table, cmd = v.table.Update(msg)
	return v, cmd
}

func (v *View) View() tea.View {
	if v.appName == "" {
		return tea.NewView("No application selected")
	}
	if len(v.entries) == 0 {
		return tea.NewView("Loading config for " + v.appName + "...")
	}
	return tea.NewView(v.table.View())
}

func (v *View) Enter(ctx view.NavigateContext) (tea.Cmd, error) {
	if ctx.Context == "" {
		return nil, fmt.Errorf("no application specified — press C on an application to view its config")
	}
	v.appName = ctx.Context
	v.entries = nil
	v.table.SetRows(nil)
	v.table.SetCursor(0)
	// Request config fetch from the app layer.
	appName := v.appName
	return func() tea.Msg {
		return FetchAppConfigMsg{AppName: appName}
	}, nil
}

func (v *View) Leave() tea.Cmd { return nil }

// SetStatus implements view.StatusReceiver.
func (v *View) SetStatus(status *model.FullStatus) {
	v.status = status
}

// SwitchTitle implements view.EntitySwitchable.
func (v *View) SwitchTitle() string { return "Switch Application" }

// SwitchableEntities implements view.EntitySwitchable.
func (v *View) SwitchableEntities() ([]string, string) {
	if v.status == nil {
		return nil, v.appName
	}
	names := make([]string, 0, len(v.status.Applications))
	for name := range v.status.Applications {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, v.appName
}

// SetFilter implements view.Filterable.
func (v *View) SetFilter(filter string) {
	v.filterStr = filter
	v.rebuildRows()
}

func (v *View) rebuildRows() {
	allRows := rows(v.entries)
	v.table.SetRows(view.FilterRows(allRows, 0, v.filterStr, v.styles.SearchHighlight))
}

// InspectSelection implements view.Inspectable.
func (v *View) InspectSelection() *view.InspectData {
	idx := v.table.Cursor()
	if idx < 0 || idx >= len(v.entries) {
		return nil
	}
	e := v.entries[idx]
	return &view.InspectData{
		Title: e.Key,
		Fields: []view.InspectField{
			{Label: "Key", Value: e.Key},
			{Label: "Value", Value: e.Value},
			{Label: "Default", Value: e.Default},
			{Label: "Source", Value: e.Source},
			{Label: "Type", Value: e.Type},
			{Label: "Description", Value: e.Description},
		},
	}
}
