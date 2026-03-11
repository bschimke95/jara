package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
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

type inputMode int

const (
	modeNormal inputMode = iota
	modeCommand
	modeFilter
)

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

	logCancel context.CancelFunc // cancels the debug-log stream

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
	return tea.Batch(m.pollStatus(), m.pollControllers(), tea.RequestWindowSize)
}

func (m Model) pollStatus() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		status, err := m.client.Status(ctx)
		if err != nil {
			if strings.Contains(err.Error(), "No selected model") {
				return noModelMsg{}
			}
			return errMsg{err}
		}
		return view.StatusUpdatedMsg{Status: status}
	}
}

func (m Model) pollControllers() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		controllers, err := m.client.Controllers(ctx)
		if err != nil {
			return errMsg{err}
		}
		return view.ControllersUpdatedMsg{Controllers: controllers}
	}
}

func (m Model) pollModels(controllerName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		models, err := m.client.Models(ctx, controllerName)
		if err != nil {
			return errMsg{err}
		}
		return view.ModelsUpdatedMsg{Models: models}
	}
}

func (m Model) scheduleNextPoll() tea.Cmd {
	return tea.Tick(pollInterval, func(_ time.Time) tea.Msg {
		return pollTickMsg{}
	})
}

type pollTickMsg struct{}
type errMsg struct{ err error }
type noModelMsg struct{} // sent when no model is selected in the current controller

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

	case view.StatusUpdatedMsg:
		m.status = msg.Status
		for _, v := range m.views {
			v.SetStatus(msg.Status)
		}
		return m, m.scheduleNextPoll()

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

	case pollTickMsg:
		return m, m.pollStatus()

	case noModelMsg:
		// No model selected on this controller — pop back to controller view
		// and show a helpful hint.
		m.stack.Pop()
		m.err = fmt.Errorf("no model selected — use \"juju add-model <name>\" to create one")
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, m.scheduleNextPoll()

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
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Back):
			return m.handleBack()
		case key.Matches(msg, m.keys.Command):
			return m.enterCommandMode()
		case key.Matches(msg, m.keys.Filter):
			return m.enterFilterMode()
		}
	}

	return m.updateActiveView(msg)
}

func (m Model) handleNavigate(msg view.NavigateMsg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Selecting a controller from the ControllerView: switch to it and show its models.
	if msg.Target == nav.ModelsView && msg.Context != "" {
		if jc, ok := m.client.(*api.JujuClient); ok {
			if err := jc.SelectController(msg.Context); err != nil {
				m.err = err
				return m, nil
			}
		}
		m.err = nil
		// Reset the models view so we start fresh.
		mv := view.NewModels()
		mv.SetSize(m.width, m.contentHeight())
		m.views[nav.ModelsView] = mv
		m.stack.Push(nav.StackEntry{View: nav.ModelsView, Context: msg.Context})
		return m, m.pollModels(msg.Context)
	}

	// Selecting a model from the ModelsView: switch to it and show the model detail.
	if msg.Target == nav.ModelView && msg.Context != "" {
		if jc, ok := m.client.(*api.JujuClient); ok {
			if err := jc.SelectModel(msg.Context); err != nil {
				m.err = err
				return m, nil
			}
		}
		m.status = nil
		m.err = nil
		for _, v := range m.views {
			v.SetStatus(nil)
		}
		m.stack.Push(nav.StackEntry{View: nav.ModelView})
		return m, tea.Batch(m.pollStatus(), m.pollControllers())
	}

	m.stack.Push(nav.StackEntry{View: msg.Target, Context: msg.Context})

	if msg.Target == nav.UnitsView && msg.Context != "" {
		uv := view.NewUnits(msg.Context)
		uv.SetSize(m.width, m.contentHeight())
		if m.status != nil {
			uv.SetStatus(m.status)
		}
		m.views[nav.UnitsView] = uv
	}

	if msg.Target == nav.DebugLogView {
		// Reset the view and start streaming.
		dl := view.NewDebugLog()
		dl.SetSize(m.width, m.contentHeight())
		m.views[nav.DebugLogView] = dl
		cmds = append(cmds, m.startDebugLogStream())
	}

	return m, tea.Batch(cmds...)
}

func (m Model) handleBack() (Model, tea.Cmd) {
	prev := m.stack.Current()
	if _, ok := m.stack.Pop(); ok {
		// Stop debug-log stream when leaving that view.
		if prev.View == nav.DebugLogView {
			m.stopDebugLogStream()
		}
		current := m.stack.Current()
		if current.View == nav.UnitsView && current.Context == "" {
			uv := view.NewUnits("")
			uv.SetSize(m.width, m.contentHeight())
			if m.status != nil {
				uv.SetStatus(m.status)
			}
			m.views[nav.UnitsView] = uv
		}
	}
	return m, nil
}

func (m Model) enterCommandMode() (Model, tea.Cmd) {
	m.mode = modeCommand
	m.input.Prompt = ":"
	m.input.SetValue("")
	return m, m.input.Focus()
}

func (m Model) enterFilterMode() (Model, tea.Cmd) {
	m.mode = modeFilter
	m.input.Prompt = "/"
	m.input.SetValue(m.filterStr)
	return m, m.input.Focus()
}

func (m Model) updateInput(msg tea.Msg) (Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch msg.String() {
		case "enter":
			value := m.input.Value()
			if m.mode == modeCommand {
				m.mode = modeNormal
				m.input.Blur()
				return m.executeCommand(value)
			}
			m.filterStr = value
			m.mode = modeNormal
			m.input.Blur()
			return m, nil

		case "esc":
			if m.mode == modeFilter {
				m.filterStr = ""
			}
			m.mode = modeNormal
			m.input.Blur()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) executeCommand(cmd string) (Model, tea.Cmd) {
	cmd = strings.TrimSpace(strings.ToLower(cmd))
	if cmd == "q" || cmd == "quit" {
		return m, tea.Quit
	}
	if viewID, ok := nav.ResolveCommand(cmd); ok {
		return m.handleNavigate(view.NavigateMsg{Target: viewID})
	}
	return m, nil
}

func (m Model) updateActiveView(msg tea.Msg) (Model, tea.Cmd) {
	currentView := m.views[m.stack.Current().View]
	updated, cmd := currentView.Update(msg)
	m.views[m.stack.Current().View] = updated.(view.View)
	return m, cmd
}

// debugLogConnectedMsg is sent when the debug-log stream is established.
type debugLogConnectedMsg struct {
	ctx context.Context
	ch  <-chan model.LogEntry
}

// startDebugLogStream begins streaming debug-log entries from the API.
// It returns a Cmd that connects and sends a debugLogConnectedMsg.
func (m *Model) startDebugLogStream() tea.Cmd {
	m.stopDebugLogStream() // cancel any existing stream

	ctx, cancel := context.WithCancel(context.Background())
	m.logCancel = cancel

	client := m.client
	return func() tea.Msg {
		ch, err := client.DebugLog(ctx)
		if err != nil {
			return view.DebugLogErrMsg{Err: err}
		}
		return debugLogConnectedMsg{ctx: ctx, ch: ch}
	}
}

// stopDebugLogStream cancels the active debug-log context, if any.
func (m *Model) stopDebugLogStream() {
	if m.logCancel != nil {
		m.logCancel()
		m.logCancel = nil
	}
}

// readNextLogBatch returns a Cmd that reads the next batch from the
// debug-log stream. The context and channel are passed through closures
// rather than stored on the model.
func readNextLogBatch(ctx context.Context, ch <-chan model.LogEntry) tea.Cmd {
	return func() tea.Msg {
		return readDebugLogBatch(ctx, ch)
	}
}

// readDebugLogBatch reads available log entries from the channel,
// batching them together before delivering to the view.
func readDebugLogBatch(ctx context.Context, ch <-chan model.LogEntry) tea.Msg {
	// Block on the first entry.
	select {
	case <-ctx.Done():
		return nil
	case entry, ok := <-ch:
		if !ok {
			return view.DebugLogErrMsg{Err: fmt.Errorf("log stream closed")}
		}
		batch := []model.LogEntry{entry}
		// Drain any additional immediately-available entries.
	drain:
		for {
			select {
			case e, ok := <-ch:
				if !ok {
					break drain
				}
				batch = append(batch, e)
				if len(batch) >= 50 {
					break drain
				}
			default:
				break drain
			}
		}
		return view.DebugLogMsg{Entries: batch, Ctx: ctx, Ch: ch}
	}
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
		if current.Context != "" {
			return "Units"
		}
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
