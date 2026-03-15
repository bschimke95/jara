package view

import (
	"strings"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/nav"
	"github.com/bschimke95/jara/internal/render"
	"github.com/bschimke95/jara/internal/ui"
)

// ModelView is the split-pane model overview with applications on the left
// and units+relations stacked on the right.
type ModelView struct {
	appTable      table.Model
	unitTable     table.Model
	relationTable table.Model

	keys   ui.KeyMap
	status *model.FullStatus

	width        int
	height       int
	selectedApp  string
	pendingScale map[string]int // net pending unit delta per app, cleared when status catches up
}

// NewModelView creates a new model overview.
func NewModelView() *ModelView {
	appCols := render.ApplicationColumns()
	at := table.New(
		table.WithColumns(appCols),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	at.SetStyles(styledTable())

	unitCols := render.UnitColumns()
	ut := table.New(
		table.WithColumns(unitCols),
		table.WithFocused(false),
		table.WithHeight(5),
	)
	ut.SetStyles(unfocusedTableStyles())

	relCols := render.ModelViewRelationColumn()
	rt := table.New(
		table.WithColumns(relCols),
		table.WithFocused(false),
		table.WithHeight(5),
	)
	rt.SetStyles(unfocusedTableStyles())

	return &ModelView{
		appTable:      at,
		unitTable:     ut,
		relationTable: rt,
		keys:          ui.DefaultKeyMap(),
		pendingScale:  make(map[string]int),
	}
}

func (m *ModelView) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.recalcLayout()
}

func (m *ModelView) SetStatus(status *model.FullStatus) {
	m.status = status
	if status == nil {
		return
	}
	m.appTable.SetRows(render.ApplicationRows(status.Applications))
	// Reconcile pending scale: clear entries where live unit count has caught up.
	// app.Scale is the desired (post-request) value; compare unit count against it.
	for appName, delta := range m.pendingScale {
		app, ok := status.Applications[appName]
		if !ok {
			delete(m.pendingScale, appName)
			continue
		}
		if delta < 0 {
			// Scaling down: clear when live unit count has dropped to desired scale.
			if len(app.Units) <= app.Scale {
				delete(m.pendingScale, appName)
			}
		} else {
			// Scaling up: clear when live unit count has reached desired scale.
			if len(app.Units) >= app.Scale {
				delete(m.pendingScale, appName)
			}
		}
	}
	m.refreshRightPane()
}

func (m *ModelView) Init() tea.Cmd { return nil }

func (m *ModelView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch msg.String() {
		case "U":
			return m, func() tea.Msg {
				return NavigateMsg{Target: nav.UnitsView, Context: m.selectedApp}
			}
		case "R":
			return m, func() tea.Msg {
				return NavigateMsg{Target: nav.RelationsView}
			}
		case "+":
			if m.selectedApp != "" {
				app := m.selectedApp
				m.pendingScale[app]++
				m.refreshRightPane()
				return m, func() tea.Msg { return ScaleRequestMsg{AppName: app, Delta: 1} }
			}
		case "-":
			if m.selectedApp != "" {
				app := m.selectedApp
				m.pendingScale[app]--
				m.refreshRightPane()
				return m, func() tea.Msg { return ScaleRequestMsg{AppName: app, Delta: -1} }
			}
		}
	}

	var cmd tea.Cmd
	m.appTable, cmd = m.appTable.Update(msg)
	m.refreshRightPane()
	return m, cmd
}

func (m *ModelView) View() tea.View {
	leftWidth, rightWidth := m.splitWidths()

	// ── Left panel: applications ──
	leftContent := m.appTable.View()
	leftBox := ui.BorderBox(
		padToHeight(leftContent, m.height-2),
		"Applications",
		leftWidth,
	)

	// ── Right panel: units (top half) + relations (bottom half) ──
	halfH := (m.height - 4) / 2 // -4 accounts for the two inner border pairs
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

// splitWidths returns the widths for the left and right panels.
func (m *ModelView) splitWidths() (int, int) {
	// Give 60% to left (apps), 40% to right (details).
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

// recalcLayout updates table dimensions and column widths after a resize.
func (m *ModelView) recalcLayout() {
	leftWidth, rightWidth := m.splitWidths()

	// Inner width = panel width minus border chars (2).
	leftInner := leftWidth - 2
	rightInner := rightWidth - 2

	m.appTable.SetWidth(leftInner)
	m.appTable.SetHeight(m.height - 2) // minus top/bottom border
	m.appTable.SetColumns(render.ScaleColumns(render.ApplicationColumns(), leftInner))

	halfH := (m.height - 4) / 2 // each box has its own 2-line border overhead
	if halfH < 2 {
		halfH = 2
	}

	m.unitTable.SetWidth(rightInner)
	m.unitTable.SetHeight(halfH)
	m.unitTable.SetColumns(render.ScaleColumns(render.UnitColumns(), rightInner))

	m.relationTable.SetWidth(rightInner)
	m.relationTable.SetHeight(halfH)
	m.relationTable.SetColumns(render.ScaleColumns(render.ModelViewRelationColumn(), rightInner))
}

// rightPaneTitle builds a pre-rendered title for a right panel section.
// The hotkey letter is rendered bold; the rest follows in normal title style.
// If an app is selected its name is appended.
func (m *ModelView) rightPaneTitle(hotkey, rest string) string {
	keyStyle := lipgloss.NewStyle().Foreground(color.BorderTitle).Bold(true).Underline(true)
	textStyle := lipgloss.NewStyle().Foreground(color.BorderTitle).Bold(true)
	title := " " + keyStyle.Render(hotkey) + textStyle.Render(rest)
	if m.selectedApp != "" {
		title += textStyle.Render("(" + m.selectedApp + ")")
	}
	title += " "
	return title
}

// refreshRightPane updates units and relations for the currently selected application.
func (m *ModelView) refreshRightPane() {
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

	// Units for the selected application.
	if app, ok := m.status.Applications[appName]; ok {
		rows := render.UnitRowsForApp(app)
		if delta := m.pendingScale[appName]; delta != 0 {
			pending := render.PendingUnitRows(appName, app.Units, delta)
			if delta < 0 {
				// pending rows replace the last len(pending) live rows
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

	// Relations involving the selected application.
	m.relationTable.SetRows(render.ModelViewRelationRowsForApp(m.status.Relations, appName))
}

// padToHeight pads content with blank lines so it fills the given height.
func padToHeight(content string, height int) string {
	lines := strings.Split(content, "\n")
	for len(lines) < height {
		lines = append(lines, "")
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}

// unfocusedTableStyles returns dimmed table styles for inactive panes.
// The selected style is intentionally identical to the cell style so that
// the cursor row carries no highlight in a non-interactive pane.
func unfocusedTableStyles() table.Styles {
	s := table.DefaultStyles()
	s.Header = lipgloss.NewStyle().
		Bold(true).
		Foreground(color.Muted).
		Padding(0, 1)
	s.Selected = lipgloss.NewStyle().
		Foreground(color.Title)
	s.Cell = lipgloss.NewStyle().
		Padding(0, 1)
	return s
}
