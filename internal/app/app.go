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

	jaraVersion string

	stack *nav.Stack
	views map[nav.ViewID]view.View

	mode               inputMode
	input              textinput.Model
	filterStr          string
	suggestions        []nav.CommandMatch
	selectedSuggestion int

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

// WithVersion sets the jara version string displayed in the header.
func WithVersion(v string) Option {
	return func(m *Model) {
		m.jaraVersion = v
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

	m.views[nav.ControllerView] = controllers.New(keys, func() tea.Cmd { return m.pollControllers() })
	m.views[nav.ModelsView] = models.New(keys,
		func(ctrl string) tea.Cmd { return m.pollModels(ctrl) },
		func(name string) error { return m.client.SelectController(name) },
		func() string { return m.client.ControllerName() },
	)
	m.views[nav.ModelView] = modelview.New(keys,
		func(name string) error { return m.client.SelectModel(name) },
	)

	for _, opt := range opts {
		opt(&m)
	}

	return m
}

// startupMsg is sent once by Init to trigger stream setup inside Update,
// where model mutations (like storing the cancel func) are preserved.
type startupMsg struct{}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		func() tea.Msg { return startupMsg{} },
		tea.RequestWindowSize,
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	// ── Infrastructure: stream lifecycle & window management ──
	// These are internal to the app and never reach views.
	case startupMsg:
		return m, tea.Batch(m.startStatusStream(), m.pollCharmhubSuggestions())

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
		if msg.ctx.Err() != nil {
			return m, nil
		}
		m.statusCh = msg.ch
		return m, readNextStatus(msg.ctx, msg.ch)

	case statusStreamUpdateMsg:
		// Discard updates from a cancelled stream (e.g. after switching models).
		if msg.ctx.Err() != nil {
			return m, nil
		}
		m.status = msg.status
		m.err = nil
		for _, v := range m.views {
			if sr, ok := v.(view.StatusReceiver); ok {
				sr.SetStatus(msg.status)
			}
		}
		return m, readNextStatus(msg.ctx, msg.ch)

	case statusStreamErrMsg:
		if msg.ctx.Err() != nil {
			return m, nil
		}
		m.err = msg.err
		return m, readNextStatus(msg.ctx, msg.ch)

	case debugLogConnectedMsg:
		return m, readFirstLogBatch(msg.ctx, msg.ch)

	case charmhubSuggestionsMsg:
		for _, v := range m.views {
			if sr, ok := v.(view.CharmSuggestionReceiver); ok {
				sr.SetCharmSuggestions(msg.Names)
			}
		}
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil

	case view.StopStatusStreamMsg:
		m.stopStatusStream()
		return m, nil

	case view.StartStatusStreamMsg:
		return m, m.startStatusStream()

	case view.StartDebugLogStreamMsg:
		return m, m.startDebugLogStream(msg.Filter)

	case view.StopDebugLogStreamMsg:
		m.stopDebugLogStream()
		return m, nil

	case view.ClearStatusMsg:
		m.status = nil
		m.err = nil
		for _, v := range m.views {
			if sr, ok := v.(view.StatusReceiver); ok {
				sr.SetStatus(nil)
			}
		}
		return m, nil
	}

	// ── Input mode: command/filter bar owns all keys ──
	if m.mode != modeNormal {
		return m.updateInput(msg)
	}

	// ── Delegate to the active view ──
	// Views get priority so they can override global key bindings
	// (e.g. the debug-log view handles '/' for in-buffer search).
	m, cmd := m.updateActiveView(msg)

	// ── Global keys (only when the view did not consume the key) ──
	if kp, ok := msg.(tea.KeyPressMsg); ok && cmd == nil {
		if m2, gcmd, handled := m.handleGlobalKeys(kp); handled {
			return m2, gcmd
		}
	}

	// ── Post-process: handle orchestration messages emitted by views ──
	// These arrive as tea.Msg from a previous view Update cycle. The view
	// above will ignore them (unhandled in its switch), so we act on them here.
	switch msg := msg.(type) {
	case modelview.NoModelMsg:
		m.stopStatusStream()
		m.err = fmt.Errorf("no model selected — use \"juju add-model <name>\" to create one")

	case view.NavigateMsg:
		return m.handleNavigate(msg)

	case view.GoBackMsg:
		return m.handleBack()

	case view.ScaleRequestMsg:
		return m, m.scaleApplication(msg.AppName, msg.Delta)

	case view.DeployRequestMsg:
		return m, m.deployApplication(msg.ModelName, msg.Options)

	case view.RelateRequestMsg:
		return m, m.relateApplications(msg.EndpointA, msg.EndpointB)

	case view.DestroyRelationRequestMsg:
		return m, m.destroyRelation(msg.EndpointA, msg.EndpointB)

	case debuglog.FilterChangedMsg:
		return m, m.startDebugLogStream(msg.Filter)
	}

	return m, cmd
}

func (m Model) updateActiveView(msg tea.Msg) (Model, tea.Cmd) {
	currentView := m.views[m.stack.Current().View]
	updated, cmd := currentView.Update(msg)
	m.views[m.stack.Current().View] = updated.(view.View)
	return m, cmd
}

func (m Model) contentHeight() int {
	// Layout: header box (logo height + 2 borders) + body box borders (2)
	// + optional input bar (variable height).
	logoHeight := ui.LogoHeight()
	headerHeight := logoHeight + 2 // +2 for top and bottom borders
	chrome := headerHeight + 2     // +2 for body box borders
	chrome += m.inputBarHeight()
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
	jujuVersion := ""
	if m.status != nil {
		jujuVersion = m.status.Model.Version
	}
	headerInner := ui.HeaderContent(controllerName, modelName, cloud, region, m.jaraVersion, jujuVersion, hints, m.width-2)
	sections = append(sections, ui.BorderBox(headerInner, "", m.width))

	// ── Input bar (command/filter mode, between header and body) ──
	if m.mode != modeNormal {
		sections = append(sections, m.renderInputBar())
	}

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

	body := lipgloss.JoinVertical(lipgloss.Left, sections...)
	return tea.NewView(body)
}
