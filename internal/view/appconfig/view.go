// Package appconfig implements the application configuration table view.
package appconfig

import (
	"fmt"

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

// KeyHints returns the view-specific key hints for the header.
func (v *View) KeyHints() []view.KeyHint {
	return []view.KeyHint{
		{Key: view.BindingKey(v.keys.Back), Desc: "back"},
	}
}

func (v *View) Init() tea.Cmd { return nil }

func (v *View) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if cfgMsg, ok := msg.(AppConfigMsg); ok {
		if cfgMsg.AppName == v.appName {
			v.entries = cfgMsg.Entries
			v.table.SetRows(rows(v.entries))
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
