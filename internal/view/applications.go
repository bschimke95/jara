package view

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/nav"
	"github.com/bschimke95/jara/internal/render"
	"github.com/bschimke95/jara/internal/ui"
)

// Applications is the Bubble Tea model for the applications table view.
type Applications struct {
	table  table.Model
	keys   ui.KeyMap
	width  int
	height int
	status *model.FullStatus
}

// NewApplications creates a new applications view.
func NewApplications() *Applications {
	cols := render.ApplicationColumns()
	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	t.SetStyles(styledTable())
	return &Applications{table: t, keys: ui.DefaultKeyMap()}
}

func (a *Applications) SetSize(width, height int) {
	a.width = width
	a.height = height
	a.table.SetWidth(width)
	a.table.SetHeight(height)
	a.table.SetColumns(render.ScaleColumns(render.ApplicationColumns(), width))
}

func (a *Applications) SetStatus(status *model.FullStatus) {
	a.status = status
	if status != nil {
		a.table.SetRows(render.ApplicationRows(status.Applications))
	}
}

func (a *Applications) Init() tea.Cmd { return nil }

func (a *Applications) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if key.Matches(msg, a.keys.Enter) {
			if row := a.table.SelectedRow(); row != nil {
				return a, func() tea.Msg {
					return NavigateMsg{Target: nav.UnitsView, Context: row[0]}
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
				return NavigateMsg{Target: nav.DebugLogView, Filter: filter}
			}
		}
		if key.Matches(msg, a.keys.LogsView) {
			return a, func() tea.Msg {
				return NavigateMsg{Target: nav.DebugLogView}
			}
		}
	}
	var cmd tea.Cmd
	a.table, cmd = a.table.Update(msg)
	return a, cmd
}

func (a *Applications) View() tea.View {
	return tea.NewView(a.table.View())
}

func styledTable() table.Styles {
	s := table.DefaultStyles()
	s.Header = lipgloss.NewStyle().
		Bold(true).
		Foreground(color.Primary).
		Padding(0, 1)
	s.Selected = lipgloss.NewStyle().
		Foreground(color.CrumbFg).
		Background(color.Highlight).
		Bold(true)
	s.Cell = lipgloss.NewStyle().
		Padding(0, 1)
	return s
}

// styledTableHighlightOnly is like styledTable but the selected row only sets a
// background highlight without overriding the cell foreground colour. This lets
// pre-coloured status values (workload, agent) remain readable when highlighted.
func styledTableHighlightOnly() table.Styles {
	s := table.DefaultStyles()
	s.Header = lipgloss.NewStyle().
		Bold(true).
		Foreground(color.Primary).
		Padding(0, 1)
	s.Selected = lipgloss.NewStyle().
		Background(color.Highlight).
		Bold(true)
	s.Cell = lipgloss.NewStyle().
		Padding(0, 1)
	return s
}
