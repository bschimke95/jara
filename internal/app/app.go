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
	"github.com/bschimke95/jara/internal/config"
	"github.com/bschimke95/jara/internal/llm"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/nav"
	"github.com/bschimke95/jara/internal/ui"
	"github.com/bschimke95/jara/internal/view"
	"github.com/bschimke95/jara/internal/view/applications"
	"github.com/bschimke95/jara/internal/view/chat"
	"github.com/bschimke95/jara/internal/view/controllers"
	"github.com/bschimke95/jara/internal/view/debuglog"
	"github.com/bschimke95/jara/internal/view/helpmodal"
	"github.com/bschimke95/jara/internal/view/machines"
	"github.com/bschimke95/jara/internal/view/models"
	"github.com/bschimke95/jara/internal/view/modelview"
	"github.com/bschimke95/jara/internal/view/relations"
	"github.com/bschimke95/jara/internal/view/secretdetail"
	"github.com/bschimke95/jara/internal/view/secrets"
	"github.com/bschimke95/jara/internal/view/units"
)

// Model is the root Bubble Tea model.
type Model struct {
	client    api.Client
	cfg       *config.Config
	status    *model.FullStatus
	styles    *color.Styles
	llmClient llm.Client

	jaraVersion string
	demo        bool

	stack *nav.Stack
	views map[nav.ViewID]view.View

	mode               inputMode
	input              textinput.Model
	filterStr          string
	suggestions        []nav.CommandMatch
	selectedSuggestion int

	keys          ui.KeyMap
	width         int
	height        int
	helpModalOpen bool
	helpModal     helpmodal.Modal

	statusCancel context.CancelFunc      // cancels the status stream
	statusCh     <-chan api.StatusUpdate // receives status snapshots
	logCancel    context.CancelFunc      // cancels the debug-log stream

	charmEndpointsFetched bool // true once charm endpoint info has been polled

	err   error
	ready bool
}

// Option configures the root Model.
type Option func(*Model)

// WithStyles sets the resolved styles on the model and applies the legacy
// global color variables for backward compatibility during migration.
func WithStyles(s *color.Styles) Option {
	return func(m *Model) {
		if s != nil {
			m.styles = s
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

// WithDemo enables demo mode (uses mock LLM client).
func WithDemo(d bool) Option {
	return func(m *Model) {
		m.demo = d
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
		styles: color.DefaultStyles(),
		stack:  nav.NewStack(nav.ModelView),
		views:  make(map[nav.ViewID]view.View),
		keys:   keys,
		input:  ti,
		mode:   modeNormal,
	}

	for _, opt := range opts {
		opt(&m)
	}

	s := m.styles

	// Initialize helpModal with the final resolved keys and styles.
	m.helpModal = helpmodal.New(m.keys, s)

	m.views[nav.ApplicationsView] = applications.New(keys, s)
	m.views[nav.UnitsView] = units.New("", keys, s)
	m.views[nav.MachinesView] = machines.New(keys, s)
	m.views[nav.RelationsView] = relations.New(keys, s)
	m.views[nav.SecretsView] = secrets.New(keys, s)
	m.views[nav.SecretDetailView] = secretdetail.New(keys, s)
	m.views[nav.DebugLogView] = debuglog.New(keys, s)
	m.views[nav.ControllerView] = controllers.New(keys, s, func() tea.Cmd { return m.pollControllers() })
	m.views[nav.ModelsView] = models.New(keys, s,
		func(ctrl string) tea.Cmd { return m.pollModels(ctrl) },
		func(name string) error { return m.client.SelectController(name) },
		func() string { return m.client.ControllerName() },
	)
	m.views[nav.ModelView] = modelview.New(keys, s,
		func(name string) error { return m.client.SelectModel(name) },
	)

	// Initialize LLM client for the AI chat view.
	var llmClient llm.Client
	var llmInitErr string
	if m.demo {
		llmClient = llm.NewMockClient(20 * time.Millisecond)
	} else {
		var err error
		llmClient, err = initLLMClient(m.cfg)
		if err != nil {
			llmInitErr = err.Error()
		}
	}
	m.llmClient = llmClient
	systemPrompt := m.cfg.Jara.AI.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = llm.DefaultSystemPrompt
	}
	m.views[nav.ChatView] = chat.New(keys, s, llmClient, systemPrompt, llmInitErr)

	return m
}

// quit closes any open resources and returns tea.Quit.
func (m Model) quit() (Model, tea.Cmd) {
	if m.llmClient != nil {
		_ = m.llmClient.Close()
	}
	return m, tea.Quit
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
		m.helpModal.SetSize(m.width, m.height)
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
		// Carry forward secrets from the previous status; they are fetched
		// via a separate API call and not included in the status stream.
		if m.status != nil && len(m.status.Secrets) > 0 && len(msg.status.Secrets) == 0 {
			msg.status.Secrets = m.status.Secrets
		}
		m.status = msg.status
		m.err = nil
		for _, v := range m.views {
			if sr, ok := v.(view.StatusReceiver); ok {
				sr.SetStatus(msg.status)
			}
		}
		var cmds []tea.Cmd
		cmds = append(cmds, readNextStatus(msg.ctx, msg.ch))
		if !m.charmEndpointsFetched {
			m.charmEndpointsFetched = true
			if cmd := m.pollCharmEndpoints(); cmd != nil {
				cmds = append(cmds, cmd)
			}
			cmds = append(cmds, m.pollSecrets())
		}
		return m, tea.Batch(cmds...)

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

	case charmEndpointsMsg:
		for _, v := range m.views {
			if sr, ok := v.(view.CharmEndpointReceiver); ok {
				sr.SetCharmEndpoints(msg.Endpoints)
			}
		}
		return m, nil

	case secretsMsg:
		if m.status != nil {
			m.status.Secrets = msg.Secrets
			for _, v := range m.views {
				if sr, ok := v.(view.StatusReceiver); ok {
					sr.SetStatus(m.status)
				}
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
		m.charmEndpointsFetched = false
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

	// ── Help modal: owns all keys when open ──
	if m.helpModalOpen {
		if _, ok := msg.(helpmodal.ClosedMsg); ok {
			m.helpModalOpen = false
			return m, nil
		}
		if _, ok := msg.(tea.KeyPressMsg); ok {
			updated, cmd := m.helpModal.Update(msg)
			if hm, ok := updated.(*helpmodal.Modal); ok {
				m.helpModal = *hm
			}
			return m, cmd
		}
		return m, nil
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

	case view.RevealSecretRequestMsg:
		return m, m.revealSecret(msg.URI, msg.Revision)

	case relations.FetchRelationDataMsg:
		return m, m.fetchRelationData(msg.RelationID)

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
	currentViewID := m.stack.Current().View
	// Layout: body box borders (2) + breadcrumb bar + bottom padding (2).
	chrome := 4
	if !m.cfg.Jara.Headless {
		// Header box: logo height + 2 borders.
		chrome += ui.LogoHeight() + 2
	}
	// Views that render their own borders don't need the outer body border.
	if currentViewID == nav.ModelView || currentViewID == nav.RelationsView {
		chrome -= 2
	}
	chrome += m.inputBarHeight()
	// When the debug-log search bar is visible it occupies 3 rows (bordered box).
	if currentViewID == nav.DebugLogView {
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
	case nav.SecretsView:
		return "Secrets"
	case nav.SecretDetailView:
		return "Secret"
	case nav.DebugLogView:
		return "Debug Log"
	case nav.ChatView:
		return "Chat"
	default:
		return "jara"
	}
}

func (m Model) View() tea.View {
	if !m.ready {
		return tea.NewView("Initializing jara...")
	}

	var sections []string

	currentView := m.views[m.stack.Current().View]

	// ── Header box: status info (left) + key hints (center) + logo (right) ──
	if !m.cfg.Jara.Headless {
		controllerName := m.client.ControllerName()
		modelName, cloud, region := "", "", ""
		if m.status != nil {
			modelName = m.status.Model.Name
			cloud = m.status.Model.Cloud
			region = m.status.Model.Region
		}
		hints := m.buildHeaderHints(currentView.KeyHints())
		jujuVersion := ""
		if m.status != nil {
			jujuVersion = m.status.Model.Version
		}
		headerInner := ui.HeaderContent(controllerName, modelName, cloud, region, m.jaraVersion, jujuVersion, hints, m.width-2, m.styles)
		sections = append(sections, ui.BorderBox(headerInner, "", m.width, m.styles))
	}

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

	// The ModelView and RelationsView render their own bordered panes, so skip the outer border.
	if m.stack.Current().View == nav.ModelView || m.stack.Current().View == nav.RelationsView {
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
				titleStyle := lipgloss.NewStyle().Foreground(m.styles.BorderTitleColor).Bold(true)
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
			sections = append(sections, ui.BorderBoxRawTitle(bodyContent, rawTitle, m.width, m.styles))
		} else {
			sections = append(sections, ui.BorderBox(bodyContent, bodyTitle, m.width, m.styles))
		}
	}

	// ── Breadcrumb bar ──
	sections = append(sections, ui.CrumbBar(m.stack.Breadcrumbs(), m.width, m.styles))

	// ── Error line ──
	if m.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(m.styles.ErrorColor)
		sections = append(sections, errStyle.Render(" Error: "+m.err.Error()))
	}

	body := lipgloss.JoinVertical(lipgloss.Left, sections...)

	// ── Help modal overlay ──
	if m.helpModalOpen {
		return tea.NewView(m.helpModal.Render(body))
	}

	return tea.NewView(body)
}

// buildHeaderHints composes the header hint list: view-specific hints first,
// then general fill hints if there is space. Help is always included.
// The total is capped at 2×ui.MaxHintsPerColumn (both columns).
func (m Model) buildHeaderHints(viewHints []ui.KeyHint) []ui.KeyHint {
	bk := func(b key.Binding) string { return b.Help().Key }
	helpHint := ui.KeyHint{Key: bk(m.keys.Help), Desc: "help"}

	// General fill hints (in priority order, excluding help which is always appended).
	generalFill := []ui.KeyHint{
		{Key: bk(m.keys.Command), Desc: "cmd"},
		{Key: bk(m.keys.Quit), Desc: "quit"},
	}

	// Reserve one slot for the help hint; fill the rest across both columns.
	limit := ui.MaxHintsPerColumn*2 - 1

	var hints []ui.KeyHint
	for _, h := range viewHints {
		if len(hints) >= limit {
			break
		}
		hints = append(hints, h)
	}
	for _, h := range generalFill {
		if len(hints) >= limit {
			break
		}
		hints = append(hints, h)
	}

	// Always append help as the last hint.
	hints = append(hints, helpHint)
	return hints
}

// initLLMClient creates an LLM client based on the current configuration.
// Returns (nil, nil) when no credentials are available (graceful degradation).
// Returns (nil, err) when credentials exist but the client cannot be created
// (e.g. the Copilot CLI binary is missing).
func initLLMClient(cfg *config.Config) (llm.Client, error) {
	provider := strings.ToLower(strings.TrimSpace(cfg.Jara.AI.Provider))
	if provider == "" {
		provider = "copilot"
	}

	cred := config.LoadAICredential(provider)
	aiCfg := cfg.Jara.AI

	switch provider {
	case "copilot":
		opts := []llm.CopilotOption{
			llm.WithCopilotModel(aiCfg.Model),
		}
		if cred != "" {
			opts = append(opts, llm.WithCopilotGitHubToken(cred))
		}
		c, err := llm.NewCopilotClient(opts...)
		if err != nil {
			// Only surface the error when credentials were found — if there are
			// no credentials this is the expected "not configured" path.
			if cred != "" {
				return nil, err
			}
			return nil, nil
		}
		return c, nil

	case "gemini":
		if cred == "" {
			return nil, nil
		}
		temp := 0.7
		if aiCfg.Temperature != nil {
			temp = *aiCfg.Temperature
		}
		maxTokens := 4096
		if aiCfg.MaxTokens != nil {
			maxTokens = *aiCfg.MaxTokens
		}
		c, err := llm.NewGeminiClient(context.Background(), cred,
			llm.WithGeminiModel(aiCfg.Model),
			llm.WithGeminiTemperature(temp),
			llm.WithGeminiMaxTokens(maxTokens),
		)
		if err != nil {
			return nil, err
		}
		return c, nil
	}

	return nil, nil
}
