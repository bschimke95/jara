// Package storage implements the self-contained storage table view.
package storage

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
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
		{Key: view.BindingKey(v.keys.EntitySwitch), Desc: "switch app"},
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

func (v *View) Enter(ctx view.NavigateContext) (tea.Cmd, error) {
	v.appName = ctx.Context
	return func() tea.Msg { return FetchStorageMsg{} }, nil
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
	// Derive app names from storage owners (owner is like "unit-myapp-0" or "myapp").
	seen := make(map[string]bool)
	for _, si := range v.instances {
		app := storageOwnerApp(si.Owner)
		if app != "" {
			seen[app] = true
		}
	}
	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)
	return names, v.appName
}

// storageOwnerApp extracts the application name from a storage owner string.
// Owner can be "unit-myapp-0" or just "myapp".
func storageOwnerApp(owner string) string {
	if strings.HasPrefix(owner, "unit-") {
		// "unit-myapp-0" -> "myapp"
		rest := owner[5:]
		if idx := strings.LastIndex(rest, "-"); idx > 0 {
			return rest[:idx]
		}
	}
	return owner
}

// SetFilter implements view.Filterable.
func (v *View) SetFilter(filter string) {
	v.filterStr = filter
	v.rebuildRows()
}

func (v *View) rebuildRows() {
	var filtered []model.StorageInstance
	for _, si := range v.instances {
		if v.appName != "" && storageOwnerApp(si.Owner) != v.appName {
			continue
		}
		filtered = append(filtered, si)
	}
	allRows := Rows(filtered, v.styles)
	v.table.SetRows(view.FilterRows(allRows, 0, v.filterStr, v.styles.SearchHighlight))
}

// InspectSelection implements view.Inspectable.
func (v *View) InspectSelection() *view.InspectData {
	row := v.table.SelectedRow()
	if row == nil {
		return nil
	}
	id := row[0]
	var si model.StorageInstance
	found := false
	for _, inst := range v.instances {
		if inst.ID == id {
			si = inst
			found = true
			break
		}
	}
	if !found {
		return nil
	}
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
