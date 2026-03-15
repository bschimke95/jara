package view

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/render"
	"github.com/bschimke95/jara/internal/ui"
)

// Units is the Bubble Tea model for the units table view.
type Units struct {
	table        table.Model
	keys         ui.KeyMap
	width        int
	height       int
	status       *model.FullStatus
	appName      string
	pendingScale map[string]int // net pending unit delta per app
}

// NewUnits creates a new units view. If appName is non-empty, only that app's units are shown.
func NewUnits(appName string) *Units {
	cols := render.UnitDetailColumns()
	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	t.SetStyles(styledTableHighlightOnly())
	return &Units{table: t, keys: ui.DefaultKeyMap(), appName: appName, pendingScale: make(map[string]int)}
}

func (u *Units) SetSize(width, height int) {
	u.width = width
	u.height = height
	u.table.SetWidth(width)
	u.table.SetHeight(height)
	u.table.SetColumns(render.ScaleColumns(render.UnitDetailColumns(), width))
}

func (u *Units) SetStatus(status *model.FullStatus) {
	u.status = status
	if status == nil {
		return
	}
	// Reconcile pending scale: clear entries where live unit count has caught up.
	// app.Scale is the desired (post-request) value; compare unit count against it.
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

// rebuildRows recomputes the table rows from the current status + pending deltas.
func (u *Units) rebuildRows() {
	if u.status == nil {
		return
	}
	var rows []table.Row
	if u.appName != "" {
		if app, ok := u.status.Applications[u.appName]; ok {
				rows = render.UnitDetailRowsForApp(app)
			if delta := u.pendingScale[u.appName]; delta != 0 {
				pending := render.PendingUnitDetailRows(u.appName, app.Units, delta)
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
		rows = render.UnitDetailRows(u.status.Applications)
		// For the all-units view, patch pending rows per app.
		for appName, delta := range u.pendingScale {
			if delta == 0 {
				continue
			}
			app := u.status.Applications[appName]
			pending := render.PendingUnitDetailRows(appName, app.Units, delta)
			if delta < 0 {
				// Replace the last len(pending) rows belonging to this app.
				// Scan for consecutive rows that contain the app prefix and
				// record the index past the last one (end).
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

func (u *Units) Init() tea.Cmd { return nil }

func (u *Units) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(msg, u.keys.ScaleUp):
			appName := u.selectedAppName()
			if appName != "" {
				u.pendingScale[appName]++
				u.rebuildRows()
				return u, func() tea.Msg { return ScaleRequestMsg{AppName: appName, Delta: 1} }
			}
		case key.Matches(msg, u.keys.ScaleDown):
			appName := u.selectedAppName()
			if appName != "" {
				u.pendingScale[appName]--
				u.rebuildRows()
				return u, func() tea.Msg { return ScaleRequestMsg{AppName: appName, Delta: -1} }
			}
		}
	}
	var cmd tea.Cmd
	u.table, cmd = u.table.Update(msg)
	return u, cmd
}

// selectedAppName returns the application name for the currently highlighted row,
// or u.appName if this view is scoped to a specific application.
func (u *Units) selectedAppName() string {
	if u.appName != "" {
		return u.appName
	}
	// For the all-units view, derive the app name from the unit name ("app/N").
	row := u.table.SelectedRow()
	if row == nil {
		return ""
	}
	// Unit name is the first column; split on "/" to get app name.
	unitName := row[0]
	if idx := len(unitName) - 1; idx >= 0 {
		for i := len(unitName) - 1; i >= 0; i-- {
			if unitName[i] == '/' {
				return unitName[:i]
			}
		}
	}
	return ""
}

func (u *Units) View() tea.View {
	// The selected row is rendered by wrapping the already-coloured cells with
	// the Selected style's background. Inner ANSI resets break the outer
	// background, so we temporarily replace the cursor row with stripped cells.
	cursor := u.table.Cursor()
	rows := u.table.Rows()
	if cursor >= 0 && cursor < len(rows) {
		original := rows[cursor]
		stripped := make(table.Row, len(original))
		for i, cell := range original {
			stripped[i] = ansi.Strip(cell)
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
