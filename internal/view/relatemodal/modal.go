// Package relatemodal implements a modal overlay for adding relations
// between two application endpoints.
package relatemodal

import (
	"sort"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/ui"
)

// AppliedMsg is emitted when the user confirms the relation.
type AppliedMsg struct {
	EndpointA string
	EndpointB string
}

// ClosedMsg is emitted when the modal is cancelled.
type ClosedMsg struct{}

// endpointSuggestion carries structured autocomplete data.
type endpointSuggestion struct {
	Display   string // "mysql:db" or "mysql"
	App       string // "mysql"
	Endpoint  string // "db" (empty for app-only)
	Interface string // "mysql"
	Role      string // "provider", "requirer", "peer"
}

type fieldFocus int

const (
	focusEndpointA fieldFocus = iota
	focusEndpointB
)

type infoMode int

const (
	infoOff infoMode = iota
	infoOn
)

// Modal is the relate-applications overlay.
type Modal struct {
	keys   ui.KeyMap
	width  int
	height int

	focus   fieldFocus
	editing bool
	info    infoMode

	input textinput.Model

	endpointA string
	endpointB string

	suggestions       []endpointSuggestion
	autocomplete      []endpointSuggestion
	autocompleteIndex int

	relations []model.Relation

	validationErr string
}

// New creates a new relate modal.
// prefill is an optional app name to pre-populate endpoint A.
func New(keys ui.KeyMap, suggestions []endpointSuggestion, relations []model.Relation, prefill string) Modal {
	ti := textinput.New()
	ti.CharLimit = 128
	ti.Placeholder = "app:endpoint"

	m := Modal{
		keys:        keys,
		input:       ti,
		suggestions: suggestions,
		relations:   relations,
	}
	if prefill != "" {
		m.endpointA = prefill
		m.focus = focusEndpointB
	}
	return m
}

// BuildSuggestions constructs endpoint suggestions from the current status.
// It uses EndpointBindings (all charm endpoints) as the primary source and
// enriches with interface/role info from active relations where available.
func BuildSuggestions(status *model.FullStatus) []endpointSuggestion {
	if status == nil {
		return nil
	}

	// Build a lookup from active relations: "app:endpoint" → (interface, role).
	type relInfo struct {
		iface string
		role  string
	}
	relLookup := make(map[string]relInfo)
	for _, rel := range status.Relations {
		for _, ep := range rel.Endpoints {
			key := ep.ApplicationName + ":" + ep.Name
			relLookup[key] = relInfo{iface: rel.Interface, role: ep.Role}
		}
	}

	var result []endpointSuggestion

	for name, app := range status.Applications {
		// App-only suggestion (bare app name).
		result = append(result, endpointSuggestion{
			Display: name,
			App:     name,
		})

		// One suggestion per endpoint from bindings.
		for epName := range app.EndpointBindings {
			key := name + ":" + epName
			s := endpointSuggestion{
				Display:  key,
				App:      name,
				Endpoint: epName,
			}
			if ri, ok := relLookup[key]; ok {
				s.Interface = ri.iface
				s.Role = ri.role
			}
			result = append(result, s)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Display < result[j].Display
	})
	return result
}

func (m *Modal) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// BeginEdit focuses the active endpoint field for editing.
func (m *Modal) BeginEdit() tea.Cmd {
	m.editing = true
	m.input.SetValue(m.activeValue())
	m.refreshAutocomplete()
	m.validationErr = ""
	return m.input.Focus()
}

func (m *Modal) Init() tea.Cmd { return nil }

func (m *Modal) View() tea.View {
	return tea.NewView(m.Render(""))
}

func (m *Modal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	kp, isKey := msg.(tea.KeyPressMsg)

	// Info mode: any key dismisses.
	if m.info == infoOn {
		if isKey {
			m.info = infoOff
		}
		return m, nil
	}

	if m.editing {
		if isKey {
			switch {
			case key.Matches(kp, m.keys.Down):
				m.advanceSuggestion(1)
				return m, nil
			case key.Matches(kp, m.keys.Up):
				m.advanceSuggestion(-1)
				return m, nil
			case key.Matches(kp, m.keys.Tab):
				if m.acceptSuggestion(false) {
					return m, nil
				}
				// Tab with no autocomplete: switch field.
				m.commitEdit()
				m.switchFocus()
				return m, m.BeginEdit()
			case key.Matches(kp, m.keys.Enter):
				m.acceptSuggestion(true)
				m.commitEdit()
				// If on endpoint A, advance to B.
				if m.focus == focusEndpointA {
					m.switchFocus()
					return m, m.BeginEdit()
				}
				// On endpoint B, try to apply.
				return m.tryApply()
			case key.Matches(kp, m.keys.Back):
				m.editing = false
				m.autocomplete = nil
				m.autocompleteIndex = 0
				m.input.Blur()
				return m, nil
			case kp.String() == "i":
				if len(m.autocomplete) > 0 {
					m.info = infoOn
					return m, nil
				}
			}
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		m.refreshAutocomplete()
		return m, cmd
	}

	// Not editing.
	if !isKey {
		return m, nil
	}

	switch {
	case key.Matches(kp, m.keys.Back):
		return m, func() tea.Msg { return ClosedMsg{} }
	case key.Matches(kp, m.keys.Tab):
		m.switchFocus()
		return m, nil
	case key.Matches(kp, m.keys.Enter):
		return m, m.BeginEdit()
	case kp.String() == "i":
		return m, m.BeginEdit()
	}
	return m, nil
}

func (m *Modal) switchFocus() {
	if m.focus == focusEndpointA {
		m.focus = focusEndpointB
	} else {
		m.focus = focusEndpointA
	}
}

func (m *Modal) activeValue() string {
	if m.focus == focusEndpointA {
		return m.endpointA
	}
	return m.endpointB
}

func (m *Modal) commitEdit() {
	val := strings.TrimSpace(m.input.Value())
	if m.focus == focusEndpointA {
		m.endpointA = val
	} else {
		m.endpointB = val
	}
	m.editing = false
	m.autocomplete = nil
	m.autocompleteIndex = 0
	m.input.Blur()
}

func (m *Modal) tryApply() (tea.Model, tea.Cmd) {
	a := strings.TrimSpace(m.endpointA)
	b := strings.TrimSpace(m.endpointB)
	if a == "" || b == "" {
		m.validationErr = "both endpoints are required"
		return m, nil
	}
	m.validationErr = ""
	return m, func() tea.Msg {
		return AppliedMsg{EndpointA: a, EndpointB: b}
	}
}

// ── Autocomplete ──

func (m *Modal) refreshAutocomplete() {
	m.autocomplete = m.filteredSuggestions()
	if len(m.autocomplete) == 0 {
		m.autocompleteIndex = 0
		return
	}
	if m.autocompleteIndex >= len(m.autocomplete) {
		m.autocompleteIndex = 0
	}
}

func (m *Modal) filteredSuggestions() []endpointSuggestion {
	if len(m.suggestions) == 0 {
		return nil
	}
	q := strings.ToLower(strings.TrimSpace(m.input.Value()))
	if q == "" {
		return append([]endpointSuggestion(nil), m.suggestions...)
	}
	var prefix, contains []endpointSuggestion
	for _, s := range m.suggestions {
		sl := strings.ToLower(s.Display)
		if strings.HasPrefix(sl, q) {
			prefix = append(prefix, s)
			continue
		}
		if strings.Contains(sl, q) {
			contains = append(contains, s)
		}
	}
	return append(prefix, contains...)
}

func (m *Modal) advanceSuggestion(delta int) {
	if len(m.autocomplete) == 0 {
		return
	}
	next := m.autocompleteIndex + delta
	if next < 0 {
		next = len(m.autocomplete) - 1
	}
	if next >= len(m.autocomplete) {
		next = 0
	}
	m.autocompleteIndex = next
}

func (m *Modal) acceptSuggestion(force bool) bool {
	if len(m.autocomplete) == 0 {
		return false
	}
	if !force && strings.TrimSpace(m.input.Value()) == "" {
		return false
	}
	m.input.SetValue(m.autocomplete[m.autocompleteIndex].Display)
	m.refreshAutocomplete()
	return true
}

// ── Rendering ──

// Render draws the modal as an overlay on the given background.
func (m *Modal) Render(background string) string {
	innerW := m.width * 45 / 100
	if innerW < 40 {
		innerW = 40
	}
	if innerW > 70 {
		innerW = 70
	}

	content := m.renderContent(innerW - 2)
	if m.validationErr != "" {
		errStyle := lipgloss.NewStyle().Foreground(color.Error)
		content += "\n" + errStyle.Render("  "+m.validationErr)
	}
	content += "\n" + m.renderFooter()

	titleStyle := lipgloss.NewStyle().Foreground(color.Primary).Bold(true)
	box := ui.BorderBoxRawTitle(content, titleStyle.Render(" Add Relation "), innerW+4)

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

func (m *Modal) renderContent(width int) string {
	labelStyle := lipgloss.NewStyle().Foreground(color.InfoLabel).Bold(true)
	valueStyle := lipgloss.NewStyle().Foreground(color.Title)

	var lines []string

	// Endpoint A.
	aLabel := "Endpoint A"
	if m.focus == focusEndpointA {
		aLabel = "▶ Endpoint A"
	}
	lines = append(lines, labelStyle.Render(aLabel))
	lines = append(lines, m.renderField(m.endpointA, focusEndpointA, width, valueStyle))
	if m.focus == focusEndpointA && m.editing {
		lines = append(lines, m.renderDropdown(width)...)
	}
	lines = append(lines, "")

	// Endpoint B.
	bLabel := "Endpoint B"
	if m.focus == focusEndpointB {
		bLabel = "▶ Endpoint B"
	}
	lines = append(lines, labelStyle.Render(bLabel))
	lines = append(lines, m.renderField(m.endpointB, focusEndpointB, width, valueStyle))
	if m.focus == focusEndpointB && m.editing {
		lines = append(lines, m.renderDropdown(width)...)
	}

	return strings.Join(lines, "\n")
}

func (m *Modal) renderField(value string, field fieldFocus, _ int, valueStyle lipgloss.Style) string {
	if m.focus == field && m.editing {
		inputLine := valueStyle.Render(m.input.Value()) + lipgloss.NewStyle().Foreground(color.Primary).Render("█")
		return "  " + inputLine
	}
	v := value
	if strings.TrimSpace(v) == "" {
		v = "(unset)"
	}
	return "  " + valueStyle.Render(v)
}

func (m *Modal) renderDropdown(width int) []string {
	if m.info == infoOn {
		return m.renderInfoBox(width)
	}

	if len(m.autocomplete) == 0 {
		return nil
	}

	selectedStyle := lipgloss.NewStyle().Foreground(color.CrumbFg).Background(color.Highlight).Bold(true)
	itemStyle := lipgloss.NewStyle().Foreground(color.Title)
	roleStyle := lipgloss.NewStyle().Foreground(color.Muted)

	var lines []string
	limit := len(m.autocomplete)
	if limit > 8 {
		limit = 8
	}
	for i := 0; i < limit; i++ {
		s := m.autocomplete[i]
		name := truncate(s.Display, width-14)
		role := ""
		if s.Role != "" {
			role = s.Role
		}
		// Right-align role.
		nameW := lipgloss.Width(name)
		roleW := lipgloss.Width(role)
		pad := width - 4 - nameW - roleW
		if pad < 1 {
			pad = 1
		}
		line := "  " + name + strings.Repeat(" ", pad) + roleStyle.Render(role)
		if i == m.autocompleteIndex {
			lines = append(lines, selectedStyle.Render(line))
		} else {
			lines = append(lines, itemStyle.Render(line))
		}
	}
	return lines
}

func (m *Modal) renderInfoBox(width int) []string {
	if len(m.autocomplete) == 0 || m.autocompleteIndex >= len(m.autocomplete) {
		return nil
	}

	sel := m.autocomplete[m.autocompleteIndex]
	labelStyle := lipgloss.NewStyle().Foreground(color.InfoLabel)
	valueStyle := lipgloss.NewStyle().Foreground(color.Title)
	mutedStyle := lipgloss.NewStyle().Foreground(color.Muted)

	var lines []string

	if sel.Endpoint == "" {
		// App-only suggestion: show all endpoints for this app.
		lines = append(lines, labelStyle.Render("  Endpoints for ")+valueStyle.Render(sel.App)+labelStyle.Render(":"))
		for _, s := range m.suggestions {
			if s.App == sel.App && s.Endpoint != "" {
				role := ""
				if s.Role != "" {
					role = " (" + s.Role + ")"
				}
				lines = append(lines, "    "+valueStyle.Render(s.Display)+mutedStyle.Render(role))
			}
		}
		if len(lines) == 1 {
			lines = append(lines, mutedStyle.Render("    (no known endpoints)"))
		}
	} else {
		// Specific endpoint info.
		lines = append(lines, labelStyle.Render("  Interface:  ")+valueStyle.Render(sel.Interface))
		lines = append(lines, labelStyle.Render("  Role:       ")+valueStyle.Render(sel.Role))

		// Find scope from relations.
		for _, rel := range m.relations {
			for _, ep := range rel.Endpoints {
				if ep.ApplicationName == sel.App && ep.Name == sel.Endpoint {
					lines = append(lines, labelStyle.Render("  Scope:      ")+valueStyle.Render(rel.Scope))
					break
				}
			}
			if len(lines) > 2 {
				break
			}
		}

		// Active relations for this endpoint.
		lines = append(lines, "")
		lines = append(lines, labelStyle.Render("  Active relations:"))
		found := false
		for _, rel := range m.relations {
			for _, ep := range rel.Endpoints {
				if ep.ApplicationName == sel.App && ep.Name == sel.Endpoint {
					for _, peer := range rel.Endpoints {
						if peer.ApplicationName != sel.App || peer.Name != sel.Endpoint {
							peerStr := peer.ApplicationName + ":" + peer.Name
							lines = append(lines, "    → "+valueStyle.Render(peerStr)+mutedStyle.Render(" ("+rel.Status+")"))
							found = true
						}
					}
				}
			}
		}
		if !found {
			lines = append(lines, mutedStyle.Render("    (none)"))
		}
	}

	// Pad to at least fill the dropdown area.
	_ = width
	return lines
}

func (m *Modal) renderFooter() string {
	keyStyle := lipgloss.NewStyle().Foreground(color.HintKey).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(color.HintDesc)
	parts := []string{
		keyStyle.Render("Tab") + descStyle.Render(" next"),
		keyStyle.Render("i") + descStyle.Render(" info"),
		keyStyle.Render("Enter") + descStyle.Render(" relate"),
		keyStyle.Render("Esc") + descStyle.Render(" close"),
	}
	return strings.Join(parts, descStyle.Render("  •  "))
}

func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= max {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes)+"…") > max {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}
