// Package helpmodal implements a key-binding help overlay.
package helpmodal

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/ui"
)

// ClosedMsg is emitted when the help modal is dismissed.
type ClosedMsg struct{}

// Modal renders a key-binding help overlay with two sections:
// a view-specific section and a general/navigation section.
type Modal struct {
	keys   ui.KeyMap
	styles *color.Styles
	width  int
	height int

	viewHints []ui.KeyHint
}

// New creates a new help modal.
func New(keys ui.KeyMap, styles *color.Styles) Modal {
	return Modal{keys: keys, styles: styles}
}

// SetViewHints updates the view-specific hints shown in the modal.
func (m *Modal) SetViewHints(hints []ui.KeyHint) {
	m.viewHints = hints
}

// SetSize informs the modal of the available screen dimensions.
func (m *Modal) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Init implements tea.Model.
func (m *Modal) Init() tea.Cmd { return nil }

// Update implements tea.Model. The modal consumes all key presses; ESC or ?
// emits ClosedMsg to dismiss it.
func (m *Modal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	kp, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	if key.Matches(kp, m.keys.Back) || key.Matches(kp, m.keys.Help) {
		return m, func() tea.Msg { return ClosedMsg{} }
	}
	return m, nil
}

// View implements tea.Model.
func (m *Modal) View() tea.View {
	return tea.NewView(m.Render(""))
}

// generalHints returns the global navigation and common key hints.
func (m *Modal) generalHints() []ui.KeyHint {
	bk := func(b key.Binding) string { return b.Help().Key }
	return []ui.KeyHint{
		{Key: bk(m.keys.Up) + "/" + bk(m.keys.Down), Desc: "up/down"},
		{Key: bk(m.keys.PageUp) + "/" + bk(m.keys.PageDown), Desc: "page up/down"},
		{Key: bk(m.keys.Top) + "/" + bk(m.keys.Bottom), Desc: "top/bottom"},
		{Key: bk(m.keys.Back), Desc: "back"},
		{Key: bk(m.keys.Filter), Desc: "filter"},
		{Key: bk(m.keys.Command), Desc: "command"},
		{Key: bk(m.keys.Help), Desc: "help"},
		{Key: bk(m.keys.Quit), Desc: "quit"},
	}
}

// renderSection renders a named section of key hints in a 2-column grid.
func renderSection(title string, hints []ui.KeyHint, contentW int, styles *color.Styles) string {
	titleStyle := lipgloss.NewStyle().Foreground(styles.Primary).Bold(true)
	keyStyle := styles.HintKey
	descStyle := styles.HintDesc

	var lines []string
	lines = append(lines, titleStyle.Render(title))

	if len(hints) == 0 {
		mutedStyle := lipgloss.NewStyle().Foreground(styles.Muted)
		lines = append(lines, mutedStyle.Render("  (none)"))
		return strings.Join(lines, "\n")
	}

	// Render hints in a 2-column grid.
	colW := contentW / 2
	mid := (len(hints) + 1) / 2

	// Determine the max width of left-column entries for alignment.
	var maxCol1W int
	for i := 0; i < mid; i++ {
		h := hints[i]
		rendered := keyStyle.Render("<"+h.Key+">") + descStyle.Render(" "+h.Desc)
		if w := lipgloss.Width(rendered); w > maxCol1W {
			maxCol1W = w
		}
	}
	if maxCol1W > colW {
		maxCol1W = colW
	}

	for i := 0; i < mid; i++ {
		h1 := hints[i]
		col1 := keyStyle.Render("<"+h1.Key+">") + descStyle.Render(" "+h1.Desc)
		pad := maxCol1W - lipgloss.Width(col1)
		if pad < 0 {
			pad = 0
		}
		line := "  " + col1 + strings.Repeat(" ", pad+2)
		if i+mid < len(hints) {
			h2 := hints[i+mid]
			line += keyStyle.Render("<"+h2.Key+">") + descStyle.Render(" "+h2.Desc)
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// renderBox builds the modal box content (without the compositor overlay).
// It is used by Render and also exposed for testing.
func (m *Modal) renderBox() string {
	modalW := m.width * 60 / 100
	if modalW < 64 {
		modalW = 64
	}
	if m.width > 4 && modalW > m.width-4 {
		modalW = m.width - 4
	}

	// Inner content width accounts for the border (2) and padding (2).
	contentW := modalW - 4
	if contentW < 10 {
		contentW = 10
	}

	divider := lipgloss.NewStyle().Foreground(m.styles.Muted).Render(strings.Repeat("─", contentW))

	viewSection := renderSection("View", m.viewHints, contentW, m.styles)
	generalSection := renderSection("General", m.generalHints(), contentW, m.styles)

	hintStyle := lipgloss.NewStyle().
		Foreground(m.styles.Muted).
		Width(contentW).
		AlignHorizontal(lipgloss.Center)
	footer := hintStyle.Render("[esc] or [?] to close")

	content := viewSection + "\n" + divider + "\n" + generalSection + "\n\n" + footer

	titleStyle := lipgloss.NewStyle().Foreground(m.styles.Primary).Bold(true)
	return ui.BorderBoxRawTitle(content, titleStyle.Render(" Key Bindings "), modalW, m.styles)
}

// Render draws the modal as an overlay on the given background string.
func (m *Modal) Render(background string) string {
	box := m.renderBox()

	modalW := lipgloss.Width(box)
	modalH := lipgloss.Height(box)
	x := (m.width - modalW) / 2
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
