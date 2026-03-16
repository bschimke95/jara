package view

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/ui"
)

const maxLogLines = 1000

// DebugLogMsg delivers a batch of new log entries to the view.
type DebugLogMsg struct {
	Entries []model.LogEntry
	Ctx     context.Context       // stream context for scheduling next read
	Ch      <-chan model.LogEntry // stream channel for scheduling next read
}

// DebugLogErrMsg signals that the debug-log stream encountered an error.
type DebugLogErrMsg struct {
	Err error
}

// DebugLogFilterChangedMsg is emitted when the user applies a new filter from
// inside the debug-log view. The app handles this by restarting the stream.
type DebugLogFilterChangedMsg struct {
	Filter model.DebugLogFilter
}

// debugMode represents the active sub-mode inside the debug-log view.
type debugMode int

const (
	debugModeNormal debugMode = iota
	debugModeFilter           // filter modal is open
	debugModeSearch           // inline search bar is active
)

// DebugLog is the Bubble Tea model for the debug-log streaming view.
type DebugLog struct {
	keys   ui.KeyMap
	width  int
	height int

	lines        []string // formatted log lines (pre-coloured)
	rawEntries   []model.LogEntry
	offset       int  // scroll offset (0 = pinned to bottom)
	paused       bool // when true, auto-scroll is paused

	mode         debugMode
	filterModal  FilterModal
	activeFilter model.DebugLogFilter // currently applied filter (for display)

	status      *model.FullStatus // latest model status, used for suggestions
	seenModules map[string]struct{} // modules observed in the log stream

	searchInput   textinput.Model
	searchQuery   string            // committed search string
	searchMatches []int             // indices into d.lines that match the query
	searchIdx     int               // position in searchMatches
}

// NewDebugLog creates a new debug-log view.
func NewDebugLog() *DebugLog {
	si := textinput.New()
	si.Prompt = "/"
	si.CharLimit = 128

	return &DebugLog{
		keys:        ui.DefaultKeyMap(),
		lines:       make([]string, 0, maxLogLines),
		rawEntries:  make([]model.LogEntry, 0, maxLogLines),
		searchInput: si,
	}
}

// SetFilter pre-populates the active filter (applied before the stream starts).
func (d *DebugLog) SetFilter(f model.DebugLogFilter) {
	d.activeFilter = f
}

// ActiveFilter returns the currently applied filter.
func (d *DebugLog) ActiveFilter() model.DebugLogFilter {
	return d.activeFilter
}

// IsModalOpen reports whether the filter modal is currently displayed.
// The app uses this to suppress global key bindings (e.g. esc/back) so they
// are handled by the modal instead of popping the navigation stack.
func (d *DebugLog) IsModalOpen() bool {
	return d.mode == debugModeFilter
}

// SetSize informs the debug-log view of the current terminal dimensions.
func (d *DebugLog) SetSize(width, height int) {
	d.width = width
	d.height = height
	d.filterModal.SetSize(width, height)
}

// SetStatus stores the latest model status so the filter modal can offer
// unit and machine suggestions.
func (d *DebugLog) SetStatus(s *model.FullStatus) {
	d.status = s
}

// buildSuggestions assembles the per-pane suggestion lists from the current
// model status and the modules seen in the log stream.
func (d *DebugLog) buildSuggestions() map[leftPane][]string {
	sugg := make(map[leftPane][]string)

	if d.status != nil {
		// Applications: all deployed application names.
		for appName := range d.status.Applications {
			sugg[leftPaneApplications] = append(sugg[leftPaneApplications], appName)
		}
		// Units: collect "appname/N" style names from all applications.
		for _, app := range d.status.Applications {
			for _, u := range app.Units {
				sugg[leftPaneUnits] = append(sugg[leftPaneUnits], u.Name)
				for _, sub := range u.Subordinates {
					sugg[leftPaneUnits] = append(sugg[leftPaneUnits], sub.Name)
				}
			}
		}
		// Machines: all top-level machines and their containers.
		for _, mach := range d.status.Machines {
			sugg[leftPaneMachines] = append(sugg[leftPaneMachines], "machine-"+mach.ID)
			for _, c := range mach.Containers {
				sugg[leftPaneMachines] = append(sugg[leftPaneMachines], "machine-"+c.ID)
			}
		}
	}

	// Modules: everything seen in the live stream so far.
	for mod := range d.seenModules {
		sugg[leftPaneModules] = append(sugg[leftPaneModules], mod)
	}

	return sugg
}

// Init satisfies tea.Model; the debug-log view has no startup commands.
func (d *DebugLog) Init() tea.Cmd { return nil }

// Update handles incoming messages: streaming log entries, filter events, search
// key presses, and normal navigation keys.
func (d *DebugLog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// ── Streaming data always handled regardless of mode ──────────────────
	switch msg := msg.(type) {
	case DebugLogMsg:
		for _, entry := range msg.Entries {
			d.rawEntries = append(d.rawEntries, entry)
			d.lines = append(d.lines, formatLogEntry(entry, d.width))
			if entry.Module != "" {
				if d.seenModules == nil {
					d.seenModules = make(map[string]struct{})
				}
				d.seenModules[entry.Module] = struct{}{}
			}
		}
		// Trim to max capacity.
		if len(d.lines) > maxLogLines {
			d.lines = d.lines[len(d.lines)-maxLogLines:]
			d.rawEntries = d.rawEntries[len(d.rawEntries)-maxLogLines:]
		}
		// Re-run search if active.
		if d.searchQuery != "" {
			d.rebuildSearchMatches()
		}
		if !d.paused {
			d.offset = d.bottomOffset()
		}
		return d, nil

	case DebugLogErrMsg:
		errLine := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff5555")).
			Render(fmt.Sprintf("  ⚠ stream error: %v", msg.Err))
		d.lines = append(d.lines, errLine)
		return d, nil
	}

	// ── Filter modal mode ─────────────────────────────────────────────────
	if d.mode == debugModeFilter {
		switch msg := msg.(type) {
		case FilterAppliedMsg:
			d.activeFilter = msg.Filter
			d.mode = debugModeNormal
			// Clear the buffer — the app will restart the stream with the new filter.
			d.lines = make([]string, 0, maxLogLines)
			d.rawEntries = make([]model.LogEntry, 0, maxLogLines)
			d.offset = 0
			d.paused = false
			// Emit DebugLogFilterChangedMsg so app.go restarts the stream.
			return d, func() tea.Msg {
				return DebugLogFilterChangedMsg{Filter: d.activeFilter}
			}
		case FilterModalClosedMsg:
			d.mode = debugModeNormal
			return d, nil
		default:
			updated, cmd := d.filterModal.Update(msg)
			if fm, ok := updated.(*FilterModal); ok {
				d.filterModal = *fm
			}
			return d, cmd
		}
	}

	// ── Search mode ───────────────────────────────────────────────────────
	if d.mode == debugModeSearch {
		kp, isKey := msg.(tea.KeyPressMsg)
		if isKey {
			switch kp.String() {
			case "enter":
				d.searchQuery = d.searchInput.Value()
				d.rebuildSearchMatches()
				d.mode = debugModeNormal
				d.searchInput.Blur()
				if len(d.searchMatches) > 0 {
					d.searchIdx = 0
					d.offset = d.searchMatches[0]
					d.paused = true
				}
				return d, nil
			case "esc":
				d.searchQuery = ""
				d.searchMatches = nil
				d.mode = debugModeNormal
				d.searchInput.Blur()
				return d, nil
			}
		}
		var cmd tea.Cmd
		d.searchInput, cmd = d.searchInput.Update(msg)
		return d, cmd
	}

	// ── Normal key handling ───────────────────────────────────────────────
	if kp, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(kp, d.keys.Up):
			d.scrollUp(1)
			d.paused = true
		case key.Matches(kp, d.keys.Down):
			d.scrollDown(1)
			if d.offset >= d.bottomOffset() {
				d.paused = false
			}
		case key.Matches(kp, d.keys.PageUp):
			d.scrollUp(d.visibleLines() / 2)
			d.paused = true
		case key.Matches(kp, d.keys.PageDown):
			d.scrollDown(d.visibleLines() / 2)
			if d.offset >= d.bottomOffset() {
				d.paused = false
			}
		case key.Matches(kp, d.keys.Top):
			d.offset = 0
			d.paused = true
		case key.Matches(kp, d.keys.Bottom):
			d.offset = d.bottomOffset()
			d.paused = false
		case kp.String() == "F": // Shift+F → open filter modal
			d.filterModal = NewFilterModal(d.activeFilter, d.buildSuggestions())
			d.filterModal.SetSize(d.width, d.height)
			d.mode = debugModeFilter
		case kp.String() == "D": // Shift+D → clear active filter
			d.activeFilter = model.DebugLogFilter{}
			d.lines = make([]string, 0, maxLogLines)
			d.rawEntries = make([]model.LogEntry, 0, maxLogLines)
			d.offset = 0
			d.paused = false
			return d, func() tea.Msg {
				return DebugLogFilterChangedMsg{Filter: d.activeFilter}
			}
		case kp.String() == "/": // search
			d.searchInput.SetValue("")
			d.mode = debugModeSearch
			return d, d.searchInput.Focus()
		case kp.String() == "n": // next search match
			d.jumpToNextMatch(1)
		case kp.String() == "N": // previous search match
			d.jumpToNextMatch(-1)
		}
	}

	return d, nil
}

// View renders the debug-log viewport, compositing the filter modal overlay
// on top when it is open.
func (d *DebugLog) View() tea.View {
	background := d.renderBackground()

	if d.mode == debugModeFilter {
		return tea.NewView(d.filterModal.Render(background))
	}

	return tea.NewView(background)
}

// renderBackground renders the log content (plus search bar if active).
func (d *DebugLog) renderBackground() string {
	if len(d.lines) == 0 {
		waitStyle := lipgloss.NewStyle().Foreground(color.Muted)
		placeholder := waitStyle.Render("  Waiting for log messages...")
		if d.mode == debugModeSearch {
			return placeholder + "\n" + d.renderSearchBar()
		}
		return placeholder
	}

	visH := d.visibleLines()
	start := d.offset
	end := start + visH
	if end > len(d.lines) {
		end = len(d.lines)
	}

	// Highlight lines matching the active search query.
	var b strings.Builder
	for i := start; i < end; i++ {
		line := d.lines[i]
		if d.searchQuery != "" && containsIgnoreCase(d.rawEntries[i%len(d.rawEntries)].Message, d.searchQuery) {
			line = highlightSearchMatch(line, d.searchQuery)
		}
		b.WriteString(line)
		if i < end-1 {
			b.WriteByte('\n')
		}
	}

	// Scroll indicator when paused / not at bottom.
	if d.paused && d.offset < d.bottomOffset() {
		remaining := len(d.lines) - end
		indicator := lipgloss.NewStyle().
			Foreground(color.Primary).
			Bold(true).
			Render(fmt.Sprintf("  ↓ %d more lines (G to resume)", remaining))
		b.WriteByte('\n')
		b.WriteString(indicator)
	}

	// Search bar at the bottom when in search mode.
	if d.mode == debugModeSearch {
		b.WriteByte('\n')
		b.WriteString(d.renderSearchBar())
	} else if d.searchQuery != "" && len(d.searchMatches) > 0 {
		b.WriteByte('\n')
		matchInfo := lipgloss.NewStyle().Foreground(color.Primary).
			Render(fmt.Sprintf("  match %d/%d  (n/N to navigate)", d.searchIdx+1, len(d.searchMatches)))
		b.WriteString(matchInfo)
	}

	return b.String()
}

func (d *DebugLog) renderSearchBar() string {
	keyStyle := lipgloss.NewStyle().Foreground(color.Primary).Bold(true)
	valStyle := lipgloss.NewStyle().Foreground(color.Title)
	return keyStyle.Render("/") + valStyle.Render(d.searchInput.Value()) + keyStyle.Render("█")
}

// FilterTitle returns a pre-styled string summarising the active filter,
// suitable for embedding in the body-box border title. Returns "" when no
// filter is active.
func (d *DebugLog) FilterTitle() string {
	f := d.activeFilter
	var parts []string
	if f.Level != "" {
		parts = append(parts, "level="+f.Level)
	}
	if len(f.Applications) > 0 {
		parts = append(parts, "app="+strings.Join(f.Applications, ","))
	}
	if len(f.IncludeEntities) > 0 {
		parts = append(parts, "entity="+strings.Join(f.IncludeEntities, ","))
	}
	if len(f.ExcludeEntities) > 0 {
		parts = append(parts, "!entity="+strings.Join(f.ExcludeEntities, ","))
	}
	if len(f.IncludeModules) > 0 {
		parts = append(parts, "module="+strings.Join(f.IncludeModules, ","))
	}
	if len(f.ExcludeModules) > 0 {
		parts = append(parts, "!module="+strings.Join(f.ExcludeModules, ","))
	}
	if len(f.IncludeLabels) > 0 {
		var lparts []string
		for k, v := range f.IncludeLabels {
			lparts = append(lparts, k+"="+v)
		}
		sort.Strings(lparts)
		parts = append(parts, "label="+strings.Join(lparts, ","))
	}
	if len(parts) == 0 {
		return ""
	}
	chipStyle := lipgloss.NewStyle().Foreground(color.InfoValue)
	sepStyle := lipgloss.NewStyle().Foreground(color.HintKey).Bold(true)
	var chips []string
	for _, p := range parts {
		chips = append(chips, chipStyle.Render(p))
	}
	return sepStyle.Render(" ▸ ") + strings.Join(chips, sepStyle.Render(" · "))
}

// visibleLines returns how many lines fit in the viewport.
func (d *DebugLog) visibleLines() int {
	h := d.height
	if h < 1 {
		h = 20
	}
	return h
}

// bottomOffset returns the offset that shows the last page of lines.
func (d *DebugLog) bottomOffset() int {
	n := len(d.lines) - d.visibleLines()
	if n < 0 {
		return 0
	}
	return n
}

func (d *DebugLog) scrollUp(n int) {
	d.offset -= n
	if d.offset < 0 {
		d.offset = 0
	}
}

func (d *DebugLog) scrollDown(n int) {
	d.offset += n
	if d.offset > d.bottomOffset() {
		d.offset = d.bottomOffset()
	}
}

// rebuildSearchMatches recomputes which line indices match the current query.
func (d *DebugLog) rebuildSearchMatches() {
	d.searchMatches = d.searchMatches[:0]
	if d.searchQuery == "" {
		return
	}
	q := strings.ToLower(d.searchQuery)
	for i, entry := range d.rawEntries {
		if strings.Contains(strings.ToLower(entry.Message), q) ||
			strings.Contains(strings.ToLower(entry.Entity), q) ||
			strings.Contains(strings.ToLower(entry.Module), q) {
			d.searchMatches = append(d.searchMatches, i)
		}
	}
}

// jumpToNextMatch advances or retreats through search matches.
func (d *DebugLog) jumpToNextMatch(dir int) {
	if len(d.searchMatches) == 0 {
		return
	}
	d.searchIdx = (d.searchIdx + dir + len(d.searchMatches)) % len(d.searchMatches)
	d.offset = d.searchMatches[d.searchIdx]
	d.paused = true
}

// containsIgnoreCase checks if s contains substr case-insensitively.
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// highlightSearchMatch wraps occurrences of query in the line with a highlight style.
func highlightSearchMatch(line, query string) string {
	lower := strings.ToLower(line)
	lq := strings.ToLower(query)
	idx := strings.Index(lower, lq)
	if idx < 0 {
		return line
	}
	hl := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#000000")).
		Background(lipgloss.Color("#ffff00"))
	return line[:idx] + hl.Render(line[idx:idx+len(query)]) + line[idx+len(query):]
}

// severityColor returns a color for the log severity level.
func severityColor(severity string) lipgloss.Style {
	switch strings.ToUpper(severity) {
	case "ERROR", "CRITICAL":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5555"))
	case "WARNING":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#ffff00"))
	case "INFO":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#00bfff"))
	case "DEBUG":
		return lipgloss.NewStyle().Foreground(color.Muted)
	case "TRACE":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#808080"))
	default:
		return lipgloss.NewStyle().Foreground(color.Subtle)
	}
}

// formatLogEntry formats a single log entry to match juju debug-log output:
//
//	entity: timestamp severity module message
func formatLogEntry(entry model.LogEntry, width int) string {
	entityStyle := lipgloss.NewStyle().Foreground(color.Primary)
	tsStyle := lipgloss.NewStyle().Foreground(color.Muted)
	sevStyle := severityColor(entry.Severity)
	moduleStyle := lipgloss.NewStyle().Foreground(color.Secondary)

	ts := entry.Timestamp.Format(time.TimeOnly)

	line := fmt.Sprintf(" %s %s %s %s %s",
		entityStyle.Render(entry.Entity+":"),
		tsStyle.Render(ts),
		sevStyle.Render(entry.Severity),
		moduleStyle.Render(entry.Module),
		entry.Message,
	)

	// Truncate to terminal width to prevent wrapping.
	if width > 0 && lipgloss.Width(line) > width {
		line = lipgloss.NewStyle().MaxWidth(width).Render(line)
	}

	return line
}
