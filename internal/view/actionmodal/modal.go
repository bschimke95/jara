// Package actionmodal implements an action selection and execution modal.
// It lists available charm actions for a unit, lets the user select one,
// optionally collects parameters, and displays the result after execution.
package actionmodal

import (
	"fmt"
	"sort"
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
	phaseParams
	phaseRunning
	phaseResult
)

// CloseMsg is emitted when the modal should be closed.
type CloseMsg struct{}

// paramField represents a single parameter input field.
type paramField struct {
	Name        string
	Description string
	Type        string // e.g. "string", "integer", "boolean"
	Default     string
	Required    bool
	Value       string
}

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

	// Unit selector: when multiple units are available the user can cycle.
	availableUnits []string // sorted unit names (e.g. ["mysql/0", "mysql/1"])
	unitIndex      int      // index into availableUnits

	// Parameter entry state.
	paramFields []paramField
	paramCursor int
	paramEdit   bool // true when editing a parameter value
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

// NewWithUnits creates a new action modal with a list of available units.
// The leader unit should be first in the list. The first unit is pre-selected.
func NewWithUnits(appName string, unitNames []string, keys ui.KeyMap, styles *color.Styles) *Modal {
	unitName := ""
	if len(unitNames) > 0 {
		unitName = unitNames[0]
	}
	return &Modal{
		keys:           keys,
		styles:         styles,
		unitName:       unitName,
		appName:        appName,
		availableUnits: unitNames,
		unitIndex:      0,
		phase:          phaseLoading,
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
		case key.Matches(msg, m.keys.Tab):
			// Cycle through available units.
			if len(m.availableUnits) > 1 {
				m.unitIndex = (m.unitIndex + 1) % len(m.availableUnits)
				m.unitName = m.availableUnits[m.unitIndex]
			}
		case key.Matches(msg, m.keys.Enter):
			if len(m.actions) > 0 {
				selected := m.actions[m.cursor]
				// If the action has parameters, show the params phase.
				if len(selected.Params) > 0 {
					m.paramFields = buildParamFields(selected.Params)
					m.paramCursor = 0
					m.paramEdit = false
					m.phase = phaseParams
					return m, nil
				}
				// No parameters — run immediately.
				m.phase = phaseRunning
				return m, func() tea.Msg {
					return view.RunActionRequestMsg{
						UnitName:   m.unitName,
						ActionName: selected.Name,
					}
				}
			}
		case key.Matches(msg, m.keys.Down):
			if m.cursor < len(m.actions)-1 {
				m.cursor++
			}
		case key.Matches(msg, m.keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		}

	case phaseParams:
		return m.handleParamsKey(msg)

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

func (m *Modal) handleParamsKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.paramEdit {
		// Editing a parameter value inline.
		switch {
		case key.Matches(msg, m.keys.CancelInput):
			m.paramEdit = false
		case key.Matches(msg, m.keys.Enter):
			m.paramEdit = false
		case msg.Key().Code == tea.KeyBackspace:
			v := m.paramFields[m.paramCursor].Value
			if len(v) > 0 {
				m.paramFields[m.paramCursor].Value = v[:len(v)-1]
			}
		default:
			r := msg.Key().Text
			if r != "" {
				m.paramFields[m.paramCursor].Value += r
			}
		}
		return m, nil
	}

	switch {
	case key.Matches(msg, m.keys.Back):
		// Go back to action selection.
		m.phase = phaseSelect
		m.paramFields = nil
	case key.Matches(msg, m.keys.Enter):
		if m.paramCursor >= len(m.paramFields) {
			// Cursor is on the "Run" button — execute.
			selected := m.actions[m.cursor]
			params := make(map[string]string, len(m.paramFields))
			for _, f := range m.paramFields {
				if f.Value != "" {
					params[f.Name] = f.Value
				}
			}
			m.phase = phaseRunning
			return m, func() tea.Msg {
				return view.RunActionRequestMsg{
					UnitName:   m.unitName,
					ActionName: selected.Name,
					Params:     params,
				}
			}
		}
		// Start editing the current field.
		m.paramEdit = true
	case key.Matches(msg, m.keys.Down):
		if m.paramCursor <= len(m.paramFields) {
			m.paramCursor++
			if m.paramCursor > len(m.paramFields) {
				m.paramCursor = len(m.paramFields)
			}
		}
	case key.Matches(msg, m.keys.Up):
		if m.paramCursor > 0 {
			m.paramCursor--
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
	innerW := m.width * 70 / 100
	if innerW < 60 {
		innerW = 60
	}
	if innerW > 120 {
		innerW = 120
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
			content = m.renderSelectTwoPane(contentW)
		}
		hintParts := "[enter] run  [↑/↓] select  [esc] close"
		if len(m.availableUnits) > 1 {
			hintParts = "[enter] run  [↑/↓] select  [tab] unit  [esc] close"
		}
		hint := lipgloss.NewStyle().Foreground(m.styles.Muted).Width(contentW).
			AlignHorizontal(lipgloss.Center).Render(hintParts)
		content += "\n" + hint

	case phaseParams:
		selected := m.actions[m.cursor]
		title = fmt.Sprintf(" %s · Parameters ", selected.Name)
		content = m.renderParamsTwoPane(contentW)

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

// renderSelectTwoPane renders the action selection phase as a two-pane layout:
// action list on the left, description of the selected action on the right.
func (m *Modal) renderSelectTwoPane(contentW int) string {
	mutedStyle := lipgloss.NewStyle().Foreground(m.styles.Muted)
	boldStyle := lipgloss.NewStyle().Bold(true)

	leftW := contentW * 40 / 100
	if leftW < 20 {
		leftW = 20
	}
	rightW := contentW - leftW - 3 // 3 for " │ " separator

	// Build left pane: action list.
	var left strings.Builder
	if len(m.availableUnits) > 1 {
		unitLabel := lipgloss.NewStyle().Foreground(m.styles.Secondary).
			Render("Target: ")
		unitValue := boldStyle.Render(m.unitName)
		unitHint := mutedStyle.Render(fmt.Sprintf(" [tab] %d/%d", m.unitIndex+1, len(m.availableUnits)))
		left.WriteString(unitLabel + unitValue + unitHint + "\n\n")
	}
	for i, a := range m.actions {
		prefix := "  "
		if i == m.cursor {
			prefix = color.ForegroundText(m.styles.HintKeyColor, "▸ ")
		}
		name := a.Name
		if i == m.cursor {
			name = boldStyle.Render(name)
		}
		left.WriteString(prefix + name + "\n")
	}

	// Build right pane: description of selected action.
	var right strings.Builder
	if m.cursor < len(m.actions) {
		selected := m.actions[m.cursor]
		right.WriteString(boldStyle.Render(selected.Name) + "\n\n")
		if selected.Description != "" {
			wrapped := lipgloss.NewStyle().Width(rightW).Render(selected.Description)
			right.WriteString(wrapped + "\n")
		} else {
			right.WriteString(mutedStyle.Render("No description available.") + "\n")
		}
		if len(selected.Params) > 0 {
			right.WriteString("\n" + mutedStyle.Render("Parameters:") + "\n")
			pf := buildParamFields(selected.Params)
			for _, f := range pf {
				label := "  " + f.Name
				if f.Type != "" {
					label += mutedStyle.Render(" (" + f.Type + ")")
				}
				if f.Required {
					label += mutedStyle.Render(" *")
				}
				right.WriteString(label + "\n")
			}
		}
	}

	leftRendered := lipgloss.NewStyle().Width(leftW).Render(left.String())
	rightRendered := lipgloss.NewStyle().Width(rightW).Render(right.String())

	sep := lipgloss.NewStyle().Foreground(m.styles.Muted).Render("│")
	return joinHorizontal(leftRendered, sep, rightRendered)
}

// renderParamsTwoPane renders the parameter entry phase as a two-pane layout:
// parameter list with inputs on the left, description of selected param on the right.
func (m *Modal) renderParamsTwoPane(contentW int) string {
	mutedStyle := lipgloss.NewStyle().Foreground(m.styles.Muted)
	boldStyle := lipgloss.NewStyle().Bold(true)

	leftW := contentW * 50 / 100
	if leftW < 25 {
		leftW = 25
	}
	rightW := contentW - leftW - 3

	// Build left pane: parameter fields.
	var left strings.Builder
	for i, f := range m.paramFields {
		prefix := "  "
		if i == m.paramCursor {
			prefix = color.ForegroundText(m.styles.HintKeyColor, "▸ ")
		}
		label := f.Name
		if i == m.paramCursor {
			label = boldStyle.Render(label)
		}

		var tags []string
		if f.Type != "" {
			tags = append(tags, f.Type)
		}
		if f.Required {
			tags = append(tags, "required")
		}
		tagStr := ""
		if len(tags) > 0 {
			tagStr = mutedStyle.Render(" (" + strings.Join(tags, ", ") + ")")
		}
		left.WriteString(prefix + label + tagStr + "\n")

		val := f.Value
		if i == m.paramCursor && m.paramEdit {
			val += "█"
		}
		if val == "" && (i != m.paramCursor || !m.paramEdit) {
			placeholder := "(empty)"
			if f.Default != "" {
				placeholder = fmt.Sprintf("(default: %s)", f.Default)
			}
			left.WriteString("    " + mutedStyle.Render(placeholder) + "\n")
		} else {
			left.WriteString("    " + val + "\n")
		}
		left.WriteString("\n")
	}

	left.WriteString("\n")
	runLabel := "  [Run Action]"
	if m.paramCursor >= len(m.paramFields) {
		runLabel = color.ForegroundText(m.styles.HintKeyColor, "▸ ") + boldStyle.Render("[Run Action]")
	}
	left.WriteString(runLabel + "\n")

	// Build right pane: description of selected parameter.
	var right strings.Builder
	if m.paramCursor < len(m.paramFields) {
		f := m.paramFields[m.paramCursor]
		right.WriteString(boldStyle.Render(f.Name) + "\n\n")
		if f.Description != "" {
			wrapped := lipgloss.NewStyle().Width(rightW).Render(f.Description)
			right.WriteString(wrapped + "\n")
		} else {
			right.WriteString(mutedStyle.Render("No description.") + "\n")
		}
		if f.Default != "" {
			right.WriteString("\n" + mutedStyle.Render("Default: "+f.Default) + "\n")
		}
	} else {
		right.WriteString(mutedStyle.Render("Press enter to execute the action.") + "\n")
	}

	leftRendered := lipgloss.NewStyle().Width(leftW).Render(left.String())
	rightRendered := lipgloss.NewStyle().Width(rightW).Render(right.String())

	sep := lipgloss.NewStyle().Foreground(m.styles.Muted).Render("│")
	result := joinHorizontal(leftRendered, sep, rightRendered)

	hint := lipgloss.NewStyle().Foreground(m.styles.Muted).Width(contentW).
		AlignHorizontal(lipgloss.Center).Render("[enter] edit/run  [↑/↓] navigate  [esc] back")
	return result + "\n" + hint
}

// joinHorizontal places left, separator, and right side by side line-by-line.
func joinHorizontal(left, sep, right string) string {
	leftLines := strings.Split(left, "\n")
	rightLines := strings.Split(right, "\n")

	maxLines := len(leftLines)
	if len(rightLines) > maxLines {
		maxLines = len(rightLines)
	}

	leftW := 0
	for _, l := range leftLines {
		if w := lipgloss.Width(l); w > leftW {
			leftW = w
		}
	}

	var sb strings.Builder
	for i := 0; i < maxLines; i++ {
		l := ""
		if i < len(leftLines) {
			l = leftLines[i]
		}
		r := ""
		if i < len(rightLines) {
			r = rightLines[i]
		}
		pad := leftW - lipgloss.Width(l)
		if pad < 0 {
			pad = 0
		}
		sb.WriteString(l + strings.Repeat(" ", pad) + " " + sep + " " + r + "\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

// buildParamFields extracts parameter names from a JSON-Schema style params map.
// Juju action params follow JSON-Schema: the actual parameters live under the
// "properties" key while top-level keys like "type", "title", "required" etc.
// are schema metadata.
func buildParamFields(params map[string]interface{}) []paramField {
	props, hasProperties := params["properties"].(map[string]interface{})
	if !hasProperties {
		props = params
	}

	// Collect the set of required parameter names.
	requiredSet := make(map[string]bool)
	if reqList, ok := params["required"].([]interface{}); ok {
		for _, r := range reqList {
			if s, ok := r.(string); ok {
				requiredSet[s] = true
			}
		}
	}

	fields := make([]paramField, 0, len(props))
	for name, spec := range props {
		// Skip JSON-Schema meta keys that leaked into props when there
		// was no "properties" wrapper.
		switch name {
		case "type", "title", "description", "required", "properties",
			"additionalProperties", "$schema", "definitions":
			if !hasProperties {
				continue
			}
		}

		f := paramField{Name: name, Required: requiredSet[name]}
		if m, ok := spec.(map[string]interface{}); ok {
			if d, ok := m["description"].(string); ok {
				f.Description = d
			}
			if t, ok := m["type"].(string); ok {
				f.Type = t
			}
			if d, ok := m["default"]; ok {
				f.Default = fmt.Sprintf("%v", d)
			}
		}
		fields = append(fields, f)
	}
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Name < fields[j].Name
	})
	return fields
}
