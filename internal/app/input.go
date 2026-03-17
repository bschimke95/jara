package app

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/nav"
	"github.com/bschimke95/jara/internal/ui"
	"github.com/bschimke95/jara/internal/view"
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
	m.suggestions = nil
	m.selectedSuggestion = 0
	return m, m.input.Focus()
}

func (m Model) enterFilterMode() (Model, tea.Cmd) {
	m.mode = modeFilter
	m.input.Prompt = "/"
	m.input.SetValue(m.filterStr)
	m.suggestions = nil
	m.selectedSuggestion = 0
	return m, m.input.Focus()
}

func (m Model) updateInput(msg tea.Msg) (Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch msg.String() {
		case "enter":
			value := m.input.Value()
			// If a suggestion is selected, use its command text.
			if m.mode == modeCommand && len(m.suggestions) > 0 {
				value = m.suggestions[m.selectedSuggestion].Command
			}
			if m.mode == modeCommand {
				m.mode = modeNormal
				m.suggestions = nil
				m.selectedSuggestion = 0
				m.input.Blur()
				return m.executeCommand(value)
			}
			m.filterStr = value
			m.mode = modeNormal
			m.suggestions = nil
			m.selectedSuggestion = 0
			m.input.Blur()
			return m, nil

		case "esc":
			if m.mode == modeFilter {
				m.filterStr = ""
			}
			m.mode = modeNormal
			m.suggestions = nil
			m.selectedSuggestion = 0
			m.input.Blur()
			return m, nil

		case "tab":
			// Auto-complete: fill in the selected suggestion.
			if m.mode == modeCommand && len(m.suggestions) > 0 {
				m.input.SetValue(m.suggestions[m.selectedSuggestion].Command)
				m.suggestions = nav.MatchCommands(m.input.Value())
				m.selectedSuggestion = 0
				return m, nil
			}

		case "down":
			if m.mode == modeCommand && len(m.suggestions) > 0 {
				m.selectedSuggestion = (m.selectedSuggestion + 1) % len(m.suggestions)
				return m, nil
			}

		case "up":
			if m.mode == modeCommand && len(m.suggestions) > 0 {
				m.selectedSuggestion = (m.selectedSuggestion - 1 + len(m.suggestions)) % len(m.suggestions)
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)

	// Update suggestions after every keystroke in command mode.
	if m.mode == modeCommand {
		m.suggestions = nav.MatchCommands(m.input.Value())
		if m.selectedSuggestion >= len(m.suggestions) {
			m.selectedSuggestion = 0
		}
	}

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

// inputBarHeight returns the number of terminal rows occupied by the active
// input bar (command box with suggestions, or filter box). Returns 0 when
// not in an input mode.
func (m Model) inputBarHeight() int {
	switch m.mode {
	case modeCommand:
		// 3 = top border + input line + bottom border, plus one per suggestion.
		return 3 + len(m.suggestions)
	case modeFilter:
		return 3 // bordered filter box
	default:
		return 0
	}
}

// renderInputBar renders the command or filter input box. Returns "" in normal mode.
func (m Model) renderInputBar() string {
	switch m.mode {
	case modeCommand:
		return m.renderCommandBox()
	case modeFilter:
		return m.renderFilterBar()
	default:
		return ""
	}
}

// renderCommandBox renders a bordered command input with inline suggestions.
func (m Model) renderCommandBox() string {
	promptStyle := lipgloss.NewStyle().Foreground(color.Primary).Bold(true)
	valStyle := lipgloss.NewStyle().Foreground(color.Title)
	cursorStyle := lipgloss.NewStyle().Foreground(color.Primary)

	inputLine := promptStyle.Render(m.input.Prompt) +
		valStyle.Render(m.input.Value()) +
		cursorStyle.Render("█")

	var lines []string
	lines = append(lines, inputLine)

	// Render suggestion rows.
	if len(m.suggestions) > 0 {
		selectedStyle := lipgloss.NewStyle().
			Foreground(color.CrumbFg).
			Background(color.Highlight).
			Bold(true)
		normalStyle := lipgloss.NewStyle().Foreground(color.Muted)
		targetStyle := lipgloss.NewStyle().Foreground(color.Secondary)

		for i, s := range m.suggestions {
			label := s.Command
			target := s.Target.String()

			var row string
			if i == m.selectedSuggestion {
				row = selectedStyle.Render(" " + label + " ")
			} else {
				row = normalStyle.Render(" " + label)
			}
			if target != "" {
				row += targetStyle.Render(" → " + target)
			}
			lines = append(lines, row)
		}
	}

	content := strings.Join(lines, "\n")
	titleStyle := lipgloss.NewStyle().Foreground(color.Primary).Bold(true)
	return ui.BorderBoxRawTitle(content, titleStyle.Render(" Command "), m.width)
}

// renderFilterBar renders the filter input as a bordered box.
func (m Model) renderFilterBar() string {
	promptStyle := lipgloss.NewStyle().Foreground(color.Primary).Bold(true)
	valStyle := lipgloss.NewStyle().Foreground(color.Title)
	cursorStyle := lipgloss.NewStyle().Foreground(color.Primary)

	inputLine := promptStyle.Render(m.input.Prompt) +
		valStyle.Render(m.input.Value()) +
		cursorStyle.Render("█")

	titleStyle := lipgloss.NewStyle().Foreground(color.Primary).Bold(true)
	return ui.BorderBoxRawTitle(inputLine, titleStyle.Render(" Filter "), m.width)
}

// handleGlobalKeys processes key presses that are active in normal mode
// regardless of the current view. This is called only when the active view
// did not consume the key (returned nil cmd), so views can override any
// global binding by handling the key themselves.
func (m Model) handleGlobalKeys(msg tea.KeyPressMsg) (Model, tea.Cmd, bool) {
	switch {
	case msg.String() == "ctrl+c":
		return m, tea.Quit, true
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit, true
	case key.Matches(msg, m.keys.Back):
		m2, cmd := m.handleBack()
		return m2, cmd, true
	case key.Matches(msg, m.keys.Command):
		m2, cmd := m.enterCommandMode()
		return m2, cmd, true
	case key.Matches(msg, m.keys.Filter):
		m2, cmd := m.enterFilterMode()
		return m2, cmd, true
	}
	return m, nil, false
}
