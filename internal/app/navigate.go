package app

import (
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/bschimke95/jara/internal/nav"
	"github.com/bschimke95/jara/internal/view"
)

func (m Model) handleNavigate(msg view.NavigateMsg) (Model, tea.Cmd) {
	target := m.views[msg.Target]
	if target == nil {
		return m, m.showToast(fmt.Sprintf("unknown view: %v", msg.Target))
	}

	// Clear any leftover filter from the previous view.
	m.filterStr = ""
	m.applyFilterToActiveView()

	// Snapshot the stack before mutating so we can roll back if Enter fails.
	snap := m.stack.Snapshot()

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
		// Roll back the stack mutation so the UI stays on the previous view.
		m.stack.Restore(snap)
		return m, m.showToast(err.Error())
	}

	if cmd != nil {
		return m, cmd
	}
	return m, nil
}

// switchEntityContext re-enters the current view with a new entity context.
// An empty entity clears the filter (show all).
func (m Model) switchEntityContext(entity string) (Model, tea.Cmd) {
	current := m.stack.Current()
	v := m.views[current.View]
	if v == nil {
		return m, nil
	}

	// Update the stack entry context.
	m.stack.SetCurrentContext(entity)

	v.SetSize(m.width, m.contentHeight())
	if m.status != nil {
		if sr, ok := v.(view.StatusReceiver); ok {
			sr.SetStatus(m.status)
		}
	}

	cmd, err := v.Enter(view.NavigateContext{Context: entity})
	if err != nil {
		return m, m.showToast(err.Error())
	}
	return m, cmd
}

func (m Model) handleBack() (Model, tea.Cmd) {
	prev := m.stack.Current()
	if _, ok := m.stack.Pop(); ok {
		var cmds []tea.Cmd

		if cmd := m.views[prev.View].Leave(); cmd != nil {
			cmds = append(cmds, cmd)
		}

		// Clear any leftover filter from the child view.
		m.filterStr = ""
		m.applyFilterToActiveView()

		current := m.stack.Current()
		// Re-size the view we are returning to so it uses the correct contentHeight.
		m.views[current.View].SetSize(m.width, m.contentHeight())

		cmd, err := m.views[current.View].Enter(view.NavigateContext{Context: current.Context})
		if err != nil {
			// Re-push to undo the pop — back navigation failed.
			m.stack.Push(prev)
			return m, m.showToast(err.Error())
		}
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		return m, tea.Batch(cmds...)
	}
	return m, nil
}
