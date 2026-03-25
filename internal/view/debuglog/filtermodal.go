package debuglog

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/ui"
)

// logLevels are the valid Juju debug-log severity levels.
var logLevels = []string{"TRACE", "DEBUG", "INFO", "WARNING", "ERROR"}

// leftPane identifies a row in the left navigation pane.
type leftPane int

const (
	leftPaneLevel leftPane = iota
	leftPaneApplications
	leftPaneUnits
	leftPaneMachines
	leftPaneModules
	leftPaneLabels
	leftPaneCount
)

var leftPaneNames = []string{"Level", "Applications", "Units", "Machines", "Modules", "Labels"}

type paneFocus int

const (
	focusLeft paneFocus = iota
	focusRight
)

// FilterAppliedMsg is emitted by the filter modal when the user confirms.
type FilterAppliedMsg struct {
	Filter model.DebugLogFilter
}

// FilterModalClosedMsg is emitted when the user cancels the modal without applying.
type FilterModalClosedMsg struct{}

// FilterModal is a two-pane vim-navigable overlay for configuring debug-log filters.
type FilterModal struct {
	keys   ui.KeyMap
	styles *color.Styles
	filter model.DebugLogFilter

	suggestions map[leftPane][]string

	leftCursor      leftPane
	rightCursor     int
	rightViewOffset int
	focus           paneFocus

	adding    bool
	textInput textinput.Model

	width  int
	height int
}

const rightPaneVisibleRows = 12

// NewFilterModal creates a modal pre-populated with the given filter defaults.
func NewFilterModal(initial model.DebugLogFilter, suggestions map[leftPane][]string, keys ui.KeyMap, styles *color.Styles) FilterModal {
	ti := textinput.New()
	ti.CharLimit = 256
	ti.Placeholder = "key=value, Enter to add"

	if suggestions == nil {
		suggestions = make(map[leftPane][]string)
	}

	return FilterModal{
		keys:        keys,
		styles:      styles,
		filter:      initial,
		suggestions: suggestions,
		textInput:   ti,
	}
}

func (m *FilterModal) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *FilterModal) Init() tea.Cmd { return nil }

func (m *FilterModal) View() tea.View {
	return tea.NewView(m.Render(""))
}

func (m *FilterModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	kp, isKey := msg.(tea.KeyPressMsg)

	if m.adding {
		if isKey {
			switch kp.String() {
			case "enter":
				v := strings.TrimSpace(m.textInput.Value())
				if v != "" {
					m.addLabelEntry(v)
				}
				m.adding = false
				m.textInput.Blur()
				return m, nil
			case "esc":
				m.adding = false
				m.textInput.Blur()
				return m, nil
			}
		}
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	if !isKey {
		return m, nil
	}

	switch {
	case key.Matches(kp, m.keys.ApplyFilter):
		return m, func() tea.Msg { return FilterAppliedMsg{Filter: m.filter} }
	case key.Matches(kp, m.keys.Back):
		if m.focus == focusRight {
			m.focus = focusLeft
			return m, nil
		}
		return m, func() tea.Msg { return FilterModalClosedMsg{} }
	}

	if m.focus == focusLeft {
		return m.updateLeftPane(kp)
	}
	return m.updateRightPane(kp)
}

func (m *FilterModal) updateLeftPane(kp tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(kp, m.keys.Down):
		m.leftCursor = (m.leftCursor + 1) % leftPaneCount
		m.rightCursor = 0
		m.rightViewOffset = 0
	case key.Matches(kp, m.keys.Up):
		if m.leftCursor == 0 {
			m.leftCursor = leftPaneCount - 1
		} else {
			m.leftCursor--
		}
		m.rightCursor = 0
		m.rightViewOffset = 0
	case key.Matches(kp, m.keys.Enter, m.keys.Right):
		m.focus = focusRight
		m.rightCursor = 0
		m.rightViewOffset = 0
	}
	return m, nil
}

type rowKind int

const (
	rowSelected rowKind = iota
	rowAddLabel
	rowDivider
	rowSuggestion
)

type rightRow struct {
	kind  rowKind
	label string
}

func (m *FilterModal) buildRightRows() []rightRow {
	switch m.leftCursor {
	case leftPaneLevel:
		rows := make([]rightRow, len(logLevels))
		for i, lv := range logLevels {
			rows[i] = rightRow{kind: rowSelected, label: lv}
		}
		return rows

	default:
		var rows []rightRow
		selected := m.selectedItems()
		for _, s := range selected {
			rows = append(rows, rightRow{kind: rowSelected, label: s})
		}
		if m.leftCursor == leftPaneLabels {
			rows = append(rows, rightRow{kind: rowAddLabel, label: "[ + add label ]"})
		}
		sugg := m.availableSuggestions()
		if len(sugg) > 0 {
			if len(selected) > 0 || m.leftCursor == leftPaneLabels {
				rows = append(rows, rightRow{kind: rowDivider})
			}
			for _, s := range sugg {
				rows = append(rows, rightRow{kind: rowSuggestion, label: s})
			}
		}
		return rows
	}
}

func (m *FilterModal) updateRightPane(kp tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	rows := m.buildRightRows()
	total := len(rows)

	switch {
	case key.Matches(kp, m.keys.Down):
		if total > 0 {
			next := (m.rightCursor + 1) % total
			for next < total && rows[next].kind == rowDivider {
				next = (next + 1) % total
			}
			m.rightCursor = next
			m.scrollRightIntoView(total)
		}
	case key.Matches(kp, m.keys.Up):
		if total > 0 {
			prev := m.rightCursor - 1
			if prev < 0 {
				prev = total - 1
			}
			for prev >= 0 && rows[prev].kind == rowDivider {
				prev--
				if prev < 0 {
					prev = total - 1
				}
			}
			m.rightCursor = prev
			m.scrollRightIntoView(total)
		}
	case key.Matches(kp, m.keys.Enter):
		if m.rightCursor < len(rows) {
			row := rows[m.rightCursor]
			switch row.kind {
			case rowSelected:
				if m.leftCursor == leftPaneLevel {
					if m.filter.Level == row.label {
						m.filter.Level = ""
					} else {
						m.filter.Level = row.label
					}
				} else {
					m.removeSelectedItem(row.label)
					newRows := m.buildRightRows()
					if m.rightCursor >= len(newRows) && m.rightCursor > 0 {
						m.rightCursor = len(newRows) - 1
					}
					m.scrollRightIntoView(len(newRows))
				}
			case rowAddLabel:
				m.textInput.SetValue("")
				m.adding = true
				return m, m.textInput.Focus()
			case rowSuggestion:
				m.addSelectedItem(row.label)
				newRows := m.buildRightRows()
				if m.rightCursor >= len(newRows) && m.rightCursor > 0 {
					m.rightCursor = len(newRows) - 1
				}
				m.scrollRightIntoView(len(newRows))
			}
		}
	case key.Matches(kp, m.keys.Left):
		m.focus = focusLeft
	}
	return m, nil
}

func (m *FilterModal) scrollRightIntoView(total int) {
	if m.rightCursor < m.rightViewOffset {
		m.rightViewOffset = m.rightCursor
	}
	if m.rightCursor >= m.rightViewOffset+rightPaneVisibleRows {
		m.rightViewOffset = m.rightCursor - rightPaneVisibleRows + 1
	}
	maxOffset := total - rightPaneVisibleRows
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.rightViewOffset > maxOffset {
		m.rightViewOffset = maxOffset
	}
}

func (m *FilterModal) selectedItems() []string {
	switch m.leftCursor {
	case leftPaneApplications:
		return append([]string(nil), m.filter.Applications...)
	case leftPaneUnits:
		var out []string
		for _, e := range m.filter.IncludeEntities {
			if !strings.HasPrefix(e, "machine-") {
				out = append(out, e)
			}
		}
		return out
	case leftPaneMachines:
		var out []string
		for _, e := range m.filter.IncludeEntities {
			if strings.HasPrefix(e, "machine-") {
				out = append(out, e)
			}
		}
		return out
	case leftPaneModules:
		return append([]string(nil), m.filter.IncludeModules...)
	case leftPaneLabels:
		var out []string
		for k, v := range m.filter.IncludeLabels {
			out = append(out, k+"="+v)
		}
		sort.Strings(out)
		return out
	}
	return nil
}

func (m *FilterModal) availableSuggestions() []string {
	raw := m.suggestions[m.leftCursor]
	if len(raw) == 0 {
		return nil
	}
	selected := make(map[string]bool)
	for _, s := range m.selectedItems() {
		selected[s] = true
	}
	var out []string
	for _, s := range raw {
		if !selected[s] {
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

func (m *FilterModal) addSelectedItem(v string) {
	switch m.leftCursor {
	case leftPaneApplications:
		m.filter.Applications = appendUnique(m.filter.Applications, v)
	case leftPaneUnits:
		m.filter.IncludeEntities = appendUnique(m.filter.IncludeEntities, v)
	case leftPaneMachines:
		if !strings.HasPrefix(v, "machine-") {
			v = "machine-" + v
		}
		m.filter.IncludeEntities = appendUnique(m.filter.IncludeEntities, v)
	case leftPaneModules:
		m.filter.IncludeModules = appendUnique(m.filter.IncludeModules, v)
	}
}

func (m *FilterModal) removeSelectedItem(v string) {
	switch m.leftCursor {
	case leftPaneApplications:
		m.filter.Applications = removeFromSlice(m.filter.Applications, v)
	case leftPaneUnits, leftPaneMachines:
		m.filter.IncludeEntities = removeFromSlice(m.filter.IncludeEntities, v)
	case leftPaneModules:
		m.filter.IncludeModules = removeFromSlice(m.filter.IncludeModules, v)
	case leftPaneLabels:
		k, _, _ := strings.Cut(v, "=")
		delete(m.filter.IncludeLabels, strings.TrimSpace(k))
	}
}

func (m *FilterModal) addLabelEntry(v string) {
	k, val, _ := strings.Cut(v, "=")
	k = strings.TrimSpace(k)
	val = strings.TrimSpace(val)
	if k == "" {
		return
	}
	if m.filter.IncludeLabels == nil {
		m.filter.IncludeLabels = make(map[string]string)
	}
	m.filter.IncludeLabels[k] = val
}

// Render returns the composed string for the modal overlay using lipgloss layers.
func (m *FilterModal) Render(background string) string {
	leftW := 18
	rightW := m.width * 35 / 100
	if rightW < 28 {
		rightW = 28
	}
	if rightW > 48 {
		rightW = 48
	}
	innerW := leftW + 2 + rightW + 2
	outerW := innerW + 4

	leftContent, rightContent := m.renderPanes(leftW, rightW)
	leftBox := ui.BorderBox(leftContent, "Filters", leftW+2, m.styles)
	rightBox := ui.BorderBox(rightContent, leftPaneNames[m.leftCursor], rightW+2, m.styles)
	combined := lipgloss.JoinHorizontal(lipgloss.Top, leftBox, rightBox)
	combined += "\n" + m.renderFooter()

	titleStyle := lipgloss.NewStyle().Foreground(m.styles.Primary).Bold(true)
	box := ui.BorderBoxRawTitle(combined, titleStyle.Render(" Debug Log Filter "), outerW, m.styles)

	modalH := lipgloss.Height(box)
	x := (m.width - outerW) / 2
	y := (m.height - modalH) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	bgLayer := lipgloss.NewLayer(background)
	overlayLayer := lipgloss.NewLayer(box).X(x).Y(y).Z(1)
	return lipgloss.NewCompositor(bgLayer, overlayLayer).Render()
}

func (m *FilterModal) renderPanes(leftW, rightW int) (string, string) {
	return m.renderLeftPane(leftW), m.renderRightPane(rightW)
}

func (m *FilterModal) renderLeftPane(w int) string {
	cursorStyle := lipgloss.NewStyle().Foreground(m.styles.Primary).Bold(true)
	activeHL := lipgloss.NewStyle().
		Foreground(m.styles.CrumbFgColor).
		Background(m.styles.Highlight).
		Bold(true)
	cursorHL := lipgloss.NewStyle().
		Foreground(m.styles.Primary).
		Background(m.styles.Highlight).
		Bold(true)
	normalStyle := lipgloss.NewStyle().Foreground(m.styles.InfoLabelColor)

	var b strings.Builder
	for i := leftPane(0); i < leftPaneCount; i++ {
		padded := fmt.Sprintf("%-*s", w, leftPaneNames[i])
		var line string
		switch {
		case i == m.leftCursor && m.focus == focusLeft:
			line = cursorStyle.Render("▶ ") + cursorHL.Render(padded)
		case i == m.leftCursor:
			line = cursorStyle.Render("▶ ") + activeHL.Render(padded)
		default:
			line = "  " + normalStyle.Render(padded)
		}
		b.WriteString(line + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func truncateLabel(s string, maxLen int) string {
	stripped := ansi.Strip(s)
	if len([]rune(stripped)) <= maxLen {
		return s
	}
	runes := []rune(stripped)
	if maxLen < 1 {
		maxLen = 1
	}
	return string(runes[:maxLen-1]) + "…"
}

func (m *FilterModal) renderRightPane(w int) string {
	checkedStyle := lipgloss.NewStyle().Foreground(m.styles.CheckGreenColor).Bold(true)
	uncheckedStyle := lipgloss.NewStyle().Foreground(m.styles.CheckRedColor)
	cursorStyle := lipgloss.NewStyle().Foreground(m.styles.Primary).Bold(true)
	activeStyle := lipgloss.NewStyle().
		Foreground(m.styles.CrumbFgColor).
		Background(m.styles.Highlight).
		Bold(true)
	normalStyle := lipgloss.NewStyle().Foreground(m.styles.InfoValueColor)
	suggStyle := lipgloss.NewStyle().Foreground(m.styles.Muted)
	addStyle := lipgloss.NewStyle().Foreground(m.styles.Muted).Italic(true)
	inputStyle := lipgloss.NewStyle().Foreground(m.styles.Title)
	dimStyle := lipgloss.NewStyle().Foreground(m.styles.Muted)

	maxLabel := w - 4
	if maxLabel < 4 {
		maxLabel = 4
	}

	rows := m.buildRightRows()

	if m.adding {
		var b strings.Builder
		for _, row := range rows {
			if row.kind == rowSelected || row.kind == rowSuggestion {
				label := truncateLabel(row.label, maxLabel)
				b.WriteString("  " + checkedStyle.Render("✔ ") + normalStyle.Render(label) + "\n")
			}
		}
		b.WriteString("  " + checkedStyle.Render("+ ") + inputStyle.Render(m.textInput.View()) + "\n")
		return strings.TrimRight(b.String(), "\n")
	}

	total := len(rows)

	maxOffset := total - rightPaneVisibleRows
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.rightViewOffset > maxOffset {
		m.rightViewOffset = maxOffset
	}
	if m.rightViewOffset < 0 {
		m.rightViewOffset = 0
	}

	visStart := m.rightViewOffset
	visEnd := visStart + rightPaneVisibleRows
	if visEnd > total {
		visEnd = total
	}

	var b strings.Builder

	for i := visStart; i < visEnd; i++ {
		row := rows[i]
		isCursor := m.focus == focusRight && i == m.rightCursor
		label := truncateLabel(row.label, maxLabel)

		switch row.kind {
		case rowSelected:
			var check string
			if m.leftCursor == leftPaneLevel {
				if m.filter.Level == row.label {
					check = checkedStyle.Render("✔ ")
				} else {
					check = uncheckedStyle.Render("✘ ")
				}
			} else {
				check = checkedStyle.Render("✔ ")
			}
			if isCursor {
				b.WriteString(cursorStyle.Render("▶ ") + check + activeStyle.Render(label) + "\n")
			} else {
				b.WriteString("  " + check + normalStyle.Render(label) + "\n")
			}

		case rowAddLabel:
			if isCursor {
				b.WriteString(cursorStyle.Render("▶ ") + "  " + addStyle.Render("[ + add label ]") + "\n")
			} else {
				b.WriteString("    " + addStyle.Render("[ + add label ]") + "\n")
			}

		case rowDivider:
			sep := dimStyle.Render(strings.Repeat("─", w-2))
			b.WriteString("  " + sep + "\n")

		case rowSuggestion:
			if isCursor {
				b.WriteString(cursorStyle.Render("▶ ") + "  " + activeStyle.Render(label) + "\n")
			} else {
				b.WriteString("    " + suggStyle.Render(label) + "\n")
			}
		}
	}

	if m.rightViewOffset > 0 {
		indicator := dimStyle.Render(fmt.Sprintf("  ↑ %d more", m.rightViewOffset))
		result := strings.TrimRight(b.String(), "\n")
		lines := strings.Split(result, "\n")
		if len(lines) >= rightPaneVisibleRows {
			lines = lines[:rightPaneVisibleRows-1]
		}
		return indicator + "\n" + strings.Join(lines, "\n")
	}
	below := total - visEnd
	if below > 0 {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  ↓ %d more", below)) + "\n")
	}

	result := strings.TrimRight(b.String(), "\n")
	if result == "" {
		result = dimStyle.Render("  (empty)")
	}
	return result
}

func (m *FilterModal) renderFooter() string {
	keyStyle := lipgloss.NewStyle().Foreground(m.styles.HintKeyColor).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(m.styles.HintDescColor)
	sep := descStyle.Render("  │  ")

	bk := func(b key.Binding) string { return b.Help().Key }

	var parts []string
	parts = append(parts, keyStyle.Render("<"+bk(m.keys.Up)+"/"+bk(m.keys.Down)+">")+" "+descStyle.Render("move"))
	if m.focus == focusLeft {
		parts = append(parts, keyStyle.Render("<"+bk(m.keys.Enter)+"/"+bk(m.keys.Right)+">")+" "+descStyle.Render("open"))
	} else {
		if m.leftCursor == leftPaneLevel {
			parts = append(parts, keyStyle.Render("<"+bk(m.keys.Enter)+">")+" "+descStyle.Render("select"))
		} else {
			parts = append(parts, keyStyle.Render("<"+bk(m.keys.Enter)+">")+" "+descStyle.Render("toggle"))
		}
		parts = append(parts, keyStyle.Render("<"+bk(m.keys.Left)+"/"+bk(m.keys.Back)+">")+" "+descStyle.Render("back"))
	}
	parts = append(parts, keyStyle.Render("<"+bk(m.keys.ApplyFilter)+">")+" "+descStyle.Render("apply"))
	parts = append(parts, keyStyle.Render("<"+bk(m.keys.Back)+">")+" "+descStyle.Render("close"))

	return " " + strings.Join(parts, sep)
}

func appendUnique(slice []string, v string) []string {
	for _, e := range slice {
		if e == v {
			return slice
		}
	}
	return append(slice, v)
}

func removeFromSlice(slice []string, v string) []string {
	out := make([]string, 0, len(slice))
	for _, e := range slice {
		if e != v {
			out = append(out, e)
		}
	}
	return out
}
