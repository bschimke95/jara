// Package models implements the self-contained models list view.
package models

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/nav"
	"github.com/bschimke95/jara/internal/ui"
	"github.com/bschimke95/jara/internal/view"
)

// New creates a new models view.
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

// SetModels updates the displayed model list.
func (m *View) SetModels(mdls []model.ModelSummary) {
	m.models = mdls
	m.table.SetRows(modelRows(mdls))
}

// KeyHints returns the view-specific key hints for the header.
func (m *View) KeyHints() []view.KeyHint {
	bk := func(b key.Binding) string { return b.Help().Key }
	return []view.KeyHint{
		{Key: bk(m.keys.Enter), Desc: "open model"},
	}
}

func (m *View) Init() tea.Cmd { return nil }

func (m *View) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case UpdatedMsg:
		m.SetModels(msg.Models)
		return m, nil

	case tea.KeyPressMsg:
		if key.Matches(msg, m.keys.Enter) {
			if row := m.table.SelectedRow(); row != nil {
				if idx := m.table.Cursor(); idx < len(m.models) {
					qualifiedName := m.models[idx].Name
					return m, func() tea.Msg {
						return view.NavigateMsg{Target: nav.ModelView, Context: qualifiedName}
					}
				}
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
