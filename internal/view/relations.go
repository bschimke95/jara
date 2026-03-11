package view

import (
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/render"
	"github.com/bschimke95/jara/internal/ui"
)

// Relations is the Bubble Tea model for the relations table view.
type Relations struct {
	table  table.Model
	keys   ui.KeyMap
	width  int
	height int
	status *model.FullStatus
}

// NewRelations creates a new relations view.
func NewRelations() *Relations {
	cols := render.RelationColumns()
	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	t.SetStyles(styledTable())
	return &Relations{table: t, keys: ui.DefaultKeyMap()}
}

func (r *Relations) SetSize(width, height int) {
	r.width = width
	r.height = height
	r.table.SetWidth(width)
	r.table.SetHeight(height)
	r.table.SetColumns(render.ScaleColumns(render.RelationColumns(), width))
}

func (r *Relations) SetStatus(status *model.FullStatus) {
	r.status = status
	if status != nil {
		r.table.SetRows(render.RelationRows(status.Relations))
	}
}

func (r *Relations) Init() tea.Cmd { return nil }

func (r *Relations) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	r.table, cmd = r.table.Update(msg)
	return r, cmd
}

func (r *Relations) View() tea.View {
	return tea.NewView(r.table.View())
}
