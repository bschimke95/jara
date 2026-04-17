// Package switchmodal implements a modal overlay for switching the active
// entity context (e.g. selecting which application to filter by).
package switchmodal

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/ui"
)

// SelectedMsg is emitted when the user confirms an entity.
type SelectedMsg struct {
	Entity string // empty string means "show all" (clear filter)
}

// PreviewMsg is emitted when the cursor moves to preview a different entity.
type PreviewMsg struct {
	Entity string
}

// ClosedMsg is emitted when the user dismisses the modal without choosing.
type ClosedMsg struct {
	Original string // the entity that was active when the modal opened
}

// Modal is a scrollable list overlay for picking an entity.
type Modal struct {
	keys   ui.KeyMap
	styles *color.Styles
	width  int
	height int

	title    string
	items    []string // first entry is always "" (show all)
	cursor   int
	original string // entity active when the modal opened
}

// New creates a new switch modal. entities should be the available names
// (sorted). current is the currently active entity (empty = show all).
// title is shown in the modal border (e.g. "Switch Application").
func New(title string, entities []string, current string, keys ui.KeyMap, styles *color.Styles) *Modal {
	items := make([]string, 0, len(entities)+1)
	items = append(items, "") // show all
	items = append(items, entities...)

	cursor := 0
	for i, e := range items {
		if e == current {
			cursor = i
			break
		}
	}

	return &Modal{
		keys:     keys,
		styles:   styles,
		title:    title,
		items:    items,
		cursor:   cursor,
		original: current,
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
		return m, func() tea.Msg { return SelectedMsg{Entity: m.items[m.cursor]} }
	case key.Matches(kp, m.keys.Back):
		return m, func() tea.Msg { return ClosedMsg{Original: m.original} }
	case key.Matches(kp, m.keys.Down):
		if m.cursor < len(m.items)-1 {
			m.cursor++
			entity := m.items[m.cursor]
			return m, func() tea.Msg { return PreviewMsg{Entity: entity} }
		}
	case key.Matches(kp, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
			entity := m.items[m.cursor]
			return m, func() tea.Msg { return PreviewMsg{Entity: entity} }
		}
	}

	return m, nil
}

// Render draws the switch modal over the given background.
func (m *Modal) Render(background string) string {
	innerW := m.width * 40 / 100
	if innerW < 30 {
		innerW = 30
	}
	if innerW > 50 {
		innerW = 50
	}
	contentW := innerW - 2

	maxVisible := 10
	if maxVisible > len(m.items) {
		maxVisible = len(m.items)
	}
	start := m.cursor - maxVisible/2
	if start < 0 {
		start = 0
	}
	if start+maxVisible > len(m.items) {
		start = len(m.items) - maxVisible
	}

	selectedStyle := lipgloss.NewStyle().
		Foreground(m.styles.CrumbFgColor).
		Background(m.styles.Highlight).
		Bold(true).
		Width(contentW)
	normalStyle := lipgloss.NewStyle().
		Foreground(m.styles.Title).
		Width(contentW)
	activeMarker := lipgloss.NewStyle().
		Foreground(m.styles.Primary).
		Bold(true)

	var lines []string
	for i := start; i < start+maxVisible; i++ {
		label := m.items[i]
		if label == "" {
			label = "(show all)"
		}

		suffix := ""
		if m.items[i] == m.original {
			suffix = activeMarker.Render(" *")
		}

		if i == m.cursor {
			lines = append(lines, selectedStyle.Render(" "+label)+suffix)
		} else {
			lines = append(lines, normalStyle.Render(" "+label)+suffix)
		}
	}

	content := strings.Join(lines, "\n")

	hintStyle := lipgloss.NewStyle().Foreground(m.styles.Muted).Width(contentW).AlignHorizontal(lipgloss.Center)
	content += "\n" + hintStyle.Render("[enter] select  [esc] cancel")

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
