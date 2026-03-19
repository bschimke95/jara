package deploymodal

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/ui"
)

type paneFocus int

const (
	focusLeft paneFocus = iota
	focusRight
)

type fieldID int

const (
	fieldCharm fieldID = iota
	fieldAppName
	fieldChannel
	fieldBase
	fieldNumUnits
	fieldRevision
	fieldConstraints
	fieldConfig
	fieldTrust
	fieldDeploy
	fieldCount
)

var fieldNames = []string{
	"Charm",
	"Application name",
	"Channel",
	"Base",
	"Units",
	"Revision",
	"Constraints",
	"Config",
	"Trust",
	"Deploy",
}

// AppliedMsg is emitted when the user confirms deployment.
type AppliedMsg struct {
	ModelName string
	Options   model.DeployOptions
}

// ClosedMsg is emitted when the modal is cancelled.
type ClosedMsg struct{}

// Modal is a two-pane overlay for configuring deploy options.
type Modal struct {
	keys      ui.KeyMap
	modelName string

	charmSuggestions []string
	appSuggestions   []string

	width  int
	height int

	leftCursor fieldID
	focus      paneFocus
	editing    bool
	configMode bool

	input textinput.Model

	charm       string
	appName     string
	channel     string
	base        string
	numUnits    string
	revision    string
	constraints string
	config      string
	trust       bool

	validationErr string

	autocomplete      []string
	autocompleteIndex int
}

// New creates a new deploy modal.
func New(modelName string, keys ui.KeyMap, charmSuggestions, appSuggestions []string) Modal {
	ti := textinput.New()
	ti.CharLimit = 256
	ti.Placeholder = "Enter value"
	return Modal{
		keys:             keys,
		modelName:        modelName,
		input:            ti,
		charmSuggestions: sortedUnique(charmSuggestions),
		appSuggestions:   sortedUnique(appSuggestions),
	}
}

func (m *Modal) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// BeginCharmEdit switches modal to simple charm edit mode and focuses input.
func (m *Modal) BeginCharmEdit() tea.Cmd {
	m.configMode = false
	m.leftCursor = fieldCharm
	m.input.SetValue(m.charm)
	m.editing = true
	m.validationErr = ""
	m.refreshAutocomplete()
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
				return m, nil
			case key.Matches(kp, m.keys.Enter):
				m.acceptSuggestion(true)
				m.setFieldValue(strings.TrimSpace(m.input.Value()))
				m.validationErr = ""
				m.editing = false
				m.autocomplete = nil
				m.autocompleteIndex = 0
				m.input.Blur()
				return m, nil
			case key.Matches(kp, m.keys.Back):
				m.editing = false
				m.autocomplete = nil
				m.autocompleteIndex = 0
				m.input.Blur()
				return m, nil
			}
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		m.refreshAutocomplete()
		return m, cmd
	}

	if !isKey {
		return m, nil
	}

	if !m.configMode {
		return m.updateSimpleMode(kp)
	}

	switch {
	case key.Matches(kp, m.keys.Back):
		if m.focus == focusRight {
			m.focus = focusLeft
			return m, nil
		}
		return m, func() tea.Msg { return ClosedMsg{} }
	case key.Matches(kp, m.keys.Deploy):
		return m.apply()
	}

	if m.focus == focusLeft {
		return m.updateLeftPane(kp)
	}
	return m.updateRightPane(kp)
}

func (m *Modal) updateSimpleMode(kp tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(kp, m.keys.Back):
		return m, func() tea.Msg { return ClosedMsg{} }
	case key.Matches(kp, m.keys.Deploy):
		return m.apply()
	case kp.String() == "o":
		m.configMode = true
		m.focus = focusLeft
		m.leftCursor = fieldCharm
		return m, nil
	case key.Matches(kp, m.keys.Enter):
		m.leftCursor = fieldCharm
		m.input.SetValue(m.charm)
		m.editing = true
		m.refreshAutocomplete()
		return m, m.input.Focus()
	}
	return m, nil
}

func (m *Modal) updateLeftPane(kp tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(kp, m.keys.Down):
		m.leftCursor = (m.leftCursor + 1) % fieldCount
	case key.Matches(kp, m.keys.Up):
		if m.leftCursor == 0 {
			m.leftCursor = fieldCount - 1
		} else {
			m.leftCursor--
		}
	case key.Matches(kp, m.keys.Enter, m.keys.Right):
		m.focus = focusRight
	}
	return m, nil
}

func (m *Modal) updateRightPane(kp tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(kp, m.keys.Left):
		m.focus = focusLeft
		return m, nil
	case key.Matches(kp, m.keys.Enter):
		if m.leftCursor == fieldDeploy {
			return m.apply()
		}
		if m.leftCursor == fieldTrust {
			m.trust = !m.trust
			return m, nil
		}
		m.input.SetValue(m.fieldValue())
		m.editing = true
		m.refreshAutocomplete()
		return m, m.input.Focus()
	}
	return m, nil
}

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

func (m *Modal) filteredSuggestions() []string {
	var source []string
	switch m.leftCursor {
	case fieldCharm:
		source = m.charmSuggestions
	case fieldAppName:
		source = m.appSuggestions
	default:
		return nil
	}
	if len(source) == 0 {
		return nil
	}
	q := strings.ToLower(strings.TrimSpace(m.input.Value()))
	if q == "" {
		return append([]string(nil), source...)
	}
	prefix := make([]string, 0, len(source))
	contains := make([]string, 0, len(source))
	for _, s := range source {
		sl := strings.ToLower(s)
		if strings.HasPrefix(sl, q) {
			prefix = append(prefix, s)
			continue
		}
		if strings.Contains(sl, q) {
			contains = append(contains, s)
		}
	}
	out := append(prefix, contains...)
	return out
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
	m.input.SetValue(m.autocomplete[m.autocompleteIndex])
	m.refreshAutocomplete()
	return true
}

func (m *Modal) apply() (tea.Model, tea.Cmd) {
	opts, err := m.options()
	if err != nil {
		m.validationErr = err.Error()
		return m, nil
	}
	m.validationErr = ""
	return m, func() tea.Msg {
		return AppliedMsg{ModelName: m.modelName, Options: opts}
	}
}

func (m *Modal) setFieldValue(v string) {
	switch m.leftCursor {
	case fieldCharm:
		m.charm = v
	case fieldAppName:
		m.appName = v
	case fieldChannel:
		m.channel = v
	case fieldBase:
		m.base = v
	case fieldNumUnits:
		m.numUnits = v
	case fieldRevision:
		m.revision = v
	case fieldConstraints:
		m.constraints = v
	case fieldConfig:
		m.config = v
	}
}

func (m *Modal) fieldValue() string {
	switch m.leftCursor {
	case fieldCharm:
		return m.charm
	case fieldAppName:
		return m.appName
	case fieldChannel:
		return m.channel
	case fieldBase:
		return m.base
	case fieldNumUnits:
		return m.numUnits
	case fieldRevision:
		return m.revision
	case fieldConstraints:
		return m.constraints
	case fieldConfig:
		return m.config
	case fieldTrust:
		if m.trust {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

func (m *Modal) options() (model.DeployOptions, error) {
	if strings.TrimSpace(m.charm) == "" {
		return model.DeployOptions{}, fmt.Errorf("charm is required")
	}
	opts := model.DeployOptions{
		CharmName:       strings.TrimSpace(m.charm),
		ApplicationName: strings.TrimSpace(m.appName),
		Channel:         strings.TrimSpace(m.channel),
		Base:            strings.TrimSpace(m.base),
		Constraints:     strings.TrimSpace(m.constraints),
		Trust:           m.trust,
	}
	if strings.TrimSpace(m.numUnits) != "" {
		n, err := strconv.Atoi(strings.TrimSpace(m.numUnits))
		if err != nil {
			return model.DeployOptions{}, fmt.Errorf("units must be an integer")
		}
		opts.NumUnits = &n
	}
	if strings.TrimSpace(m.revision) != "" {
		rev, err := strconv.Atoi(strings.TrimSpace(m.revision))
		if err != nil {
			return model.DeployOptions{}, fmt.Errorf("revision must be an integer")
		}
		opts.Revision = &rev
	}
	if strings.TrimSpace(m.config) != "" {
		cfg, err := parseConfigMap(m.config)
		if err != nil {
			return model.DeployOptions{}, err
		}
		opts.Config = cfg
	}
	return opts, nil
}

func parseConfigMap(raw string) (map[string]string, error) {
	cfg := make(map[string]string)
	parts := strings.Split(raw, ",")
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		k, v, ok := strings.Cut(item, "=")
		if !ok {
			return nil, fmt.Errorf("invalid config entry %q; expected key=value", item)
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k == "" {
			return nil, fmt.Errorf("config key cannot be empty")
		}
		cfg[k] = v
	}
	return cfg, nil
}

func (m *Modal) Render(background string) string {
	if !m.configMode {
		return m.renderSimple(background)
	}

	leftW := 22
	rightW := m.width * 40 / 100
	if rightW < 36 {
		rightW = 36
	}
	if rightW > 64 {
		rightW = 64
	}
	innerW := leftW + 2 + rightW + 2
	outerW := innerW + 4

	leftContent := m.renderLeftPane(leftW)
	rightContent := m.renderRightPane(rightW)
	leftBox := ui.BorderBox(leftContent, "Deploy options", leftW+2)
	rightBox := ui.BorderBox(rightContent, fieldNames[m.leftCursor], rightW+2)
	combined := lipgloss.JoinHorizontal(lipgloss.Top, leftBox, rightBox)
	if m.validationErr != "" {
		errStyle := lipgloss.NewStyle().Foreground(color.Error)
		combined += "\n" + errStyle.Render("  "+m.validationErr)
	}
	combined += "\n" + m.renderFooter()

	title := " Deploy Charm "
	if m.modelName != "" {
		title = " Deploy Charm(" + m.modelName + ") "
	}
	titleStyle := lipgloss.NewStyle().Foreground(color.Primary).Bold(true)
	box := ui.BorderBoxRawTitle(combined, titleStyle.Render(title), outerW)

	modalH := lipgloss.Height(box)
	x := (m.width - outerW) / 2
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

func (m *Modal) renderLeftPane(width int) string {
	selectedStyle := lipgloss.NewStyle().Foreground(color.CrumbFg).Background(color.Highlight).Bold(true)
	focusedStyle := lipgloss.NewStyle().Foreground(color.Title)
	mutedStyle := lipgloss.NewStyle().Foreground(color.Muted)

	var rows []string
	for i, name := range fieldNames {
		prefix := "  "
		if fieldID(i) == m.leftCursor {
			prefix = "▶ "
		}
		line := prefix + name
		if fieldID(i) == m.leftCursor {
			if m.focus == focusLeft {
				rows = append(rows, selectedStyle.Render(truncate(line, width)))
			} else {
				rows = append(rows, focusedStyle.Render(truncate(line, width)))
			}
		} else {
			rows = append(rows, mutedStyle.Render(truncate(line, width)))
		}
	}
	return strings.Join(rows, "\n")
}

func (m *Modal) renderRightPane(width int) string {
	labelStyle := lipgloss.NewStyle().Foreground(color.InfoLabel)
	valueStyle := lipgloss.NewStyle().Foreground(color.Title)
	hintStyle := lipgloss.NewStyle().Foreground(color.Muted)
	selectedStyle := lipgloss.NewStyle().Foreground(color.CrumbFg).Background(color.Highlight).Bold(true)

	if m.editing {
		inputLine := valueStyle.Render(m.input.Value()) + lipgloss.NewStyle().Foreground(color.Primary).Render("█")
		lines := []string{inputLine, hintStyle.Render("Enter: save  Tab: complete  Esc: cancel")}
		if len(m.autocomplete) > 0 {
			lines = append(lines, hintStyle.Render("Suggestions:"))
			limit := len(m.autocomplete)
			if limit > 6 {
				limit = 6
			}
			for i := 0; i < limit; i++ {
				item := "  " + truncate(m.autocomplete[i], width-2)
				if i == m.autocompleteIndex {
					lines = append(lines, selectedStyle.Render(item))
				} else {
					lines = append(lines, valueStyle.Render(item))
				}
			}
		}
		return strings.Join(lines, "\n")
	}

	if m.leftCursor == fieldDeploy {
		return ""
	}
	if m.leftCursor == fieldTrust {
		state := "false"
		if m.trust {
			state = "true"
		}
		return labelStyle.Render("Trust") + ": " + valueStyle.Render(state)
	}

	value := m.fieldValue()
	if value == "" {
		value = "(unset)"
	}
	var note string
	switch m.leftCursor {
	case fieldCharm:
		note = "Required. Example: postgresql"
	case fieldAppName:
		note = "Optional. Defaults to charm name"
	case fieldChannel:
		note = "Example: latest/stable"
	case fieldBase:
		note = "Example: ubuntu@22.04"
	case fieldNumUnits:
		note = "Integer"
	case fieldRevision:
		note = "Integer; channel should be set"
	case fieldConstraints:
		note = "Example: cores=2 mem=4G"
	case fieldConfig:
		note = "Comma separated key=value pairs"
	}
	return labelStyle.Render("Value") + ": " + valueStyle.Render(value) + "\n" +
		hintStyle.Render(note)
}

func (m *Modal) renderSimple(background string) string {
	rightW := m.width * 55 / 100
	if rightW < 48 {
		rightW = 48
	}
	if rightW > 84 {
		rightW = 84
	}

	content := m.renderSimplePane(rightW - 2)
	combined := ui.BorderBox(content, "Charm", rightW)
	if m.validationErr != "" {
		errStyle := lipgloss.NewStyle().Foreground(color.Error)
		combined += "\n" + errStyle.Render("  "+m.validationErr)
	}
	combined += "\n" + m.renderSimpleFooter()

	title := " Deploy Charm "
	if m.modelName != "" {
		title = " Deploy Charm(" + m.modelName + ") "
	}
	titleStyle := lipgloss.NewStyle().Foreground(color.Primary).Bold(true)
	box := ui.BorderBoxRawTitle(combined, titleStyle.Render(title), rightW+4)

	modalH := lipgloss.Height(box)
	x := (m.width - (rightW + 4)) / 2
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

func (m *Modal) renderSimplePane(width int) string {
	valueStyle := lipgloss.NewStyle().Foreground(color.Title)
	hintStyle := lipgloss.NewStyle().Foreground(color.Muted)
	selectedStyle := lipgloss.NewStyle().Foreground(color.CrumbFg).Background(color.Highlight).Bold(true)

	lines := []string{}
	if m.editing {
		inputLine := valueStyle.Render(m.input.Value()) + lipgloss.NewStyle().Foreground(color.Primary).Render("█")
		lines = append(lines, inputLine)
	} else {
		v := m.charm
		if strings.TrimSpace(v) == "" {
			v = "(unset)"
		}
		lines = append(lines, valueStyle.Render(v))
	}

	if len(m.autocomplete) > 0 {
		lines = append(lines, hintStyle.Render("Suggestions:"))
		limit := len(m.autocomplete)
		if limit > 8 {
			limit = 8
		}
		for i := 0; i < limit; i++ {
			item := "  " + truncate(m.autocomplete[i], width-2)
			if i == m.autocompleteIndex {
				lines = append(lines, selectedStyle.Render(item))
			} else {
				lines = append(lines, valueStyle.Render(item))
			}
		}
	}

	return strings.Join(lines, "\n")
}

func (m *Modal) renderSimpleFooter() string {
	keyStyle := lipgloss.NewStyle().Foreground(color.HintKey).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(color.HintDesc)
	parts := []string{
		keyStyle.Render("Enter") + descStyle.Render(" search/edit"),
		keyStyle.Render("Tab") + descStyle.Render(" complete"),
		keyStyle.Render("o") + descStyle.Render(" options"),
		keyStyle.Render("Esc") + descStyle.Render(" close"),
		keyStyle.Render("D") + descStyle.Render(" deploy"),
	}
	return strings.Join(parts, descStyle.Render("  •  "))
}

func (m *Modal) renderFooter() string {
	keyStyle := lipgloss.NewStyle().Foreground(color.HintKey).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(color.HintDesc)
	parts := []string{
		keyStyle.Render("↑/↓") + descStyle.Render(" move"),
		keyStyle.Render("→/Enter") + descStyle.Render(" open/edit"),
		keyStyle.Render("Tab") + descStyle.Render(" complete"),
		keyStyle.Render("←/Esc") + descStyle.Render(" back/close"),
		keyStyle.Render("D") + descStyle.Render(" deploy"),
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

func sortedUnique(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		v := strings.TrimSpace(item)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		result = append(result, v)
	}
	sort.Strings(result)
	return result
}
