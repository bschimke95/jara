// Package infomodal implements a read-only detail overlay that displays
// the full untruncated fields for a selected table row.
package infomodal

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/ui"
)

// ClosedMsg is emitted when the info modal is dismissed.
type ClosedMsg struct{}

// Field is a single label-value pair displayed in the modal.
type Field struct {
	Label string
	Value string
}

// Data carries the title and fields to display.
type Data struct {
	Title  string
	Fields []Field
}

// Modal is the info detail overlay.
type Modal struct {
	keys   ui.KeyMap
	styles *color.Styles
	width  int
	height int
	data   Data
	scroll int // vertical scroll offset
}

// New creates a new info modal.
func New(data Data, keys ui.KeyMap, styles *color.Styles) *Modal {
	return &Modal{
		keys:   keys,
		styles: styles,
		data:   data,
	}
}

// SetSize updates the modal dimensions.
func (m *Modal) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Init implements tea.Model.
func (m *Modal) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m *Modal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	kp, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	switch {
	case key.Matches(kp, m.keys.Back), key.Matches(kp, m.keys.Inspect):
		return m, func() tea.Msg { return ClosedMsg{} }
	case key.Matches(kp, m.keys.Down):
		m.scroll++
	case key.Matches(kp, m.keys.Up):
		if m.scroll > 0 {
			m.scroll--
		}
	}
	return m, nil
}

// View implements tea.Model.
func (m *Modal) View() tea.View {
	return tea.NewView(m.Render(""))
}

// Render draws the modal overlay on top of the given background.
func (m *Modal) Render(background string) string {
	innerW := m.width * 55 / 100
	if innerW < 50 {
		innerW = 50
	}
	if innerW > 80 {
		innerW = 80
	}
	contentW := innerW - 4 // border (2) + padding (2)
	if contentW < 10 {
		contentW = 10
	}

	labelStyle := lipgloss.NewStyle().Foreground(m.styles.Secondary).Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(m.styles.Muted)

	var lines []string
	for _, f := range m.data.Fields {
		lines = append(lines, labelStyle.Render(f.Label))
		// Word-wrap the value to contentW.
		wrapped := wordWrap(f.Value, contentW-2)
		for _, wl := range strings.Split(wrapped, "\n") {
			lines = append(lines, "  "+wl)
		}
		lines = append(lines, "")
	}

	// Apply scroll.
	if m.scroll > len(lines) {
		m.scroll = len(lines)
	}
	if m.scroll > 0 && m.scroll < len(lines) {
		lines = lines[m.scroll:]
	}

	// Cap visible height to leave room for chrome.
	maxH := m.height*70/100 - 4
	if maxH < 5 {
		maxH = 5
	}
	if len(lines) > maxH {
		lines = lines[:maxH]
	}

	hint := mutedStyle.Render("[esc/i] close  [↑/↓] scroll")
	content := strings.Join(lines, "\n") + "\n\n" +
		lipgloss.NewStyle().Width(contentW).AlignHorizontal(lipgloss.Center).Render(hint)

	titleStyle := lipgloss.NewStyle().Foreground(m.styles.Primary).Bold(true)
	box := ui.BorderBoxRawTitle(content, titleStyle.Render(" "+m.data.Title+" "), innerW, m.styles)

	modalH := lipgloss.Height(box)
	x := (m.width - innerW) / 2
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

// wordWrap breaks s into lines of at most width characters, splitting on
// whitespace boundaries where possible.
func wordWrap(s string, width int) string {
	if width <= 0 {
		return s
	}
	var out strings.Builder
	for _, line := range strings.Split(s, "\n") {
		if out.Len() > 0 {
			out.WriteByte('\n')
		}
		col := 0
		for i, word := range strings.Fields(line) {
			wl := len(word)
			if i > 0 && col+1+wl > width {
				out.WriteByte('\n')
				col = 0
			} else if i > 0 {
				out.WriteByte(' ')
				col++
			}
			out.WriteString(word)
			col += wl
		}
	}
	return out.String()
}
