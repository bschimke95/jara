// Package machines implements the self-contained machines table view.
package machines

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/nav"
	"github.com/bschimke95/jara/internal/ui"
	"github.com/bschimke95/jara/internal/view"
)

// New creates a new machines view.
func New(keys ui.KeyMap) *View {
	cols := columns()
	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	t.SetStyles(ui.StyledTable())
	return &View{table: t, keys: keys}
}

func (m *View) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.table.SetWidth(width)
	m.table.SetHeight(height)
	m.table.SetColumns(ui.ScaleColumns(columns(), width))
}

// SetStatus implements view.StatusReceiver.
func (m *View) SetStatus(status *model.FullStatus) {
	m.status = status
	if status != nil {
		m.table.SetRows(machineRows(status.Machines))
	}
}

// KeyHints returns the view-specific key hints for the header.
func (m *View) KeyHints() []view.KeyHint {
	bk := func(b key.Binding) string { return b.Help().Key }
	return []view.KeyHint{
		{Key: bk(m.keys.LogsJump), Desc: "logs (machine)"},
		{Key: bk(m.keys.LogsView), Desc: "logs"},
	}
}

func (m *View) Init() tea.Cmd { return nil }

func (m *View) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if kp, ok := msg.(tea.KeyPressMsg); ok {
		if key.Matches(kp, m.keys.LogsJump) {
			var filter *model.DebugLogFilter
			if row := m.table.SelectedRow(); row != nil {
				f := model.DebugLogFilter{IncludeEntities: []string{"machine-" + row[0]}}
				filter = &f
			}
			return m, func() tea.Msg {
				return view.NavigateMsg{Target: nav.DebugLogView, Filter: filter}
			}
		}
		if key.Matches(kp, m.keys.LogsView) {
			return m, func() tea.Msg {
				return view.NavigateMsg{Target: nav.DebugLogView}
			}
		}
	}
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *View) View() tea.View {
	return tea.NewView(m.table.View())
}
