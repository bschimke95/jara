package app

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bschimke95/jara/internal/api"
	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/config"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/nav"
	"github.com/bschimke95/jara/internal/ui"
	"github.com/bschimke95/jara/internal/view"
	"github.com/bschimke95/jara/internal/view/applications"
	"github.com/bschimke95/jara/internal/view/controllers"
	"github.com/bschimke95/jara/internal/view/debuglog"
	"github.com/bschimke95/jara/internal/view/machines"
	"github.com/bschimke95/jara/internal/view/models"
	"github.com/bschimke95/jara/internal/view/modelview"
	"github.com/bschimke95/jara/internal/view/relations"
	"github.com/bschimke95/jara/internal/view/units"
)

// Model is the root Bubble Tea model.
type Model struct {
	client api.Client
	cfg    *config.Config
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

// Option configures the root Model.
type Option func(*Model)

// WithTheme applies a resolved theme to the color package globals.
func WithTheme(t *config.Theme) Option {
	return func(_ *Model) {
		if t != nil {
			t.Apply()
		}
	}
}

// WithKeyMap overrides the default key bindings.
func WithKeyMap(km ui.KeyMap) Option {
	return func(m *Model) {
		m.keys = km
	}
}

// WithConfig attaches the full configuration to the model.
func WithConfig(cfg *config.Config) Option {
	return func(m *Model) {
		m.cfg = cfg
	}
}

// New creates the root model.
func New(client api.Client, opts ...Option) Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.CharLimit = 64

	keys := ui.DefaultKeyMap()

	m := Model{
		client: client,
		cfg:    config.NewDefault(),
		stack:  nav.NewStack(nav.ModelView),
		views: map[nav.ViewID]view.View{
			nav.ControllerView:   controllers.New(keys),
			nav.ModelsView:       models.New(keys),
			nav.ModelView:        modelview.New(keys),
			nav.ApplicationsView: applications.New(keys),
			nav.UnitsView:        units.New("", keys),
			nav.MachinesView:     machines.New(keys),
			nav.RelationsView:    relations.New(keys),
			nav.DebugLogView:     debuglog.New(keys),
		},
		keys:  keys,
		input: ti,
		mode:  modeNormal,
	}

	for _, opt := range opts {
		opt(&m)
	}

	return m
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
			if sr, ok := v.(view.StatusReceiver); ok {
				sr.SetStatus(msg.status)
			}
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
			if sr, ok := v.(view.StatusReceiver); ok {
				sr.SetStatus(msg.Status)
			}
		}
		return m, nil

	case controllers.UpdatedMsg:
		if cv, ok := m.views[nav.ControllerView].(*controllers.View); ok {
			cv.SetControllers(msg.Controllers)
		}
		return m, nil

	case models.UpdatedMsg:
		if mv, ok := m.views[nav.ModelsView].(*models.View); ok {
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

	case view.ScaleRequestMsg:
		return m, m.scaleApplication(msg.AppName, msg.Delta)

	case view.NavigateMsg:
		return m.handleNavigate(msg)

	case view.GoBackMsg:
		return m.handleBack()

	case debugLogConnectedMsg:
		// The stream just connected — deliver the first batch and schedule more.
		return m, readNextLogBatch(msg.ctx, msg.ch)

	case debuglog.Msg:
		m2, cmd := m.updateActiveView(msg)
		// Schedule reading the next batch from the stream.
		if msg.Ctx != nil && msg.Ch != nil {
			return m2, tea.Batch(cmd, readNextLogBatch(msg.Ctx, msg.Ch))
		}
		return m2, cmd

	case debuglog.ErrMsg:
		return m.updateActiveView(msg)

	case debuglog.FilterChangedMsg:
		// The user applied a new filter from inside the debug-log view.
		// Restart the stream with the new filter; keep the same view instance.
		return m, m.startDebugLogStream(msg.Filter)
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
	// When the debug-log search bar is visible it occupies 3 rows (bordered box).
	if m.stack.Current().View == nav.DebugLogView {
		if dl, ok := m.views[nav.DebugLogView].(*debuglog.View); ok && dl.IsSearchActive() {
			chrome += 3
		}
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
	controllerName := m.client.ControllerName()
	modelName, cloud, region := "", "", ""
	if m.status != nil {
		modelName = m.status.Model.Name
		cloud = m.status.Model.Cloud
		region = m.status.Model.Region
	}
	// Combine common hints with the active view's own key hints.
	currentView := m.views[m.stack.Current().View]
	bk := func(b key.Binding) string { return b.Help().Key }
	commonHints := []ui.KeyHint{
		{Key: bk(m.keys.Command), Desc: "cmd"},
		{Key: bk(m.keys.Help), Desc: "help"},
		{Key: bk(m.keys.Quit), Desc: "quit"},
	}
	hints := append(currentView.KeyHints(), commonHints...)
	headerInner := ui.HeaderContent(controllerName, modelName, cloud, region, hints, m.width-2)
	sections = append(sections, ui.BorderBox(headerInner, "", m.width))

	// ── Search bar (debug-log only, between header and body) ──
	if m.stack.Current().View == nav.DebugLogView {
		if dl, ok := m.views[nav.DebugLogView].(*debuglog.View); ok && dl.IsSearchActive() {
			sections = append(sections, dl.RenderSearchBar(m.width))
		}
	}

	// ── Body: view content ──
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

		// For the debug-log view, embed the active filter summary in the title.
		var rawTitle string
		if m.stack.Current().View == nav.DebugLogView {
			if dl, ok := m.views[nav.DebugLogView].(*debuglog.View); ok {
				titleStyle := lipgloss.NewStyle().Foreground(color.BorderTitle).Bold(true)
				rawTitle = titleStyle.Render(" "+bodyTitle+" ") + dl.FilterTitle()
			}
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
		if rawTitle != "" {
			sections = append(sections, ui.BorderBoxRawTitle(bodyContent, rawTitle, m.width))
		} else {
			sections = append(sections, ui.BorderBox(bodyContent, bodyTitle, m.width))
		}
	}

	// ── Error line ──
	if m.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(color.Error)
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
