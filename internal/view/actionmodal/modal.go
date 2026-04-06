// Package actionmodal implements an action selection and execution modal.
// It lists available charm actions for a unit, lets the user select one,
// and displays the result after execution.
package actionmodal

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/ui"
	"github.com/bschimke95/jara/internal/view"
)

// phase tracks the modal's current state.
type phase int

const (
	phaseLoading phase = iota
	phaseSelect
	phaseRunning
	phaseResult
)

// CloseMsg is emitted when the modal should be closed.
type CloseMsg struct{}

// Modal is the action selection and execution overlay.
type Modal struct {
	keys     ui.KeyMap
	styles   *color.Styles
	width    int
	height   int
	unitName string
	appName  string
	phase    phase
	actions  []model.ActionSpec
	cursor   int
	result   *model.ActionResult
	err      error
}

// New creates a new action modal for the given unit.
func New(unitName, appName string, keys ui.KeyMap, styles *color.Styles) *Modal {
	return &Modal{
		keys:     keys,
		styles:   styles,
		unitName: unitName,
		appName:  appName,
		phase:    phaseLoading,
	}
}

// SetSize updates the modal dimensions.
func (m *Modal) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Init returns a command to fetch available actions.
func (m *Modal) Init() tea.Cmd {
	return func() tea.Msg {
		return view.FetchActionsRequestMsg{AppName: m.appName}
	}
}

// Update handles messages for the action modal.
func (m *Modal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case view.FetchActionsResponseMsg:
		if msg.Err != nil {
			m.err = msg.Err
			m.phase = phaseResult
			return m, nil
		}
		m.actions = msg.Actions
		m.phase = phaseSelect
		return m, nil

	case view.RunActionResultMsg:
		if msg.Err != nil {
			m.err = msg.Err
		} else {
			m.result = msg.Result
		}
		m.phase = phaseResult
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *Modal) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch m.phase {
	case phaseSelect:
		switch {
		case key.Matches(msg, m.keys.Back):
			return m, func() tea.Msg { return CloseMsg{} }
		case key.Matches(msg, m.keys.Enter):
			if len(m.actions) > 0 {
				selected := m.actions[m.cursor]
				m.phase = phaseRunning
				return m, func() tea.Msg {
					return view.RunActionRequestMsg{
						UnitName:   m.unitName,
						ActionName: selected.Name,
					}
				}
			}
		case msg.Key().Code == 'j' || msg.Key().Code == tea.KeyDown:
			if m.cursor < len(m.actions)-1 {
				m.cursor++
			}
		case msg.Key().Code == 'k' || msg.Key().Code == tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
		}
	case phaseResult:
		if key.Matches(msg, m.keys.Back) || key.Matches(msg, m.keys.Enter) {
			return m, func() tea.Msg { return CloseMsg{} }
		}
	case phaseLoading, phaseRunning:
		if key.Matches(msg, m.keys.Back) {
			return m, func() tea.Msg { return CloseMsg{} }
		}
	}
	return m, nil
}

// View returns the modal's tea.View.
func (m *Modal) View() tea.View {
	return tea.NewView(m.Render(""))
}

// Render draws the modal overlay on top of the given background.
func (m *Modal) Render(background string) string {
	innerW := m.width * 50 / 100
	if innerW < 40 {
		innerW = 40
	}
	if innerW > 72 {
		innerW = 72
	}
	contentW := innerW - 2

	var title, content string

	switch m.phase {
	case phaseLoading:
		title = " Actions "
		content = lipgloss.NewStyle().Width(contentW).AlignHorizontal(lipgloss.Center).
			Render("Loading actions...")

	case phaseSelect:
		title = fmt.Sprintf(" Actions · %s ", m.unitName)
		if len(m.actions) == 0 {
			content = lipgloss.NewStyle().Width(contentW).AlignHorizontal(lipgloss.Center).
				Render("No actions available for this charm.")
		} else {
			var sb strings.Builder
			for i, a := range m.actions {
				prefix := "  "
				if i == m.cursor {
					prefix = color.ForegroundText(m.styles.HintKeyColor, "▸ ")
				}
				name := a.Name
				if i == m.cursor {
					name = lipgloss.NewStyle().Bold(true).Render(name)
				}
				desc := ""
				if a.Description != "" {
					desc = lipgloss.NewStyle().Foreground(m.styles.Muted).
						Render(" — " + truncate(a.Description, contentW-len(a.Name)-6))
				}
				sb.WriteString(prefix + name + desc + "\n")
			}
			content = sb.String()
		}
		hint := lipgloss.NewStyle().Foreground(m.styles.Muted).Width(contentW).
			AlignHorizontal(lipgloss.Center).Render("[enter] run  [↑/↓] select  [esc] close")
		content += "\n" + hint

	case phaseRunning:
		title = fmt.Sprintf(" Running · %s ", m.actions[m.cursor].Name)
		content = lipgloss.NewStyle().Width(contentW).AlignHorizontal(lipgloss.Center).
			Render(fmt.Sprintf("Running %q on %s...", m.actions[m.cursor].Name, m.unitName))

	case phaseResult:
		if m.err != nil {
			title = " Action Failed "
			content = lipgloss.NewStyle().Width(contentW).Foreground(m.styles.ErrorColor).
				Render(m.err.Error())
		} else if m.result != nil {
			title = fmt.Sprintf(" %s · %s ", m.result.Status, m.result.ID)
			var sb strings.Builder
			fmt.Fprintf(&sb, "Status: %s\n", m.result.Status)
			if m.result.Message != "" {
				fmt.Fprintf(&sb, "Message: %s\n", m.result.Message)
			}
			if len(m.result.Output) > 0 {
				sb.WriteString("\nOutput:\n")
				for k, v := range m.result.Output {
					fmt.Fprintf(&sb, "  %s: %v\n", k, v)
				}
			}
			content = sb.String()
		}
		hint := lipgloss.NewStyle().Foreground(m.styles.Muted).Width(contentW).
			AlignHorizontal(lipgloss.Center).Render("[enter/esc] close")
		content += "\n" + hint
	}

	titleStyle := lipgloss.NewStyle().Foreground(m.styles.Primary).Bold(true)
	box := ui.BorderBoxRawTitle(content, titleStyle.Render(title), innerW+4, m.styles)

	modalH := lipgloss.Height(box)
	x := (m.width - (innerW + 4)) / 2
	y := (m.height - modalH) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	bg := background
	if bg == "" {
		bg = strings.Repeat("\n", m.height)
	}
	bgLayer := lipgloss.NewLayer(bg)
	overlayLayer := lipgloss.NewLayer(box).X(x).Y(y).Z(1)
	return lipgloss.NewCompositor(bgLayer, overlayLayer).Render()
}

func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
