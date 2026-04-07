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
	Display     string // "mysql:db" or "mysql"
	App         string // "mysql"
	Endpoint    string // "db" (empty for app-only)
	Interface   string // "mysql"
	Role        string // "provider", "requirer", "peer"
	Description string // from Charmhub metadata
}

type fieldFocus int

const (
	focusEndpointA fieldFocus = iota
	focusEndpointB
)

// Modal is the relate-applications overlay.
type Modal struct {
	keys   ui.KeyMap
	styles *color.Styles
	width  int
	height int

	focus   fieldFocus
	editing bool

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
func New(keys ui.KeyMap, styles *color.Styles, suggestions []endpointSuggestion, relations []model.Relation) Modal {
	ti := textinput.New()
	ti.CharLimit = 128
	ti.Placeholder = "app:endpoint"

	return Modal{
		keys:        keys,
		styles:      styles,
		input:       ti,
		suggestions: suggestions,
		relations:   relations,
	}
}

// BuildSuggestions constructs endpoint suggestions from the current status.
// It uses EndpointBindings (all charm endpoints) as the primary source and
// enriches with interface/role/description from Charmhub and active relations.
func BuildSuggestions(status *model.FullStatus, charmEndpoints map[string]map[string]model.CharmEndpoint) []endpointSuggestion {
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

		// Charmhub endpoint metadata for this app's charm.
		var charmEPs map[string]model.CharmEndpoint
		if charmEndpoints != nil {
			charmEPs = charmEndpoints[app.Charm]
		}

		// One suggestion per endpoint from bindings.
		for epName := range app.EndpointBindings {
			// Skip empty endpoint names and endpoints whose name matches
			// the app name — the bare app suggestion already covers that.
			if epName == "" || epName == name {
				continue
			}
			key := name + ":" + epName
			s := endpointSuggestion{
				Display:  key,
				App:      name,
				Endpoint: epName,
			}
			// Enrich from Charmhub metadata first.
			if ce, ok := charmEPs[epName]; ok {
				s.Interface = ce.Interface
				s.Role = ce.Role
				s.Description = ce.Description
			}
			// Override interface/role from active relations (more accurate for live state).
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
				return m, func() tea.Msg { return ClosedMsg{} }
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
		errStyle := lipgloss.NewStyle().Foreground(m.styles.ErrorColor)
		content += "\n" + errStyle.Render("  "+m.validationErr)
	}
	content += "\n" + m.renderFooter()

	titleStyle := lipgloss.NewStyle().Foreground(m.styles.Primary).Bold(true)
	box := ui.BorderBoxRawTitle(content, titleStyle.Render(" Add Relation "), innerW+4, m.styles)

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
	labelStyle := lipgloss.NewStyle().Foreground(m.styles.InfoLabelColor).Bold(true)
	valueStyle := lipgloss.NewStyle().Foreground(m.styles.Title)

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
		inputLine := valueStyle.Render(m.input.Value()) + lipgloss.NewStyle().Foreground(m.styles.Primary).Render("█")
		return "  " + inputLine
	}
	v := value
	if strings.TrimSpace(v) == "" {
		v = "(unset)"
	}
	return "  " + valueStyle.Render(v)
}

func (m *Modal) renderDropdown(width int) []string {
	if len(m.autocomplete) == 0 {
		return nil
	}

	selectedStyle := lipgloss.NewStyle().Foreground(m.styles.CrumbFgColor).Background(m.styles.Highlight).Bold(true)
	itemStyle := lipgloss.NewStyle().Foreground(m.styles.Title)
	roleStyle := lipgloss.NewStyle().Foreground(m.styles.Muted)

	maxVisible := 8
	total := len(m.autocomplete)
	if maxVisible > total {
		maxVisible = total
	}

	// Window around the selected index.
	start := m.autocompleteIndex - maxVisible/2
	if start < 0 {
		start = 0
	}
	if start+maxVisible > total {
		start = total - maxVisible
	}

	var lines []string
	for i := start; i < start+maxVisible; i++ {
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

func (m *Modal) renderFooter() string {
	keyStyle := lipgloss.NewStyle().Foreground(m.styles.HintKeyColor).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(m.styles.HintDescColor)
	parts := []string{
		keyStyle.Render("Tab") + descStyle.Render(" next"),
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
