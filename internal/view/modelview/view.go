// Package modelview implements the split-pane model overview with
// applications on the left and units+relations stacked on the right.
package modelview

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/nav"
	"github.com/bschimke95/jara/internal/ui"
	"github.com/bschimke95/jara/internal/view"
	"github.com/bschimke95/jara/internal/view/relations"
	"github.com/bschimke95/jara/internal/view/units"
)

// New creates a new model overview.
func New(keys ui.KeyMap) *View {
	appCols := applicationColumns()
	at := table.New(
		table.WithColumns(appCols),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	at.SetStyles(ui.StyledTable())

	unitCols := units.CompactColumns()
	ut := table.New(
		table.WithColumns(unitCols),
		table.WithFocused(false),
		table.WithHeight(5),
	)
	ut.SetStyles(ui.UnfocusedTableStyles())

	relCols := relations.CompactColumn()
	rt := table.New(
		table.WithColumns(relCols),
		table.WithFocused(false),
		table.WithHeight(5),
	)
	rt.SetStyles(ui.UnfocusedTableStyles())

	return &View{
		appTable:      at,
		unitTable:     ut,
		relationTable: rt,
		keys:          keys,
		pendingScale:  make(map[string]int),
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
	m.appTable.SetRows(applicationRows(status.Applications))
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

// KeyHints returns the view-specific key hints for the header.
func (m *View) KeyHints() []view.KeyHint {
	bk := func(b key.Binding) string { return b.Help().Key }
	return []view.KeyHint{
		{Key: bk(m.keys.Enter), Desc: "select"},
		{Key: bk(m.keys.UnitsNav), Desc: "units"},
		{Key: bk(m.keys.RelationsNav), Desc: "relations"},
		{Key: bk(m.keys.ScaleUp) + "/" + bk(m.keys.ScaleDown), Desc: "scale"},
		{Key: bk(m.keys.LogsJump), Desc: "logs (app)"},
		{Key: bk(m.keys.LogsView), Desc: "logs"},
	}
}

// NoModelMsg is sent by the status stream when no model is selected on the
// current controller. The view handles it by requesting navigation back.
type NoModelMsg struct{}

func (m *View) Init() tea.Cmd { return nil }

func (m *View) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(NoModelMsg); ok {
		return m, func() tea.Msg { return view.GoBackMsg{} }
	}

	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(msg, m.keys.UnitsNav):
			return m, func() tea.Msg {
				return view.NavigateMsg{Target: nav.UnitsView, Context: m.selectedApp}
			}
		case key.Matches(msg, m.keys.RelationsNav):
			return m, func() tea.Msg {
				return view.NavigateMsg{Target: nav.RelationsView}
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
		case key.Matches(msg, m.keys.LogsView):
			return m, func() tea.Msg {
				return view.NavigateMsg{Target: nav.DebugLogView}
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
		}
	}

	var cmd tea.Cmd
	m.appTable, cmd = m.appTable.Update(msg)
	m.refreshRightPane()
	return m, cmd
}

func (m *View) View() tea.View {
	leftWidth, rightWidth := m.splitWidths()

	leftContent := m.appTable.View()
	leftBox := ui.BorderBox(
		padToHeight(leftContent, m.height-2),
		"Applications",
		leftWidth,
	)

	halfH := (m.height - 4) / 2
	if halfH < 2 {
		halfH = 2
	}

	unitBox := ui.BorderBoxRawTitle(
		padToHeight(m.unitTable.View(), halfH),
		m.rightPaneTitle("U", "nits"),
		rightWidth,
	)
	relBox := ui.BorderBoxRawTitle(
		padToHeight(m.relationTable.View(), halfH),
		m.rightPaneTitle("R", "elations"),
		rightWidth,
	)
	rightBox := lipgloss.JoinVertical(lipgloss.Left, unitBox, relBox)

	combined := lipgloss.JoinHorizontal(lipgloss.Top, leftBox, rightBox)
	return tea.NewView(combined)
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
	m.appTable.SetHeight(m.height - 2)
	m.appTable.SetColumns(ui.ScaleColumns(applicationColumns(), leftInner))

	halfH := (m.height - 4) / 2
	if halfH < 2 {
		halfH = 2
	}

	m.unitTable.SetWidth(rightInner)
	m.unitTable.SetHeight(halfH)
	m.unitTable.SetColumns(ui.ScaleColumns(units.CompactColumns(), rightInner))

	m.relationTable.SetWidth(rightInner)
	m.relationTable.SetHeight(halfH)
	m.relationTable.SetColumns(ui.ScaleColumns(relations.CompactColumn(), rightInner))
}

func (m *View) rightPaneTitle(hotkey, rest string) string {
	keyStyle := lipgloss.NewStyle().Foreground(color.BorderTitle).Bold(true).Underline(true)
	textStyle := lipgloss.NewStyle().Foreground(color.BorderTitle).Bold(true)
	title := " " + keyStyle.Render(hotkey) + textStyle.Render(rest)
	if m.selectedApp != "" {
		title += textStyle.Render("(" + m.selectedApp + ")")
	}
	title += " "
	return title
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
		rows := units.CompactRowsForApp(app)
		if delta := m.pendingScale[appName]; delta != 0 {
			pending := units.PendingCompactRows(appName, app.Units, delta)
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
