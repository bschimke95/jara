// Package helpmodal implements a key-binding help overlay.
package helpmodal

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/ui"
	"github.com/bschimke95/jara/internal/view"
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
	return []ui.KeyHint{
		{Key: view.BindingKey(m.keys.Up) + "/" + view.BindingKey(m.keys.Down), Desc: "up/down"},
		{Key: view.BindingKey(m.keys.PageUp) + "/" + view.BindingKey(m.keys.PageDown), Desc: "page up/down"},
		{Key: view.BindingKey(m.keys.Top) + "/" + view.BindingKey(m.keys.Bottom), Desc: "top/bottom"},
		{Key: view.BindingKey(m.keys.Back), Desc: "back"},
		{Key: view.BindingKey(m.keys.Filter), Desc: "filter"},
		{Key: view.BindingKey(m.keys.Command), Desc: "cmd"},
		{Key: view.BindingKey(m.keys.Help), Desc: "help"},
		{Key: view.BindingKey(m.keys.Quit), Desc: "quit"},
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

	// Render hints in a 2-column grid, with keys and descriptions aligned
	// per column so descriptions line up vertically within each column.
	// Cap the left column at ui.MaxHintsPerColumn rows; overflow spills right.
	mid := min(len(hints), ui.MaxHintsPerColumn)

	// Compute max key width separately for left and right columns.
	var maxKeyWLeft, maxKeyWRight int
	for i := 0; i < mid; i++ {
		if w := lipgloss.Width(keyStyle.Render("<" + hints[i].Key + ">")); w > maxKeyWLeft {
			maxKeyWLeft = w
		}
	}
	for i := mid; i < len(hints); i++ {
		if w := lipgloss.Width(keyStyle.Render("<" + hints[i].Key + ">")); w > maxKeyWRight {
			maxKeyWRight = w
		}
	}

	// renderHintCol returns "<key><pad> desc" with the key padded to maxKeyW.
	renderHintCol := func(h ui.KeyHint, maxKeyW int) string {
		k := keyStyle.Render("<" + h.Key + ">")
		pad := maxKeyW - lipgloss.Width(k)
		if pad < 0 {
			pad = 0
		}
		return k + strings.Repeat(" ", pad) + descStyle.Render(" "+h.Desc)
	}

	// Determine the max rendered width of left-column entries for inter-column gap.
	var maxCol1W int
	for i := 0; i < mid; i++ {
		if w := lipgloss.Width(renderHintCol(hints[i], maxKeyWLeft)); w > maxCol1W {
			maxCol1W = w
		}
	}

	// Build each row and measure the total grid width for centering.
	type row struct{ left, right string }
	rows := make([]row, mid)
	for i := 0; i < mid; i++ {
		left := renderHintCol(hints[i], maxKeyWLeft)
		pad := maxCol1W - lipgloss.Width(left)
		if pad < 0 {
			pad = 0
		}
		left += strings.Repeat(" ", pad)
		var right string
		if i+mid < len(hints) {
			right = renderHintCol(hints[i+mid], maxKeyWRight)
		}
		rows[i] = row{left, right}
	}

	// Compute grid width for centering within contentW.
	gridW := maxCol1W + 2 // left col + gap
	for i := mid; i < len(hints); i++ {
		if w := lipgloss.Width(renderHintCol(hints[i], maxKeyWRight)); w > gridW-maxCol1W-2 {
			gridW = maxCol1W + 2 + lipgloss.Width(renderHintCol(hints[i], maxKeyWRight))
		}
	}
	leftPad := (contentW - gridW) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	indent := strings.Repeat(" ", leftPad)

	for _, r := range rows {
		line := indent + r.left + "  " + r.right
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// viewNavHints returns the view navigation shortcuts.
func (m *Modal) viewNavHints() []ui.KeyHint {
	return []ui.KeyHint{
		{Key: view.BindingKey(m.keys.ApplicationsNav), Desc: "Applications"},
		{Key: view.BindingKey(m.keys.UnitsNav), Desc: "Units"},
		{Key: view.BindingKey(m.keys.RelationsNav), Desc: "Relations"},
		{Key: view.BindingKey(m.keys.MachinesNav), Desc: "Machines"},
		{Key: view.BindingKey(m.keys.SecretsNav), Desc: "Secrets"},
		{Key: view.BindingKey(m.keys.OffersNav), Desc: "Offers"},
		{Key: view.BindingKey(m.keys.StorageNav), Desc: "Storage"},
		{Key: view.BindingKey(m.keys.ConfigNav), Desc: "Config"},
		{Key: view.BindingKey(m.keys.ChatNav), Desc: "Chat"},
		{Key: view.BindingKey(m.keys.LogsView), Desc: "Debug Log"},
	}
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
	viewsSection := renderSection("Views", m.viewNavHints(), contentW, m.styles)
	generalSection := renderSection("General", m.generalHints(), contentW, m.styles)

	hintStyle := lipgloss.NewStyle().
		Foreground(m.styles.Muted).
		Width(contentW).
		AlignHorizontal(lipgloss.Center)
	footer := hintStyle.Render("[esc] or [?] to close")

	content := viewSection + "\n" + divider + "\n" + viewsSection + "\n" + divider + "\n" + generalSection + "\n\n" + footer

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
