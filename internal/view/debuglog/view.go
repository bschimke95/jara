// Package debuglog implements the self-contained debug-log streaming view
// with inline search, scroll navigation, and a two-pane filter modal overlay.
package debuglog

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/ui"
	"github.com/bschimke95/jara/internal/view"
)

// New creates a new debug-log view.
func New(keys ui.KeyMap, styles *color.Styles) *View {
	si := textinput.New()
	si.Prompt = "/"
	si.CharLimit = 128

	return &View{
		keys:        keys,
		styles:      styles,
		lines:       make([]string, 0, maxLogLines),
		rawEntries:  make([]model.LogEntry, 0, maxLogLines),
		searchInput: si,
	}
}

// SetFilter pre-populates the active filter (applied before the stream starts).
func (d *View) SetFilter(f model.DebugLogFilter) {
	d.activeFilter = f
}

// ActiveFilter returns the currently applied filter.
func (d *View) ActiveFilter() model.DebugLogFilter {
	return d.activeFilter
}

// IsModalOpen reports whether the filter modal is currently displayed.
func (d *View) IsModalOpen() bool {
	return d.mode == debugModeFilter
}

// IsSearchActive reports whether the inline search bar is currently open.
func (d *View) IsSearchActive() bool {
	return d.mode == debugModeSearch
}

func (d *View) SetSize(width, height int) {
	d.width = width
	d.height = height
	d.filterModal.SetSize(width, height)
}

// SetStatus implements view.StatusReceiver.
func (d *View) SetStatus(s *model.FullStatus) {
	d.status = s
}

// KeyHints returns the view-specific key hints for the header.
func (d *View) KeyHints() []view.KeyHint {
	return []view.KeyHint{
		{Key: view.BindingKey(d.keys.Back), Desc: "back"},
		{Key: view.BindingKey(d.keys.Bottom), Desc: "bottom"},
		{Key: view.BindingKey(d.keys.Top), Desc: "top"},
		{Key: view.BindingKey(d.keys.FilterOpen), Desc: "filter"},
		{Key: view.BindingKey(d.keys.ClearFilter), Desc: "clear filter"},
		{Key: view.BindingKey(d.keys.SearchOpen), Desc: "search"},
		{Key: view.BindingKey(d.keys.SearchNext) + "/" + view.BindingKey(d.keys.SearchPrev), Desc: "next/prev match"},
	}
}

// buildSuggestions assembles the per-pane suggestion lists from the current
// model status and the modules seen in the log stream.
func (d *View) buildSuggestions() map[leftPane][]string {
	sugg := make(map[leftPane][]string)

	if d.status != nil {
		for appName := range d.status.Applications {
			sugg[leftPaneApplications] = append(sugg[leftPaneApplications], appName)
		}
		for _, app := range d.status.Applications {
			for _, u := range app.Units {
				sugg[leftPaneUnits] = append(sugg[leftPaneUnits], u.Name)
				for _, sub := range u.Subordinates {
					sugg[leftPaneUnits] = append(sugg[leftPaneUnits], sub.Name)
				}
			}
		}
		for _, mach := range d.status.Machines {
			sugg[leftPaneMachines] = append(sugg[leftPaneMachines], "machine-"+mach.ID)
			for _, c := range mach.Containers {
				sugg[leftPaneMachines] = append(sugg[leftPaneMachines], "machine-"+c.ID)
			}
		}
	}

	for mod := range d.seenModules {
		sugg[leftPaneModules] = append(sugg[leftPaneModules], mod)
	}

	return sugg
}

// ReadNextLogBatch reads the next batch of entries from the stream channel.
// It blocks on the first entry, then drains any immediately available extras.
func ReadNextLogBatch(ctx context.Context, ch <-chan model.LogEntry) tea.Msg {
	select {
	case <-ctx.Done():
		return nil
	case entry, ok := <-ch:
		if !ok {
			return ErrMsg{Err: fmt.Errorf("log stream closed")}
		}
		batch := []model.LogEntry{entry}
	drainLoop:
		for {
			select {
			case e, ok := <-ch:
				if !ok {
					break drainLoop
				}
				batch = append(batch, e)
				if len(batch) >= 50 {
					break drainLoop
				}
			default:
				break drainLoop
			}
		}
		return Msg{Entries: batch, Ctx: ctx, Ch: ch}
	}
}

func (d *View) Init() tea.Cmd { return nil }

func (d *View) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case Msg:
		for _, entry := range msg.Entries {
			d.rawEntries = append(d.rawEntries, entry)
			d.lines = append(d.lines, formatLogEntry(entry, d.width, d.styles))
			if entry.Module != "" {
				if d.seenModules == nil {
					d.seenModules = make(map[string]struct{})
				}
				d.seenModules[entry.Module] = struct{}{}
			}
		}
		if len(d.lines) > maxLogLines {
			d.lines = d.lines[len(d.lines)-maxLogLines:]
			d.rawEntries = d.rawEntries[len(d.rawEntries)-maxLogLines:]
		}
		if d.searchQuery != "" {
			d.rebuildSearchMatches()
		}
		if !d.paused {
			d.offset = d.bottomOffset()
		}
		// Schedule the next read from the stream.
		var nextRead tea.Cmd
		if msg.Ctx != nil && msg.Ch != nil {
			ctx, ch := msg.Ctx, msg.Ch
			nextRead = func() tea.Msg { return ReadNextLogBatch(ctx, ch) }
		}
		return d, nextRead

	case ErrMsg:
		errLine := lipgloss.NewStyle().
			Foreground(d.styles.ErrorColor).
			Render(fmt.Sprintf("  ⚠ stream error: %v", msg.Err))
		d.lines = append(d.lines, errLine)
		return d, nil
	}

	if d.mode == debugModeFilter {
		switch msg := msg.(type) {
		case FilterAppliedMsg:
			d.activeFilter = msg.Filter
			d.mode = debugModeNormal
			d.lines = make([]string, 0, maxLogLines)
			d.rawEntries = make([]model.LogEntry, 0, maxLogLines)
			d.offset = 0
			d.paused = false
			return d, func() tea.Msg {
				return FilterChangedMsg{Filter: d.activeFilter}
			}
		case FilterModalClosedMsg:
			d.mode = debugModeNormal
			return d, nil
		default:
			updated, cmd := d.filterModal.Update(msg)
			if fm, ok := updated.(*FilterModal); ok {
				d.filterModal = *fm
			}
			// If the modal consumed a key but returned nil cmd, return a
			// non-nil no-op so the global Back handler does not fire.
			if cmd == nil {
				if _, isKey := msg.(tea.KeyPressMsg); isKey {
					cmd = func() tea.Msg { return nil }
				}
			}
			return d, cmd
		}
	}

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
				// Return a non-nil no-op cmd to signal the key was consumed
				// here, preventing the global Back handler from also firing.
				return d, func() tea.Msg { return nil }
			}
		}
		var cmd tea.Cmd
		d.searchInput, cmd = d.searchInput.Update(msg)
		return d, cmd
	}

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
		case key.Matches(kp, d.keys.FilterOpen):
			d.filterModal = NewFilterModal(d.activeFilter, d.buildSuggestions(), d.keys, d.styles)
			d.filterModal.SetSize(d.width, d.height)
			d.mode = debugModeFilter
		case key.Matches(kp, d.keys.ClearFilter):
			d.activeFilter = model.DebugLogFilter{}
			d.lines = make([]string, 0, maxLogLines)
			d.rawEntries = make([]model.LogEntry, 0, maxLogLines)
			d.offset = 0
			d.paused = false
			return d, func() tea.Msg {
				return FilterChangedMsg{Filter: d.activeFilter}
			}
		case key.Matches(kp, d.keys.SearchOpen):
			d.searchInput.SetValue("")
			d.mode = debugModeSearch
			return d, d.searchInput.Focus()
		case key.Matches(kp, d.keys.SearchNext):
			d.jumpToNextMatch(1)
		case key.Matches(kp, d.keys.SearchPrev):
			d.jumpToNextMatch(-1)
		}
	}

	return d, nil
}

func (d *View) View() tea.View {
	background := d.renderBackground()

	if d.mode == debugModeFilter {
		return tea.NewView(d.filterModal.Render(background))
	}

	return tea.NewView(background)
}

func (d *View) renderBackground() string {
	if len(d.lines) == 0 {
		waitStyle := lipgloss.NewStyle().Foreground(d.styles.Muted)
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

	var b strings.Builder
	for i := start; i < end; i++ {
		line := d.lines[i]
		if d.searchQuery != "" {
			entry := d.rawEntries[i]
			if containsIgnoreCase(entry.Message, d.searchQuery) ||
				containsIgnoreCase(entry.Entity, d.searchQuery) ||
				containsIgnoreCase(entry.Module, d.searchQuery) {
				line = formatLogEntryHighlighted(entry, d.width, d.searchQuery, d.styles)
			}
		}
		b.WriteString(line)
		if i < end-1 {
			b.WriteByte('\n')
		}
	}

	if d.paused && d.offset < d.bottomOffset() {
		remaining := len(d.lines) - end
		indicator := lipgloss.NewStyle().
			Foreground(d.styles.Primary).
			Bold(true).
			Render(fmt.Sprintf("  ↓ %d more lines (G to resume)", remaining))
		b.WriteByte('\n')
		b.WriteString(indicator)
	}

	if d.mode == debugModeSearch {
		b.WriteByte('\n')
		b.WriteString(d.renderSearchBar())
	} else if d.searchQuery != "" && len(d.searchMatches) > 0 {
		b.WriteByte('\n')
		matchInfo := lipgloss.NewStyle().Foreground(d.styles.Primary).
			Render(fmt.Sprintf("  match %d/%d  (n/N to navigate)", d.searchIdx+1, len(d.searchMatches)))
		b.WriteString(matchInfo)
	}

	return b.String()
}

// RenderSearchBar returns a full-width bordered search bar row.
func (d *View) RenderSearchBar(width int) string {
	innerW := width - 2
	if innerW < 2 {
		innerW = 2
	}
	keyStyle := lipgloss.NewStyle().Foreground(d.styles.Primary).Bold(true)
	valStyle := lipgloss.NewStyle().Foreground(d.styles.Title)
	cursorStyle := lipgloss.NewStyle().Foreground(d.styles.Primary)
	inputLine := keyStyle.Render("/") + valStyle.Render(d.searchInput.Value()) + cursorStyle.Render("█")
	pad := innerW - lipgloss.Width(inputLine)
	if pad < 0 {
		pad = 0
	}
	content := inputLine + strings.Repeat(" ", pad)
	titleStyle := lipgloss.NewStyle().Foreground(d.styles.Primary).Bold(true)
	return ui.BorderBoxRawTitle(content, titleStyle.Render(" Search "), width, d.styles)
}

func (d *View) renderSearchBar() string {
	keyStyle := lipgloss.NewStyle().Foreground(d.styles.Primary).Bold(true)
	valStyle := lipgloss.NewStyle().Foreground(d.styles.Title)
	return keyStyle.Render("/") + valStyle.Render(d.searchInput.Value()) + keyStyle.Render("█")
}

// FilterTitle returns a pre-styled string summarising the active filter,
// suitable for embedding in the body-box border title.
func (d *View) FilterTitle() string {
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
	chipStyle := lipgloss.NewStyle().Foreground(d.styles.InfoValueColor)
	sepStyle := lipgloss.NewStyle().Foreground(d.styles.HintKeyColor).Bold(true)
	var chips []string
	for _, p := range parts {
		chips = append(chips, chipStyle.Render(p))
	}
	return sepStyle.Render(" ▸ ") + strings.Join(chips, sepStyle.Render(" · "))
}

func (d *View) visibleLines() int {
	h := d.height
	if h < 1 {
		h = 20
	}
	return h
}

func (d *View) bottomOffset() int {
	n := len(d.lines) - d.visibleLines()
	if n < 0 {
		return 0
	}
	return n
}

func (d *View) scrollUp(n int) {
	d.offset -= n
	if d.offset < 0 {
		d.offset = 0
	}
}

func (d *View) scrollDown(n int) {
	d.offset += n
	if d.offset > d.bottomOffset() {
		d.offset = d.bottomOffset()
	}
}

func (d *View) rebuildSearchMatches() {
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

func (d *View) jumpToNextMatch(dir int) {
	if len(d.searchMatches) == 0 {
		return
	}
	d.searchIdx = (d.searchIdx + dir + len(d.searchMatches)) % len(d.searchMatches)
	d.offset = d.searchMatches[d.searchIdx]
	d.paused = true
}

func (d *View) Enter(ctx view.NavigateContext) (tea.Cmd, error) {
	if ctx.Filter != nil {
		si := textinput.New()
		si.Prompt = "/"
		si.CharLimit = 128
		d.lines = make([]string, 0, maxLogLines)
		d.rawEntries = make([]model.LogEntry, 0, maxLogLines)
		d.searchInput = si
		d.mode = debugModeNormal
		d.searchQuery = ""
		d.searchMatches = nil
		d.offset = 0
		d.paused = false
		d.seenModules = nil
		d.activeFilter = *ctx.Filter
	}
	return func() tea.Msg {
		return view.StartDebugLogStreamMsg{Filter: d.activeFilter}
	}, nil
}

func (d *View) Leave() tea.Cmd {
	return func() tea.Msg { return view.StopDebugLogStreamMsg{} }
}
