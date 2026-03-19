// Package units implements the self-contained units table view.
package units

import (
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
)

// New creates a new units view. If appName is non-empty, only that app's units are shown.
func New(appName string, keys ui.KeyMap) *View {
	cols := DetailColumns()
	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	t.SetStyles(ui.StyledTableHighlightOnly())
	return &View{table: t, keys: keys, appName: appName, pendingScale: make(map[string]int)}
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
	bk := func(b key.Binding) string { return b.Help().Key }
	return []view.KeyHint{
		{Key: bk(u.keys.ScaleUp) + "/" + bk(u.keys.ScaleDown), Desc: "scale"},
		{Key: bk(u.keys.LogsJump), Desc: "logs (unit)"},
		{Key: bk(u.keys.LogsView), Desc: "logs"},
	}
}

// rebuildRows recomputes the table rows from the current status + pending deltas.
func (u *View) rebuildRows() {
	if u.status == nil {
		return
	}
	var rows []table.Row
	if u.appName != "" {
		if app, ok := u.status.Applications[u.appName]; ok {
			rows = DetailRowsForApp(app)
			if delta := u.pendingScale[u.appName]; delta != 0 {
				pending := PendingDetailRows(u.appName, app.Units, delta)
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
		rows = DetailRows(u.status.Applications)
		for appName, delta := range u.pendingScale {
			if delta == 0 {
				continue
			}
			app := u.status.Applications[appName]
			pending := PendingDetailRows(appName, app.Units, delta)
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
	u.table.SetRows(rows)
}

func (u *View) Init() tea.Cmd { return nil }

func (u *View) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		case key.Matches(msg, u.keys.LogsView):
			return u, func() tea.Msg {
				return view.NavigateMsg{Target: nav.DebugLogView}
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
	unitName := row[0]
	for i := len(unitName) - 1; i >= 0; i-- {
		if unitName[i] == '/' {
			return unitName[:i]
		}
	}
	return ""
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
				stripped[0] = color.ForegroundText(color.HintKey, "★") + rest
			}
		}
		if len(stripped) > 1 {
			stripped[1] = color.StatusText(stripped[1])
		}
		if len(stripped) > 2 {
			stripped[2] = color.StatusText(stripped[2])
		}
		rows[cursor] = stripped
		u.table.SetRows(rows)
		defer func() {
			rows[cursor] = original
			u.table.SetRows(rows)
		}()
	}
	return tea.NewView(u.table.View())
}
