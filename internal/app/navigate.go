package app

import (
	tea "charm.land/bubbletea/v2"

	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/nav"
	"github.com/bschimke95/jara/internal/view"
)

func (m Model) handleNavigate(msg view.NavigateMsg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Selecting a controller from the ControllerView: switch to it and show its models.
	if msg.Target == nav.ModelsView && msg.Context != "" {
		if err := m.client.SelectController(msg.Context); err != nil {
			m.err = err
			return m, nil
		}
		m.stopStatusStream() // stop watching the previous model
		m.err = nil
		// Reset the models view so we start fresh.
		mv := view.NewModels()
		mv.SetSize(m.width, m.contentHeight())
		m.views[nav.ModelsView] = mv
		m.stack.Push(nav.StackEntry{View: nav.ModelsView, Context: msg.Context})
		return m, m.pollModels(msg.Context)
	}

	// Selecting a model from the ModelsView: switch to it and show the model detail.
	if msg.Target == nav.ModelView && msg.Context != "" {
		if err := m.client.SelectModel(msg.Context); err != nil {
			m.err = err
			return m, nil
		}
		m.status = nil
		m.err = nil
		for _, v := range m.views {
			v.SetStatus(nil)
		}
		m.stack.Push(nav.StackEntry{View: nav.ModelView})
		return m, tea.Batch(m.startStatusStream(), m.pollControllers())
	}

	m.stack.Push(nav.StackEntry{View: msg.Target, Context: msg.Context})

	if msg.Target == nav.UnitsView && msg.Context != "" {
		uv := view.NewUnits(msg.Context)
		uv.SetSize(m.width, m.contentHeight())
		if m.status != nil {
			uv.SetStatus(m.status)
		}
		m.views[nav.UnitsView] = uv
	}

	if msg.Target == nav.DebugLogView {
		var filter model.DebugLogFilter
		if msg.Filter != nil {
			// Explicit new filter (e.g. entity pre-fill from another view):
			// create a fresh view instance so the buffer is clean.
			dl := view.NewDebugLog()
			dl.SetSize(m.width, m.contentHeight())
			filter = *msg.Filter
			dl.SetFilter(filter)
			if m.status != nil {
				dl.SetStatus(m.status)
			}
			m.views[nav.DebugLogView] = dl
		} else {
			// No new filter: reuse the existing view and its current filter so
			// the user's filter state is preserved across navigation.
			if existing, ok := m.views[nav.DebugLogView].(*view.DebugLog); ok {
				filter = existing.ActiveFilter()
			} else {
				dl := view.NewDebugLog()
				dl.SetSize(m.width, m.contentHeight())
				if m.status != nil {
					dl.SetStatus(m.status)
				}
				m.views[nav.DebugLogView] = dl
			}
		}
		cmds = append(cmds, m.startDebugLogStream(filter))
	}

	return m, tea.Batch(cmds...)
}

func (m Model) handleBack() (Model, tea.Cmd) {
	prev := m.stack.Current()
	if _, ok := m.stack.Pop(); ok {
		// Stop debug-log stream when leaving that view.
		if prev.View == nav.DebugLogView {
			m.stopDebugLogStream()
		}
		current := m.stack.Current()
		if current.View == nav.UnitsView && current.Context == "" {
			uv := view.NewUnits("")
			uv.SetSize(m.width, m.contentHeight())
			if m.status != nil {
				uv.SetStatus(m.status)
			}
			m.views[nav.UnitsView] = uv
		}
	}
	return m, nil
}
