// Package confirmodal implements a generic confirmation modal overlay.
package confirmodal

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/ui"
)

// ConfirmedMsg is emitted when the user confirms the action.
type ConfirmedMsg struct{}

// CancelledMsg is emitted when the user cancels the action.
type CancelledMsg struct{}

// Modal is a simple yes/no confirmation overlay.
type Modal struct {
	keys   ui.KeyMap
	width  int
	height int

	title   string
	message string
}

// New creates a new confirmation modal.
func New(keys ui.KeyMap, title, message string) Modal {
	return Modal{
		keys:    keys,
		title:   title,
		message: message,
	}
}

func (m *Modal) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *Modal) Init() tea.Cmd { return nil }

func (m *Modal) View() tea.View {
	return tea.NewView(m.Render(""))
}

func (m *Modal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	kp, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch {
	case key.Matches(kp, m.keys.Enter):
		return m, func() tea.Msg { return ConfirmedMsg{} }
	case key.Matches(kp, m.keys.Back):
		return m, func() tea.Msg { return CancelledMsg{} }
	}

	return m, nil
}

// Render draws the confirmation modal over the given background.
func (m *Modal) Render(background string) string {
	innerW := m.width * 40 / 100
	if innerW < 36 {
		innerW = 36
	}
	if innerW > 60 {
		innerW = 60
	}

	contentW := innerW - 2
	msgStyle := lipgloss.NewStyle().Width(contentW).AlignHorizontal(lipgloss.Center)
	content := msgStyle.Render(m.message)

	hintStyle := lipgloss.NewStyle().
		Foreground(color.Muted).
		Width(contentW).
		AlignHorizontal(lipgloss.Center)
	content += "\n\n" + hintStyle.Render("[enter] confirm  [esc] cancel")

	titleStyle := lipgloss.NewStyle().Foreground(color.Error).Bold(true)
	box := ui.BorderBoxRawTitle(content, titleStyle.Render(" "+m.title+" "), innerW+4)

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
