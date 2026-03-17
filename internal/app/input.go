package app

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/bschimke95/jara/internal/nav"
	"github.com/bschimke95/jara/internal/view"
	"github.com/bschimke95/jara/internal/view/debuglog"
)

type inputMode int

const (
	modeNormal inputMode = iota
	modeCommand
	modeFilter
)

func (m Model) enterCommandMode() (Model, tea.Cmd) {
	m.mode = modeCommand
	m.input.Prompt = ":"
	m.input.SetValue("")
	return m, m.input.Focus()
}

func (m Model) enterFilterMode() (Model, tea.Cmd) {
	m.mode = modeFilter
	m.input.Prompt = "/"
	m.input.SetValue(m.filterStr)
	return m, m.input.Focus()
}

func (m Model) updateInput(msg tea.Msg) (Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch msg.String() {
		case "enter":
			value := m.input.Value()
			if m.mode == modeCommand {
				m.mode = modeNormal
				m.input.Blur()
				return m.executeCommand(value)
			}
			m.filterStr = value
			m.mode = modeNormal
			m.input.Blur()
			return m, nil

		case "esc":
			if m.mode == modeFilter {
				m.filterStr = ""
			}
			m.mode = modeNormal
			m.input.Blur()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) executeCommand(cmd string) (Model, tea.Cmd) {
	cmd = strings.TrimSpace(strings.ToLower(cmd))
	if cmd == "q" || cmd == "quit" {
		return m, tea.Quit
	}
	if viewID, ok := nav.ResolveCommand(cmd); ok {
		return m.handleNavigate(view.NavigateMsg{Target: viewID})
	}
	return m, nil
}

// handleGlobalKeys processes key presses that are active in normal mode
// regardless of the current view.
func (m Model) handleGlobalKeys(msg tea.KeyPressMsg) (Model, tea.Cmd, bool) {
	// ctrl+c always quits, even when a modal is open.
	if msg.String() == "ctrl+c" {
		return m, tea.Quit, true
	}

	// When the debug-log filter modal or inline search is open it owns all other
	// key events — global bindings (quit via q, back, etc.) must not fire.
	if dl, ok := m.views[m.stack.Current().View].(*debuglog.View); ok {
		if dl.IsModalOpen() || dl.IsSearchActive() {
			return m, nil, false
		}
	}

	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit, true
	case key.Matches(msg, m.keys.Back):
		m2, cmd := m.handleBack()
		return m2, cmd, true
	case key.Matches(msg, m.keys.Command):
		m2, cmd := m.enterCommandMode()
		return m2, cmd, true
	case key.Matches(msg, m.keys.Filter):
		// The debug-log view owns '/' for in-buffer search; let it handle it.
		if m.stack.Current().View == nav.DebugLogView {
			return m, nil, false
		}
		m2, cmd := m.enterFilterMode()
		return m2, cmd, true
	}
	return m, nil, false
}
