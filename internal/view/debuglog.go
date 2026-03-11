package view

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
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

// DebugLog is the Bubble Tea model for the debug-log streaming view.
type DebugLog struct {
	keys   ui.KeyMap
	width  int
	height int

	lines  []string // formatted log lines
	offset int      // scroll offset (0 = pinned to bottom)
	paused bool     // when true, auto-scroll is paused
}

// NewDebugLog creates a new debug-log view.
func NewDebugLog() *DebugLog {
	return &DebugLog{
		keys:  ui.DefaultKeyMap(),
		lines: make([]string, 0, maxLogLines),
	}
}

func (d *DebugLog) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// SetStatus is a no-op — the debug-log view uses its own streaming data.
func (d *DebugLog) SetStatus(_ *model.FullStatus) {}

func (d *DebugLog) Init() tea.Cmd { return nil }

func (d *DebugLog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case DebugLogMsg:
		for _, entry := range msg.Entries {
			d.lines = append(d.lines, formatLogEntry(entry, d.width))
		}
		// Trim to max capacity.
		if len(d.lines) > maxLogLines {
			d.lines = d.lines[len(d.lines)-maxLogLines:]
		}
		// If not paused, stay pinned to the bottom.
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

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, d.keys.Up):
			d.scrollUp(1)
			d.paused = true
		case key.Matches(msg, d.keys.Down):
			d.scrollDown(1)
			if d.offset >= d.bottomOffset() {
				d.paused = false
			}
		case key.Matches(msg, d.keys.PageUp):
			d.scrollUp(d.visibleLines() / 2)
			d.paused = true
		case key.Matches(msg, d.keys.PageDown):
			d.scrollDown(d.visibleLines() / 2)
			if d.offset >= d.bottomOffset() {
				d.paused = false
			}
		case key.Matches(msg, d.keys.Top):
			d.offset = 0
			d.paused = true
		case key.Matches(msg, d.keys.Bottom):
			d.offset = d.bottomOffset()
			d.paused = false
		}
		return d, nil
	}

	return d, nil
}

func (d *DebugLog) View() tea.View {
	if len(d.lines) == 0 {
		waitStyle := lipgloss.NewStyle().Foreground(color.Muted)
		return tea.NewView(waitStyle.Render("  Waiting for log messages..."))
	}

	visible := d.visibleLines()
	start := d.offset
	end := start + visible
	if end > len(d.lines) {
		end = len(d.lines)
	}

	var b strings.Builder
	for i := start; i < end; i++ {
		b.WriteString(d.lines[i])
		if i < end-1 {
			b.WriteByte('\n')
		}
	}

	// Show scroll indicator when paused / not at bottom.
	if d.paused && d.offset < d.bottomOffset() {
		remaining := len(d.lines) - end
		indicator := lipgloss.NewStyle().
			Foreground(color.Primary).
			Bold(true).
			Render(fmt.Sprintf("  ↓ %d more lines (G to resume)", remaining))
		b.WriteByte('\n')
		b.WriteString(indicator)
	}

	return tea.NewView(b.String())
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
