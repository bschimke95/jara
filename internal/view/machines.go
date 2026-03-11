package view

import (
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/render"
	"github.com/bschimke95/jara/internal/ui"
)

// Machines is the Bubble Tea model for the machines table view.
type Machines struct {
	table  table.Model
	keys   ui.KeyMap
	width  int
	height int
	status *model.FullStatus
}

// NewMachines creates a new machines view.
func NewMachines() *Machines {
	cols := render.MachineColumns()
	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	t.SetStyles(styledTable())
	return &Machines{table: t, keys: ui.DefaultKeyMap()}
}

func (m *Machines) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.table.SetWidth(width)
	m.table.SetHeight(height)
	m.table.SetColumns(render.ScaleColumns(render.MachineColumns(), width))
}

func (m *Machines) SetStatus(status *model.FullStatus) {
	m.status = status
	if status != nil {
		m.table.SetRows(render.MachineRows(status.Machines))
	}
}

func (m *Machines) Init() tea.Cmd { return nil }

func (m *Machines) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *Machines) View() tea.View {
	return tea.NewView(m.table.View())
}
