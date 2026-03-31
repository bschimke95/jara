package app

import (
	tea "charm.land/bubbletea/v2"

	"github.com/bschimke95/jara/internal/nav"
	"github.com/bschimke95/jara/internal/view"
)

func (m Model) handleNavigate(msg view.NavigateMsg) (Model, tea.Cmd) {
	target := m.views[msg.Target]

	// Push first so contentHeight() uses the new view's chrome.
	entry := nav.StackEntry{View: msg.Target, Context: msg.Context}
	if msg.ResetStack {
		m.stack.Reset(entry)
	} else {
		m.stack.Push(entry)
	}

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
		// Re-size the view we are returning to so it uses the correct contentHeight.
		m.views[current.View].SetSize(m.width, m.contentHeight())

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
