// Package revealmodal implements a read-only modal overlay that displays
// decoded secret key-value content.
package revealmodal

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/ui"
)

// ClosedMsg is emitted when the user dismisses the reveal modal.
type ClosedMsg struct{}

// Modal is a read-only overlay that shows secret key-value pairs.
type Modal struct {
	keys   ui.KeyMap
	styles *color.Styles
	width  int
	height int

	title  string
	values map[string]string

	scroll int // vertical scroll offset
	lines  []string
}

// New creates a new reveal modal.
func New(keys ui.KeyMap, styles *color.Styles, title string, values map[string]string) Modal {
	m := Modal{
		keys:   keys,
		styles: styles,
		title:  title,
		values: values,
	}
	m.buildLines()
	return m
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
	case key.Matches(kp, m.keys.Back):
		return m, func() tea.Msg { return ClosedMsg{} }
	case key.Matches(kp, m.keys.Down):
		if m.scroll < len(m.lines)-1 {
			m.scroll++
		}
	case key.Matches(kp, m.keys.Up):
		if m.scroll > 0 {
			m.scroll--
		}
	}

	return m, nil
}

// Render draws the reveal modal over the given background.
func (m *Modal) Render(background string) string {
	innerW := m.width * 50 / 100
	if innerW < 40 {
		innerW = 40
	}
	if innerW > 80 {
		innerW = 80
	}

	contentW := innerW - 2
	if contentW < 10 {
		contentW = 10
	}

	// Build visible content with scrolling.
	maxVisibleLines := m.height/2 - 4
	if maxVisibleLines < 5 {
		maxVisibleLines = 5
	}

	visible := m.lines
	if len(visible) > maxVisibleLines {
		end := m.scroll + maxVisibleLines
		if end > len(visible) {
			end = len(visible)
			m.scroll = end - maxVisibleLines
		}
		if m.scroll < 0 {
			m.scroll = 0
		}
		visible = visible[m.scroll:end]
	}

	keyStyle := lipgloss.NewStyle().Foreground(m.styles.Title).Bold(true)
	valStyle := lipgloss.NewStyle().Foreground(m.styles.InfoValueColor)

	var rendered []string
	for _, line := range visible {
		if idx := strings.Index(line, ": "); idx >= 0 {
			k := line[:idx]
			v := line[idx+2:]
			rendered = append(rendered, keyStyle.Render(k)+": "+valStyle.Render(v))
		} else {
			rendered = append(rendered, line)
		}
	}

	content := strings.Join(rendered, "\n")

	hintStyle := lipgloss.NewStyle().
		Foreground(m.styles.Muted).
		Width(contentW).
		AlignHorizontal(lipgloss.Center)
	scrollHint := ""
	if len(m.lines) > maxVisibleLines {
		scrollHint = fmt.Sprintf(" (%d/%d)", m.scroll+1, len(m.lines)-maxVisibleLines+1)
	}
	content += "\n\n" + hintStyle.Render("[j/k] scroll  [esc] close"+scrollHint)

	titleStyle := lipgloss.NewStyle().Foreground(m.styles.BorderTitleColor).Bold(true)
	box := ui.BorderBoxRawTitle(content, titleStyle.Render(" "+m.title+" "), innerW+4, m.styles)

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

// buildLines prepares the sorted key-value display lines.
func (m *Modal) buildLines() {
	keys := make([]string, 0, len(m.values))
	for k := range m.values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	m.lines = make([]string, 0, len(keys))
	for _, k := range keys {
		m.lines = append(m.lines, k+": "+m.values[k])
	}
}
