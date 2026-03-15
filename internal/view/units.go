package view

import (
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/render"
	"github.com/bschimke95/jara/internal/ui"
)

// Units is the Bubble Tea model for the units table view.
type Units struct {
	table   table.Model
	keys    ui.KeyMap
	width   int
	height  int
	status  *model.FullStatus
	appName string
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
	return &Units{table: t, keys: ui.DefaultKeyMap(), appName: appName}
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
	var rows []table.Row
	if u.appName != "" {
		if app, ok := status.Applications[u.appName]; ok {
			rows = render.UnitDetailRowsForApp(app)
		}
	} else {
		rows = render.UnitDetailRows(status.Applications)
	}
	u.table.SetRows(rows)
}

func (u *Units) Init() tea.Cmd { return nil }

func (u *Units) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	u.table, cmd = u.table.Update(msg)
	return u, cmd
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
