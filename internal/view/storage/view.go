// Package storage implements the self-contained storage table view.
package storage

import (
	"fmt"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/ui"
	"github.com/bschimke95/jara/internal/view"
)

// New creates a new storage view.
func New(keys ui.KeyMap, styles *color.Styles) *View {
	cols := Columns()
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
	v.table.SetColumns(ui.ScaleColumns(Columns(), width))
}

// KeyHints returns the view-specific key hints for the header.
func (v *View) KeyHints() []view.KeyHint {
	return []view.KeyHint{
		{Key: view.BindingKey(v.keys.Inspect), Desc: "info"},
	}
}

func (v *View) Init() tea.Cmd { return nil }

func (v *View) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case StorageDataMsg:
		if msg.Err != nil {
			v.err = msg.Err
			return v, nil
		}
		v.err = nil
		v.hasData = true
		v.instances = msg.Instances
		v.rebuildRows()
		return v, nil
	}
	var cmd tea.Cmd
	v.table, cmd = v.table.Update(msg)
	return v, cmd
}

func (v *View) View() tea.View {
	if v.err != nil {
		return tea.NewView(v.styles.ErrorStyle.Render("Error: " + v.err.Error()))
	}
	if !v.hasData {
		return tea.NewView(v.styles.MutedText.Render("Loading storage..."))
	}
	if len(v.table.Rows()) == 0 {
		return tea.NewView(v.styles.MutedText.Render("No storage instances found."))
	}
	return tea.NewView(v.table.View())
}

func (v *View) Enter(_ view.NavigateContext) (tea.Cmd, error) {
	return func() tea.Msg { return FetchStorageMsg{} }, nil
}

func (v *View) Leave() tea.Cmd { return nil }

// SetFilter implements view.Filterable.
func (v *View) SetFilter(filter string) {
	v.filterStr = filter
	v.rebuildRows()
}

func (v *View) rebuildRows() {
	allRows := Rows(v.instances, v.styles)
	v.table.SetRows(view.FilterRows(allRows, 0, v.filterStr, v.styles.SearchHighlight))
}

// InspectSelection implements view.Inspectable.
func (v *View) InspectSelection() *view.InspectData {
	idx := v.table.Cursor()
	if idx < 0 || idx >= len(v.instances) {
		return nil
	}
	si := v.instances[idx]
	return &view.InspectData{
		Title: si.ID,
		Fields: []view.InspectField{
			{Label: "ID", Value: si.ID},
			{Label: "Kind", Value: si.Kind},
			{Label: "Owner", Value: si.Owner},
			{Label: "Status", Value: si.Status},
			{Label: "Persistent", Value: fmt.Sprintf("%v", si.Persistent)},
			{Label: "Life", Value: si.Life},
			{Label: "Pool", Value: si.Pool},
		},
	}
}
