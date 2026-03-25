package app

import (
	tea "charm.land/bubbletea/v2"

	"github.com/bschimke95/jara/internal/nav"
	"github.com/bschimke95/jara/internal/view"
)

func (m Model) handleNavigate(msg view.NavigateMsg) (Model, tea.Cmd) {
	target := m.views[msg.Target]
	target.SetSize(m.width, m.contentHeight())
	if m.status != nil {
		if sr, ok := target.(view.StatusReceiver); ok {
			sr.SetStatus(m.status)
		}
	}

	navCtx := view.NavigateContext{Context: msg.Context, Filter: msg.Filter}
	cmd, err := target.Enter(navCtx)
	if err != nil {
		m.err = err
		return m, nil
	}
	m.err = nil

	entry := nav.StackEntry{View: msg.Target, Context: msg.Context}
	if msg.ResetStack {
		m.stack.Reset(entry)
	} else {
		m.stack.Push(entry)
	}
	if cmd != nil {
		return m, cmd
	}
	return m, nil
}

func (m Model) handleBack() (Model, tea.Cmd) {
	prev := m.stack.Current()
	if _, ok := m.stack.Pop(); ok {
		var cmds []tea.Cmd

		if cmd := m.views[prev.View].Leave(); cmd != nil {
			cmds = append(cmds, cmd)
		}

		current := m.stack.Current()

		cmd, err := m.views[current.View].Enter(view.NavigateContext{Context: current.Context})
		if err != nil {
			// Re-push to undo the pop — back navigation failed.
			m.stack.Push(prev)
			m.err = err
			return m, nil
		}
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		return m, tea.Batch(cmds...)
	}
	return m, nil
}
