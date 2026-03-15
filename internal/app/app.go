package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bschimke95/jara/internal/api"
	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/nav"
	"github.com/bschimke95/jara/internal/ui"
	"github.com/bschimke95/jara/internal/view"
)

const pollInterval = 3 * time.Second

// Model is the root Bubble Tea model.
type Model struct {
	client api.Client
	status *model.FullStatus

	stack *nav.Stack
	views map[nav.ViewID]view.View

	mode      inputMode
	input     textinput.Model
	filterStr string

	keys   ui.KeyMap
	width  int
	height int

	statusCancel context.CancelFunc      // cancels the status stream
	statusCh     <-chan api.StatusUpdate // receives status snapshots
	logCancel    context.CancelFunc      // cancels the debug-log stream

	err   error
	ready bool
}

// New creates the root model.
func New(client api.Client) Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.CharLimit = 64

	return Model{
		client: client,
		stack:  nav.NewStack(nav.ModelView),
		views: map[nav.ViewID]view.View{
			nav.ControllerView:   view.NewControllers(),
			nav.ModelsView:       view.NewModels(),
			nav.ModelView:        view.NewModelView(),
			nav.ApplicationsView: view.NewApplications(),
			nav.UnitsView:        view.NewUnits(""),
			nav.MachinesView:     view.NewMachines(),
			nav.RelationsView:    view.NewRelations(),
			nav.DebugLogView:     view.NewDebugLog(),
		},
		keys:  ui.DefaultKeyMap(),
		input: ti,
		mode:  modeNormal,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.startStatusStream(), m.pollControllers(), tea.RequestWindowSize)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		ch := m.contentHeight()
		for _, v := range m.views {
			v.SetSize(m.width, ch)
		}
		return m, nil

	case statusStreamConnectedMsg:
		// The status stream just connected — store the channel and read the first update.
		m.statusCh = msg.ch
		return m, readNextStatus(msg.ctx, msg.ch)

	case statusStreamUpdateMsg:
		m.status = msg.status
		m.err = nil
		for _, v := range m.views {
			v.SetStatus(msg.status)
		}
		return m, readNextStatus(msg.ctx, msg.ch)

	case statusStreamErrMsg:
		m.err = msg.err
		// The stream is still alive; keep reading for the next update.
		return m, readNextStatus(msg.ctx, msg.ch)

	case view.StatusUpdatedMsg:
		// Legacy path kept for views that emit this directly.
		m.status = msg.Status
		for _, v := range m.views {
			v.SetStatus(msg.Status)
		}
		return m, nil

	case view.ControllersUpdatedMsg:
		if cv, ok := m.views[nav.ControllerView].(*view.Controllers); ok {
			cv.SetControllers(msg.Controllers)
		}
		return m, nil

	case view.ModelsUpdatedMsg:
		if mv, ok := m.views[nav.ModelsView].(*view.Models); ok {
			mv.SetModels(msg.Models)
		}
		return m, nil

	case noModelMsg:
		// No model selected on this controller — pop back to controller view
		// and show a helpful hint.
		m.stopStatusStream()
		m.stack.Pop()
		m.err = fmt.Errorf("no model selected — use \"juju add-model <name>\" to create one")
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil

	case view.NavigateMsg:
		return m.handleNavigate(msg)

	case view.GoBackMsg:
		return m.handleBack()

	case debugLogConnectedMsg:
		// The stream just connected — deliver the first batch and schedule more.
		return m, readNextLogBatch(msg.ctx, msg.ch)

	case view.DebugLogMsg:
		m2, cmd := m.updateActiveView(msg)
		// Schedule reading the next batch from the stream.
		if msg.Ctx != nil && msg.Ch != nil {
			return m2, tea.Batch(cmd, readNextLogBatch(msg.Ctx, msg.Ch))
		}
		return m2, cmd

	case view.DebugLogErrMsg:
		return m.updateActiveView(msg)
	}

	if m.mode != modeNormal {
		return m.updateInput(msg)
	}

	if msg, ok := msg.(tea.KeyPressMsg); ok {
		if m2, cmd, handled := m.handleGlobalKeys(msg); handled {
			return m2, cmd
		}
	}

	return m.updateActiveView(msg)
}

func (m Model) updateActiveView(msg tea.Msg) (Model, tea.Cmd) {
	currentView := m.views[m.stack.Current().View]
	updated, cmd := currentView.Update(msg)
	m.views[m.stack.Current().View] = updated.(view.View)
	return m, cmd
}

func (m Model) contentHeight() int {
	// Layout: header box (logo height + 2 borders) + body box borders (2)
	// + optional error (0) + optional input (0-1)
	logoHeight := ui.LogoHeight()
	headerHeight := logoHeight + 2 // +2 for top and bottom borders
	chrome := headerHeight + 2     // +2 for body box borders
	if m.mode != modeNormal {
		chrome++
	}
	h := m.height - chrome
	if h < 5 {
		h = 5
	}
	return h
}

// viewName returns the display name for the current view.
func (m Model) viewName() string {
	current := m.stack.Current()
	switch current.View {
	case nav.ControllerView:
		return "Controllers"
	case nav.ModelsView:
		return "Models"
	case nav.ModelView:
		return "Model"
	case nav.ApplicationsView:
		return "Applications"
	case nav.UnitsView:
		return "Units"
	case nav.MachinesView:
		return "Machines"
	case nav.RelationsView:
		return "Relations"
	case nav.DebugLogView:
		return "Debug Log"
	default:
		return "jara"
	}
}

func (m Model) View() tea.View {
	if !m.ready {
		return tea.NewView("Initializing jara...")
	}

	var sections []string

	// ── Header box: status info (left) + key hints (center) + logo (right) ──
	controllerName := "local"
	if jc, ok := m.client.(*api.JujuClient); ok {
		controllerName = jc.ControllerName()
	}
	modelName, cloud, region := "", "", ""
	if m.status != nil {
		modelName = m.status.Model.Name
		cloud = m.status.Model.Cloud
		region = m.status.Model.Region
	}
	hints := ui.HintsForView(m.viewName())
	headerInner := ui.HeaderContent(controllerName, modelName, cloud, region, hints, m.width-2)
	sections = append(sections, ui.BorderBox(headerInner, "", m.width))

	// ── Body: view content ──
	currentView := m.views[m.stack.Current().View]
	viewOutput := currentView.View()

	// The ModelView renders its own bordered panes, so skip the outer border.
	if m.stack.Current().View == nav.ModelView {
		contentLines := strings.Split(viewOutput.Content, "\n")
		targetH := m.contentHeight() + 2 // no outer border, reclaim those 2 lines
		for len(contentLines) < targetH {
			contentLines = append(contentLines, "")
		}
		if len(contentLines) > targetH {
			contentLines = contentLines[:targetH]
		}
		sections = append(sections, strings.Join(contentLines, "\n"))
	} else {
		// Build the body title (view name + optional context).
		bodyTitle := m.viewName()
		if ctx := m.stack.Current().Context; ctx != "" {
			bodyTitle += "(" + ctx + ")"
		}

		// Pad or truncate content to fill available height.
		contentLines := strings.Split(viewOutput.Content, "\n")
		targetH := m.contentHeight()
		for len(contentLines) < targetH {
			contentLines = append(contentLines, "")
		}
		if len(contentLines) > targetH {
			contentLines = contentLines[:targetH]
		}
		bodyContent := strings.Join(contentLines, "\n")
		sections = append(sections, ui.BorderBox(bodyContent, bodyTitle, m.width))
	}

	// ── Error line ──
	if m.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000"))
		sections = append(sections, errStyle.Render(" Error: "+m.err.Error()))
	}

	// ── Input bar (command/filter mode) ──
	if m.mode != modeNormal {
		promptStyle := lipgloss.NewStyle().Foreground(color.Primary).Bold(true)
		valueStyle := lipgloss.NewStyle().Foreground(color.Title)
		prompt := m.input.Prompt
		val := m.input.Value()
		cursor := lipgloss.NewStyle().
			Foreground(color.Primary).
			Render("█")
		sections = append(sections, fmt.Sprintf(" %s%s%s", promptStyle.Render(prompt), valueStyle.Render(val), cursor))
	}

	body := lipgloss.JoinVertical(lipgloss.Left, sections...)
	return tea.NewView(body)
}
