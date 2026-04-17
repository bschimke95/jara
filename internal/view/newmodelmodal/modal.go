// Package newmodelmodal implements a modal overlay for creating a new model.
package newmodelmodal

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/ui"
)

// AppliedMsg is emitted when the user confirms model creation.
type AppliedMsg struct {
	Name string
}

// ClosedMsg is emitted when the modal is cancelled.
type ClosedMsg struct{}

// Modal is a simple overlay for entering a new model name.
type Modal struct {
	keys   ui.KeyMap
	styles *color.Styles
	width  int
	height int

	input         textinput.Model
	validationErr string
}

// New creates a new model creation modal.
func New(keys ui.KeyMap, styles *color.Styles) Modal {
	ti := textinput.New()
	ti.CharLimit = 64
	ti.Placeholder = "model name"
	return Modal{
		keys:   keys,
		styles: styles,
		input:  ti,
	}
}

func (m *Modal) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *Modal) Init() tea.Cmd {
	return m.input.Focus()
}

func (m *Modal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.Enter):
			name := strings.TrimSpace(m.input.Value())
			if name == "" {
				m.validationErr = "model name is required"
				return m, nil
			}
			return m, func() tea.Msg { return AppliedMsg{Name: name} }
		case key.Matches(msg, m.keys.Back):
			return m, func() tea.Msg { return ClosedMsg{} }
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.validationErr = ""
	return m, cmd
}

func (m *Modal) View() tea.View {
	return tea.NewView(m.Render(""))
}

// Render draws the modal over the given background.
func (m *Modal) Render(background string) string {
	innerW := m.width * 40 / 100
	if innerW < 36 {
		innerW = 36
	}
	if innerW > 60 {
		innerW = 60
	}

	contentW := innerW - 2
	labelStyle := lipgloss.NewStyle().Foreground(m.styles.Muted)
	content := labelStyle.Render("Name") + "\n" + m.input.View()

	if m.validationErr != "" {
		errStyle := lipgloss.NewStyle().Foreground(m.styles.ErrorColor)
		content += "\n" + errStyle.Render(m.validationErr)
	}

	hintStyle := lipgloss.NewStyle().
		Foreground(m.styles.Muted).
		Width(contentW).
		AlignHorizontal(lipgloss.Center)
	content += "\n\n" + hintStyle.Render("[enter] create  [esc] cancel")

	titleStyle := lipgloss.NewStyle().Foreground(m.styles.Primary).Bold(true)
	box := ui.BorderBoxRawTitle(content, titleStyle.Render(" New Model "), innerW+4, m.styles)

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
