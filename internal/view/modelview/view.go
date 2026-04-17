// Package modelview implements the split-pane model overview with
// applications on the left and units+relations stacked on the right.
package modelview

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/nav"
	"github.com/bschimke95/jara/internal/ui"
	"github.com/bschimke95/jara/internal/view"
	"github.com/bschimke95/jara/internal/view/actionmodal"
	"github.com/bschimke95/jara/internal/view/deploymodal"
	"github.com/bschimke95/jara/internal/view/relatemodal"
	"github.com/bschimke95/jara/internal/view/relations"
	"github.com/bschimke95/jara/internal/view/units"
)

// New creates a new model overview.
func New(keys ui.KeyMap, styles *color.Styles, selectModelFn func(string) error) *View {
	appCols := applicationColumns()
	at := table.New(
		table.WithColumns(appCols),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	at.SetStyles(ui.StyledTableHighlightOnly(styles))

	unitCols := units.CompactColumns()
	ut := table.New(
		table.WithColumns(unitCols),
		table.WithFocused(false),
		table.WithHeight(5),
	)
	ut.SetStyles(ui.UnfocusedTableStyles(styles))

	relCols := relations.CompactColumn()
	rt := table.New(
		table.WithColumns(relCols),
		table.WithFocused(false),
		table.WithHeight(5),
	)
	rt.SetStyles(ui.UnfocusedTableStyles(styles))

	return &View{
		appTable:      at,
		unitTable:     ut,
		relationTable: rt,
		keys:          keys,
		styles:        styles,
		pendingScale:  make(map[string]int),
		selectModelFn: selectModelFn,
	}
}

func (m *View) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.recalcLayout()
}

// SetStatus implements view.StatusReceiver.
func (m *View) SetStatus(status *model.FullStatus) {
	m.status = status
	if status == nil {
		return
	}
	m.rebuildAppRows()
	for appName, delta := range m.pendingScale {
		app, ok := status.Applications[appName]
		if !ok {
			delete(m.pendingScale, appName)
			continue
		}
		if delta < 0 {
			if len(app.Units) <= app.Scale {
				delete(m.pendingScale, appName)
			}
		} else {
			if len(app.Units) >= app.Scale {
				delete(m.pendingScale, appName)
			}
		}
	}
	m.refreshRightPane()
}

// SetFilter implements view.Filterable.
func (m *View) SetFilter(filter string) {
	m.filterStr = filter
	m.rebuildAppRows()
	m.refreshRightPane()
}

func (m *View) rebuildAppRows() {
	if m.status == nil {
		return
	}
	allRows := applicationRows(m.status.Applications, m.styles)
	m.appTable.SetRows(view.FilterRows(allRows, 0, m.filterStr, m.styles.SearchHighlight))
}

// SetCharmSuggestions stores external charm suggestions for deploy modal.
func (m *View) SetCharmSuggestions(names []string) {
	m.charmhubSuggestions = append([]string(nil), names...)
}

// SetCharmEndpoints stores charm endpoint metadata from Charmhub.
func (m *View) SetCharmEndpoints(endpoints map[string]map[string]model.CharmEndpoint) {
	m.charmEndpoints = endpoints
}

// KeyHints returns the view-specific key hints for the header.
func (m *View) KeyHints() []view.KeyHint {
	return []view.KeyHint{
		{Key: view.BindingKey(m.keys.Enter), Desc: "select"},
		{Key: view.BindingKey(m.keys.RunAction), Desc: "action"},
		{Key: view.BindingKey(m.keys.Deploy), Desc: "deploy"},
		{Key: view.BindingKey(m.keys.Relate), Desc: "relate"},
		{Key: view.BindingKey(m.keys.ApplicationsNav), Desc: "apps"},
		{Key: view.BindingKey(m.keys.UnitsNav), Desc: "units"},
		{Key: view.BindingKey(m.keys.RelationsNav), Desc: "relations"},
		{Key: view.BindingKey(m.keys.ScaleUp) + "/" + view.BindingKey(m.keys.ScaleDown), Desc: "scale"},
		{Key: view.BindingKey(m.keys.LogsJump), Desc: "logs (app)"},
	}
}

// CopySelection implements view.Copyable.
func (m *View) CopySelection() string {
	return view.CopySelectedRow(m.appTable)
}

// NoModelMsg is sent by the status stream when no model is selected on the
// current controller. The view handles it by requesting navigation back.
type NoModelMsg struct{}

func (m *View) Init() tea.Cmd { return nil }

func (m *View) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Action modal takes priority when open.
	if m.actionModalOpen {
		switch msg.(type) {
		case actionmodal.CloseMsg:
			m.actionModalOpen = false
			m.actionModal = nil
			return m, nil
		default:
			var cmd tea.Cmd
			newModel, cmd := m.actionModal.Update(msg)
			m.actionModal = newModel.(*actionmodal.Modal)
			return m, cmd
		}
	}

	if m.deployModalOpen {
		switch msg := msg.(type) {
		case deploymodal.AppliedMsg:
			m.deployModalOpen = false
			return m, func() tea.Msg {
				return view.DeployRequestMsg{ModelName: msg.ModelName, Options: msg.Options}
			}
		case deploymodal.ClosedMsg:
			m.deployModalOpen = false
			return m, nil
		default:
			updated, cmd := m.deployModal.Update(msg)
			if dm, ok := updated.(*deploymodal.Modal); ok {
				m.deployModal = *dm
			}
			return m, cmd
		}
	}

	if m.relateModalOpen {
		switch msg := msg.(type) {
		case relatemodal.AppliedMsg:
			m.relateModalOpen = false
			return m, func() tea.Msg {
				return view.RelateRequestMsg{EndpointA: msg.EndpointA, EndpointB: msg.EndpointB}
			}
		case relatemodal.ClosedMsg:
			m.relateModalOpen = false
			return m, nil
		default:
			updated, cmd := m.relateModal.Update(msg)
			if rm, ok := updated.(*relatemodal.Modal); ok {
				m.relateModal = *rm
			}
			return m, cmd
		}
	}

	if _, ok := msg.(NoModelMsg); ok {
		return m, func() tea.Msg { return view.GoBackMsg{} }
	}

	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(msg, m.keys.ApplicationsNav):
			return m, func() tea.Msg {
				return view.NavigateMsg{Target: nav.ApplicationsView}
			}
		case key.Matches(msg, m.keys.Deploy):
			m.deployModal = deploymodal.New("", m.keys, m.styles, m.charmSuggestions(), m.applicationSuggestions())
			m.deployModal.SetSize(m.width, m.height)
			m.deployModalOpen = true
			return m, m.deployModal.BeginCharmEdit()
		case key.Matches(msg, m.keys.Relate):
			suggestions := relatemodal.BuildSuggestions(m.status, m.charmEndpoints)
			var rels []model.Relation
			if m.status != nil {
				rels = m.status.Relations
			}
			m.relateModal = relatemodal.New(m.keys, m.styles, suggestions, rels)
			m.relateModal.SetSize(m.width, m.height)
			m.relateModalOpen = true
			return m, m.relateModal.BeginEdit()
		case key.Matches(msg, m.keys.UnitsNav):
			return m, func() tea.Msg {
				return view.NavigateMsg{Target: nav.UnitsView, Context: m.selectedApp}
			}
		case key.Matches(msg, m.keys.RelationsNav):
			return m, func() tea.Msg {
				return view.NavigateMsg{Target: nav.RelationsView}
			}
		case key.Matches(msg, m.keys.MachinesNav):
			return m, func() tea.Msg {
				return view.NavigateMsg{Target: nav.MachinesView}
			}
		case key.Matches(msg, m.keys.LogsJump):
			var filter *model.DebugLogFilter
			if m.selectedApp != "" {
				f := model.DebugLogFilter{Applications: []string{m.selectedApp}}
				filter = &f
			}
			return m, func() tea.Msg {
				return view.NavigateMsg{Target: nav.DebugLogView, Filter: filter}
			}
		case key.Matches(msg, m.keys.ScaleUp):
			if m.selectedApp != "" {
				app := m.selectedApp
				m.pendingScale[app]++
				m.refreshRightPane()
				return m, func() tea.Msg { return view.ScaleRequestMsg{AppName: app, Delta: 1} }
			}
		case key.Matches(msg, m.keys.ScaleDown):
			if m.selectedApp != "" {
				app := m.selectedApp
				m.pendingScale[app]--
				m.refreshRightPane()
				return m, func() tea.Msg { return view.ScaleRequestMsg{AppName: app, Delta: -1} }
			}
		case key.Matches(msg, m.keys.RunAction):
			if m.selectedApp != "" && m.status != nil {
				if app, ok := m.status.Applications[m.selectedApp]; ok {
					unitNames := view.UnitNamesLeaderFirst(app)
					if len(unitNames) > 0 {
						am := actionmodal.NewWithUnits(m.selectedApp, unitNames, m.keys, m.styles)
						am.SetSize(m.width, m.height)
						m.actionModal = am
						m.actionModalOpen = true
						return m, am.Init()
					}
				}
			}
		}
	}

	var cmd tea.Cmd
	m.appTable, cmd = m.appTable.Update(msg)
	m.refreshRightPane()
	return m, cmd
}

func (m *View) View() tea.View {
	background := m.renderBackground()
	if m.actionModalOpen && m.actionModal != nil {
		return tea.NewView(m.actionModal.Render(background))
	}
	if m.deployModalOpen {
		return tea.NewView(m.deployModal.Render(background))
	}
	if m.relateModalOpen {
		return tea.NewView(m.relateModal.Render(background))
	}
	return tea.NewView(background)
}

func (m *View) renderBackground() string {
	leftWidth, rightWidth := m.splitWidths()

	leftContent := m.appTableView()
	leftBox := ui.BorderBoxRawTitle(
		view.PadToHeight(leftContent, m.height-2),
		m.leftPaneTitle("A", "pplications"),
		leftWidth,
		m.styles,
	)

	halfH := (m.height - 4) / 2
	if halfH < 2 {
		halfH = 2
	}

	unitBox := ui.BorderBoxRawTitle(
		view.PadToHeight(m.unitTable.View(), halfH),
		m.rightPaneTitle("U", "nits"),
		rightWidth,
		m.styles,
	)
	relBox := ui.BorderBoxRawTitle(
		view.PadToHeight(m.relationTable.View(), halfH),
		m.rightPaneTitle("R", "elations"),
		rightWidth,
		m.styles,
	)
	rightBox := lipgloss.JoinVertical(lipgloss.Left, unitBox, relBox)

	combined := lipgloss.JoinHorizontal(lipgloss.Top, leftBox, rightBox)
	return combined
}

func (m *View) appTableView() string {
	cursor := m.appTable.Cursor()
	rows := m.appTable.Rows()
	if cursor >= 0 && cursor < len(rows) {
		original := rows[cursor]
		stripped := make(table.Row, len(original))
		for i, cell := range original {
			stripped[i] = ansi.Strip(cell)
		}
		if len(stripped) > 1 {
			stripped[1] = m.styles.StatusText(stripped[1])
		}
		rows[cursor] = stripped
		m.appTable.SetRows(rows)
		defer func() {
			rows[cursor] = original
			m.appTable.SetRows(rows)
		}()
	}
	return m.appTable.View()
}

func (m *View) splitWidths() (int, int) {
	left := m.width * 60 / 100
	if left < 30 {
		left = 30
	}
	right := m.width - left
	if right < 20 {
		right = 20
	}
	return left, right
}

func (m *View) recalcLayout() {
	leftWidth, rightWidth := m.splitWidths()

	leftInner := leftWidth - 2
	rightInner := rightWidth - 2

	m.appTable.SetWidth(leftInner)
	m.appTable.SetHeight(m.height - 3) // -2 for border, -1 for header row
	m.appTable.SetColumns(ui.ScaleColumns(applicationColumns(), leftInner))

	halfH := (m.height - 4) / 2
	if halfH < 2 {
		halfH = 2
	}

	m.unitTable.SetWidth(rightInner)
	m.unitTable.SetHeight(halfH - 1) // -1 for header row
	m.unitTable.SetColumns(ui.ScaleColumns(units.CompactColumns(), rightInner))

	m.relationTable.SetWidth(rightInner)
	m.relationTable.SetHeight(halfH - 1) // -1 for header row
	m.relationTable.SetColumns(ui.ScaleColumns(relations.CompactColumn(), rightInner))
}

func (m *View) rightPaneTitle(hotkey, rest string) string {
	keyStyle := lipgloss.NewStyle().Foreground(m.styles.BorderTitleColor).Bold(true).Underline(true)
	textStyle := lipgloss.NewStyle().Foreground(m.styles.BorderTitleColor).Bold(true)
	title := " " + keyStyle.Render(hotkey) + textStyle.Render(rest)
	if m.selectedApp != "" {
		title += textStyle.Render("(" + m.selectedApp + ")")
	}
	title += " "
	return title
}

func (m *View) leftPaneTitle(hotkey, rest string) string {
	keyStyle := lipgloss.NewStyle().Foreground(m.styles.BorderTitleColor).Bold(true).Underline(true)
	textStyle := lipgloss.NewStyle().Foreground(m.styles.BorderTitleColor).Bold(true)
	return " " + keyStyle.Render(hotkey) + textStyle.Render(rest) + " "
}

func (m *View) refreshRightPane() {
	if m.status == nil {
		return
	}

	row := m.appTable.SelectedRow()
	if row == nil {
		m.selectedApp = ""
		m.unitTable.SetRows(nil)
		m.relationTable.SetRows(nil)
		return
	}

	appName := row[0]
	m.selectedApp = appName

	if app, ok := m.status.Applications[appName]; ok {
		rows := units.CompactRowsForApp(app, m.styles)
		if delta := m.pendingScale[appName]; delta != 0 {
			pending := units.PendingCompactRows(appName, app.Units, delta, m.styles)
			if delta < 0 {
				tail := len(rows) - len(pending)
				if tail < 0 {
					tail = 0
				}
				rows = append(rows[:tail], pending...)
			} else {
				rows = append(rows, pending...)
			}
		}
		m.unitTable.SetRows(rows)
	} else {
		m.unitTable.SetRows(nil)
	}

	m.relationTable.SetRows(relations.CompactRowsForApp(m.status.Relations, appName))
}

func (m *View) charmSuggestions() []string {
	out := append([]string(nil), m.charmhubSuggestions...)
	if m.status == nil {
		return out
	}
	for _, appName := range ui.SortedKeys(m.status.Applications) {
		app := m.status.Applications[appName]
		if app.Charm != "" {
			out = append(out, app.Charm)
		}
	}
	return out
}

func (m *View) applicationSuggestions() []string {
	if m.status == nil {
		return nil
	}
	out := make([]string, 0, len(m.status.Applications))
	out = append(out, ui.SortedKeys(m.status.Applications)...)
	return out
}

func (m *View) Enter(ctx view.NavigateContext) (tea.Cmd, error) {
	if ctx.Context != "" {
		if err := m.selectModelFn(ctx.Context); err != nil {
			return nil, err
		}
		return tea.Batch(
			func() tea.Msg { return view.ClearStatusMsg{} },
			func() tea.Msg { return view.StartStatusStreamMsg{} },
		), nil
	}
	return nil, nil
}

func (m *View) Leave() tea.Cmd {
	return func() tea.Msg { return view.StopStatusStreamMsg{} }
}
