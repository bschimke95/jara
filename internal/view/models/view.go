// Package models implements the self-contained models list view.
package models

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/nav"
	"github.com/bschimke95/jara/internal/ui"
	"github.com/bschimke95/jara/internal/view"
	"github.com/bschimke95/jara/internal/view/confirmodal"
	"github.com/bschimke95/jara/internal/view/newmodelmodal"
)

// New creates a new models view.
// pollFn is called from Enter to fetch data for the given controller.
func New(keys ui.KeyMap, styles *color.Styles, pollFn func(controller string) tea.Cmd, selectControllerFn func(string) error, controllerNameFn func() string) *View {
	cols := columns()
	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	t.SetStyles(ui.StyledTableHighlightOnly(styles))
	return &View{table: t, keys: keys, styles: styles, pollFn: pollFn, selectControllerFn: selectControllerFn, controllerNameFn: controllerNameFn}
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
	m.rebuildRows()
}

// SetFilter implements view.Filterable.
func (m *View) SetFilter(filter string) {
	m.filterStr = filter
	m.rebuildRows()
}

func (m *View) rebuildRows() {
	allRows := modelRows(m.models)
	m.table.SetRows(view.FilterRows(allRows, 0, m.filterStr, m.styles.SearchHighlight))
}

// KeyHints returns the view-specific key hints for the header.
func (m *View) KeyHints() []view.KeyHint {
	return []view.KeyHint{
		{Key: view.BindingKey(m.keys.Enter), Desc: "open model"},
		{Key: view.BindingKey(m.keys.NewModel), Desc: "new model"},
		{Key: view.BindingKey(m.keys.RemoveModel), Desc: "remove model"},
		{Key: view.BindingKey(m.keys.Inspect), Desc: "info"},
	}
}

// CopySelection implements view.Copyable.
func (m *View) CopySelection() string {
	return view.CopySelectedRow(m.table)
}

func (m *View) Init() tea.Cmd { return nil }

func (m *View) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// ── New model modal takes priority ──
	if m.newModelOpen {
		switch msg := msg.(type) {
		case newmodelmodal.AppliedMsg:
			m.newModelOpen = false
			name := msg.Name
			return m, func() tea.Msg {
				return view.CreateModelRequestMsg{Name: name}
			}
		case newmodelmodal.ClosedMsg:
			m.newModelOpen = false
			return m, nil
		default:
			updated, cmd := m.newModelModal.Update(msg)
			if nm, ok := updated.(*newmodelmodal.Modal); ok {
				m.newModelModal = *nm
			}
			return m, cmd
		}
	}

	// ── Remove confirm modal takes priority ──
	if m.confirmOpen {
		switch msg := msg.(type) {
		case confirmodal.ConfirmedMsg:
			m.confirmOpen = false
			name, force := m.removingName, m.removeForce
			m.removingName = ""
			m.removeForce = false
			return m, func() tea.Msg {
				return view.DestroyModelRequestMsg{QualifiedName: name, Force: force}
			}
		case confirmodal.CancelledMsg:
			m.confirmOpen = false
			m.removingName = ""
			m.removeForce = false
			return m, nil
		default:
			if kp, ok := msg.(tea.KeyPressMsg); ok && kp.String() == "f" {
				m.removeForce = !m.removeForce
				m.confirmModal = m.buildRemoveConfirmModal()
				return m, nil
			}
			updated, cmd := m.confirmModal.Update(msg)
			if cm, ok := updated.(*confirmodal.Modal); ok {
				m.confirmModal = *cm
			}
			return m, cmd
		}
	}

	switch msg := msg.(type) {
	case UpdatedMsg:
		m.SetModels(msg.Models)
		return m, nil

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.Enter):
			if row := m.table.SelectedRow(); row != nil {
				if idx := m.table.Cursor(); idx < len(m.models) {
					qualifiedName := m.models[idx].Name
					return m, func() tea.Msg {
						return view.NavigateMsg{Target: nav.ModelView, Context: qualifiedName}
					}
				}
			}
		case key.Matches(msg, m.keys.NewModel):
			m.newModelModal = newmodelmodal.New(m.keys, m.styles)
			m.newModelModal.SetSize(m.width, m.height)
			m.newModelOpen = true
			return m, m.newModelModal.Init()
		case key.Matches(msg, m.keys.RemoveModel):
			if idx := m.table.Cursor(); idx >= 0 && idx < len(m.models) {
				m.removingName = m.models[idx].Name
				m.removeForce = false
				m.confirmModal = m.buildRemoveConfirmModal()
				m.confirmOpen = true
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *View) View() tea.View {
	bg := m.table.View()
	if m.newModelOpen {
		return tea.NewView(m.newModelModal.Render(bg))
	}
	if m.confirmOpen {
		return tea.NewView(m.confirmModal.Render(bg))
	}
	return tea.NewView(bg)
}

func (m *View) Enter(ctx view.NavigateContext) (tea.Cmd, error) {
	cols := columns()
	m.table = table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	m.table.SetStyles(ui.StyledTableHighlightOnly(m.styles))
	if m.width > 0 {
		m.table.SetWidth(m.width)
		m.table.SetHeight(m.height)
		m.table.SetColumns(ui.ScaleColumns(columns(), m.width))
	}
	m.models = nil

	controllerName := ctx.Context
	if controllerName == "" {
		controllerName = m.controllerNameFn()
	}
	if ctx.Context != "" {
		if err := m.selectControllerFn(ctx.Context); err != nil {
			return nil, err
		}
	}
	return tea.Batch(
		func() tea.Msg { return view.StopStatusStreamMsg{} },
		m.pollFn(controllerName),
	), nil
}

func (m *View) Leave() tea.Cmd { return nil }

// buildRemoveConfirmModal creates the confirmation modal for model removal.
func (m *View) buildRemoveConfirmModal() confirmodal.Modal {
	forceLabel := "off"
	if m.removeForce {
		forceLabel = "ON"
	}
	msg := fmt.Sprintf("Destroy model %s?\n\n[f] force: %s", m.removingName, forceLabel)
	cm := confirmodal.New(m.keys, m.styles, "Destroy Model", msg)
	cm.SetSize(m.width, m.height)
	return cm
}

// InspectSelection implements view.Inspectable.
func (m *View) InspectSelection() *view.InspectData {
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.models) {
		return nil
	}
	ms := m.models[idx]
	return &view.InspectData{
		Title: ms.ShortName,
		Fields: []view.InspectField{
			{Label: "Name", Value: ms.Name},
			{Label: "Short Name", Value: ms.ShortName},
			{Label: "Owner", Value: ms.Owner},
			{Label: "Type", Value: ms.Type},
			{Label: "UUID", Value: ms.UUID},
			{Label: "Current", Value: fmt.Sprintf("%v", ms.Current)},
		},
	}
}
