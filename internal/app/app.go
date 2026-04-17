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
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/nav"
	"github.com/bschimke95/jara/internal/ui"
	"github.com/bschimke95/jara/internal/view"
	"github.com/bschimke95/jara/internal/view/appconfig"
	"github.com/bschimke95/jara/internal/view/applications"
	"github.com/bschimke95/jara/internal/view/controllers"
	"github.com/bschimke95/jara/internal/view/debuglog"
	"github.com/bschimke95/jara/internal/view/helpmodal"
	"github.com/bschimke95/jara/internal/view/infomodal"
	"github.com/bschimke95/jara/internal/view/machines"
	"github.com/bschimke95/jara/internal/view/models"
	"github.com/bschimke95/jara/internal/view/modelview"
	"github.com/bschimke95/jara/internal/view/offers"
	"github.com/bschimke95/jara/internal/view/relations"
	"github.com/bschimke95/jara/internal/view/secretdetail"
	"github.com/bschimke95/jara/internal/view/secrets"
	"github.com/bschimke95/jara/internal/view/storage"
	"github.com/bschimke95/jara/internal/view/switchmodal"
	"github.com/bschimke95/jara/internal/view/units"
)

// Model is the root Bubble Tea model.
type Model struct {
	client api.Client
	cfg    *config.Config
	status *model.FullStatus
	styles *color.Styles

	jaraVersion string
	demo        bool

	stack *nav.Stack
	views map[nav.ViewID]view.View

	mode               inputMode
	input              textinput.Model
	filterStr          string
	suggestions        []nav.CommandMatch
	selectedSuggestion int

	keys            ui.KeyMap
	width           int
	height          int
	helpModalOpen   bool
	helpModal       helpmodal.Modal
	infoModalOpen   bool
	infoModal       *infomodal.Modal
	switchModalOpen bool
	switchModal     *switchmodal.Modal

	statusCancel context.CancelFunc      // cancels the status stream
	statusCh     <-chan api.StatusUpdate // receives status snapshots
	logCancel    context.CancelFunc      // cancels the debug-log stream

	charmEndpointsFetched bool // true once charm endpoint info has been polled
	secretsFetched        bool // true once secrets have been explicitly fetched

	toast      *toastState // active error toast (nil = no toast)
	toastSeqNo int         // monotonic counter to match dismiss callbacks
	ready      bool
}

// toastState holds the currently visible error toast.
type toastState struct {
	message string // error message text
	seqNo   int    // sequence number for matching dismiss callbacks
}

// toastExpiredMsg is sent when the toast timer expires.
type toastExpiredMsg struct {
	seqNo int // the sequence number of the toast that should be dismissed
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

// WithDemo enables demo mode.
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

	m.views[nav.AppConfigView] = appconfig.New(keys, s)
	m.views[nav.ApplicationsView] = applications.New(keys, s)
	m.views[nav.UnitsView] = units.New("", keys, s)
	m.views[nav.MachinesView] = machines.New(keys, s)
	m.views[nav.RelationsView] = relations.New(keys, s)
	m.views[nav.SecretsView] = secrets.New(keys, s)
	m.views[nav.SecretDetailView] = secretdetail.New(keys, s)
	m.views[nav.OffersView] = offers.New(keys, s)
	m.views[nav.StorageView] = storage.New(keys, s)
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

	return m
}

// quit closes any open resources and returns tea.Quit.
func (m Model) quit() (Model, tea.Cmd) {
	m.stopStatusStream()
	m.stopDebugLogStream()
	if m.client != nil {
		_ = m.client.Close()
	}
	return m, tea.Quit
}

// showToast creates a new error toast and returns a command that will dismiss
// it after the configured duration. The toast is visible until the timer fires,
// regardless of navigation or status updates.
func (m *Model) showToast(message string) tea.Cmd {
	m.toastSeqNo++
	seq := m.toastSeqNo
	m.toast = &toastState{message: message, seqNo: seq}
	dur := m.cfg.Jara.ToastDuration
	if dur <= 0 {
		dur = config.DefaultToastDuration
	}
	return tea.Tick(dur, func(_ time.Time) tea.Msg {
		return toastExpiredMsg{seqNo: seq}
	})
}

// clearToast removes the active toast if the sequence number matches.
func (m *Model) clearToast(seqNo int) {
	if m.toast != nil && m.toast.seqNo == seqNo {
		m.toast = nil
	}
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
		// Only carry forward when secrets have been explicitly fetched
		// (secretsFetched == true) and the new status has none — this avoids
		// incorrectly preserving stale secrets after a model switch.
		if m.secretsFetched && m.status != nil && len(m.status.Secrets) > 0 && len(msg.status.Secrets) == 0 {
			msg.status.Secrets = m.status.Secrets
		}
		m.status = msg.status
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
		cmd := m.showToast(msg.err.Error())
		return m, tea.Batch(cmd, readNextStatus(msg.ctx, msg.ch))

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
		m.secretsFetched = true
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
		return m, m.showToast(msg.err.Error())

	case toastExpiredMsg:
		m.clearToast(msg.seqNo)
		return m, nil

	case view.StopStatusStreamMsg:
		m.stopStatusStream()
		return m, nil

	case view.StartStatusStreamMsg:
		m.charmEndpointsFetched = false
		m.secretsFetched = false
		return m, m.startStatusStream()

	case view.StartDebugLogStreamMsg:
		return m, m.startDebugLogStream(msg.Filter)

	case view.StopDebugLogStreamMsg:
		m.stopDebugLogStream()
		return m, nil

	case view.ClearStatusMsg:
		m.status = nil
		for _, v := range m.views {
			if sr, ok := v.(view.StatusReceiver); ok {
				sr.SetStatus(nil)
			}
			if cr, ok := v.(view.CharmEndpointReceiver); ok {
				cr.SetCharmEndpoints(nil)
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

	// ── Info modal: owns all keys when open ──
	if m.infoModalOpen {
		if _, ok := msg.(infomodal.ClosedMsg); ok {
			m.infoModalOpen = false
			m.infoModal = nil
			return m, nil
		}
		if _, ok := msg.(tea.KeyPressMsg); ok {
			updated, cmd := m.infoModal.Update(msg)
			if im, ok := updated.(*infomodal.Modal); ok {
				m.infoModal = im
			}
			return m, cmd
		}
		return m, nil
	}

	// ── Switch modal: owns all keys when open ──
	if m.switchModalOpen {
		switch msg := msg.(type) {
		case switchmodal.SelectedMsg:
			m.switchModalOpen = false
			m.switchModal = nil
			return m.switchEntityContext(msg.Entity)
		case switchmodal.PreviewMsg:
			return m.switchEntityContext(msg.Entity)
		case switchmodal.ClosedMsg:
			m.switchModalOpen = false
			m.switchModal = nil
			// Restore the original entity if the user cancelled.
			return m.switchEntityContext(msg.Original)
		default:
			if _, ok := msg.(tea.KeyPressMsg); ok {
				updated, cmd := m.switchModal.Update(msg)
				if sm, ok := updated.(*switchmodal.Modal); ok {
					m.switchModal = sm
				}
				return m, cmd
			}
			return m, nil
		}
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
		return m, m.showToast("no model selected — use \"juju add-model <name>\" to create one")
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

	case offers.FetchOffersMsg:
		return m, m.fetchOffers()

	case appconfig.FetchAppConfigMsg:
		return m, m.fetchAppConfig(msg.AppName)

	case view.FetchActionsRequestMsg:
		return m, m.fetchActions(msg.AppName)

	case view.RunActionRequestMsg:
		return m, m.runAction(msg.UnitName, msg.ActionName, msg.Params)

	case storage.FetchStorageMsg:
		return m, m.fetchStorage()

	case debuglog.FilterChangedMsg:
		return m, m.startDebugLogStream(msg.Filter)
	}

	return m, cmd
}

func (m Model) updateActiveView(msg tea.Msg) (Model, tea.Cmd) {
	currentView := m.views[m.stack.Current().View]
	updated, cmd := currentView.Update(msg)
	if v, ok := updated.(view.View); ok {
		m.views[m.stack.Current().View] = v
	}
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
	case nav.OffersView:
		return "Offers"
	case nav.AppConfigView:
		return "Config"
	case nav.StorageView:
		return "Storage"
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

	currentView := m.views[m.stack.Current().View]

	// ── Header box: status info (left) + key hints (center) + logo (right) ──
	if !m.cfg.Jara.Headless {
		controllerName := m.client.ControllerName()
		modelName, cloud, region, timestamp := "", "", "", ""
		if m.status != nil {
			modelName = m.status.Model.Name
			cloud = m.status.Model.Cloud
			region = m.status.Model.Region
			if m.status.ControllerTimestamp != nil {
				timestamp = shortAge(time.Since(*m.status.ControllerTimestamp))
			}
		}
		hints := m.buildHeaderHints(currentView.KeyHints())
		jujuVersion := ""
		if m.status != nil {
			jujuVersion = m.status.Model.Version
		}
		headerInner := ui.HeaderContent(controllerName, modelName, cloud, region, m.jaraVersion, jujuVersion, timestamp, hints, m.width-2, m.styles)
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

	// ── Breadcrumb bar (with optional inline toast) ──
	crumbLine := ui.CrumbBar(m.stack.Breadcrumbs(), m.width, m.styles)
	if m.toast != nil {
		toastText := m.styles.ToastStyle.Render(" ⚠ " + m.toast.message + " ")
		toastWidth := lipgloss.Width(toastText)
		crumbWidth := lipgloss.Width(crumbLine)
		gap := m.width - crumbWidth - toastWidth
		if gap < 1 {
			gap = 1
		}
		crumbLine = crumbLine + strings.Repeat(" ", gap) + toastText + "\n"
	}
	sections = append(sections, crumbLine)

	body := lipgloss.JoinVertical(lipgloss.Left, sections...)

	// ── Info modal overlay ──
	if m.infoModalOpen && m.infoModal != nil {
		return tea.NewView(m.infoModal.Render(body))
	}

	// ── Switch modal overlay ──
	if m.switchModalOpen && m.switchModal != nil {
		return tea.NewView(m.switchModal.Render(body))
	}

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

// shortAge formats a duration as a compact age string (e.g. "3m22s", "2h5m", "4d12h").
func shortAge(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		m := int(d.Minutes())
		s := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%ds", m, s)
	case d < 24*time.Hour:
		h := int(d.Hours())
		m := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh%dm", h, m)
	default:
		days := int(d.Hours()) / 24
		h := int(d.Hours()) % 24
		return fmt.Sprintf("%dd%dh", days, h)
	}
}
