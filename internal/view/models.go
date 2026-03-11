package view

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/nav"
	"github.com/bschimke95/jara/internal/render"
	"github.com/bschimke95/jara/internal/ui"
)

// ModelsUpdatedMsg is sent when the model list for a controller arrives.
type ModelsUpdatedMsg struct {
	Models []model.ModelSummary
}

// Models is the Bubble Tea model for the models list view.
type Models struct {
	table  table.Model
	keys   ui.KeyMap
	width  int
	height int
	models []model.ModelSummary
}

// NewModels creates a new models view.
func NewModels() *Models {
	cols := render.ModelColumns()
	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	t.SetStyles(styledTable())
	return &Models{table: t, keys: ui.DefaultKeyMap()}
}

func (m *Models) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.table.SetWidth(width)
	m.table.SetHeight(height)
	m.table.SetColumns(render.ScaleColumns(render.ModelColumns(), width))
}

// SetStatus is a no-op — the models view uses SetModels instead.
func (m *Models) SetStatus(_ *model.FullStatus) {}

// SetModels updates the displayed model list.
func (m *Models) SetModels(models []model.ModelSummary) {
	m.models = models
	m.table.SetRows(render.ModelRows(models))
}

func (m *Models) Init() tea.Cmd { return nil }

func (m *Models) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ModelsUpdatedMsg:
		m.SetModels(msg.Models)
		return m, nil

	case tea.KeyPressMsg:
		if key.Matches(msg, m.keys.Enter) {
			if row := m.table.SelectedRow(); row != nil {
				// row[0] is the short model name (possibly with " *" suffix).
				// Find the full qualified name from m.models by index.
				if idx := m.table.Cursor(); idx < len(m.models) {
					qualifiedName := m.models[idx].Name
					return m, func() tea.Msg {
						return NavigateMsg{Target: nav.ModelView, Context: qualifiedName}
					}
				}
			}
		}
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *Models) View() tea.View {
	return tea.NewView(m.table.View())
}
