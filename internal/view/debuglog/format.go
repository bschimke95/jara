package debuglog

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
)

func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

func highlightSearchMatch(line, query string) string {
	lower := strings.ToLower(line)
	lq := strings.ToLower(query)
	idx := strings.Index(lower, lq)
	if idx < 0 {
		return line
	}
	hl := lipgloss.NewStyle().
		Foreground(color.SearchHighlightFg).
		Background(color.SearchHighlightBg)
	return line[:idx] + hl.Render(line[idx:idx+len(query)]) + line[idx+len(query):]
}

func severityColor(severity string) lipgloss.Style {
	switch strings.ToUpper(severity) {
	case "ERROR", "CRITICAL":
		return lipgloss.NewStyle().Foreground(color.CheckRed)
	case "WARNING":
		return lipgloss.NewStyle().Foreground(color.StatusColor("waiting"))
	case "INFO":
		return lipgloss.NewStyle().Foreground(color.Primary)
	case "DEBUG":
		return lipgloss.NewStyle().Foreground(color.Muted)
	case "TRACE":
		return lipgloss.NewStyle().Foreground(color.Subtle)
	default:
		return lipgloss.NewStyle().Foreground(color.Subtle)
	}
}

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

	if width > 0 && lipgloss.Width(line) > width {
		line = lipgloss.NewStyle().MaxWidth(width).Render(line)
	}

	return line
}

func formatLogEntryHighlighted(entry model.LogEntry, width int, query string) string {
	entityStyle := lipgloss.NewStyle().Foreground(color.Primary)
	tsStyle := lipgloss.NewStyle().Foreground(color.Muted)
	sevStyle := severityColor(entry.Severity)
	moduleStyle := lipgloss.NewStyle().Foreground(color.Secondary)

	ts := entry.Timestamp.Format(time.TimeOnly)

	highlightField := func(s string) string {
		return highlightSearchMatch(s, query)
	}

	line := fmt.Sprintf(" %s %s %s %s %s",
		entityStyle.Render(highlightField(entry.Entity+":")),
		tsStyle.Render(ts),
		sevStyle.Render(entry.Severity),
		moduleStyle.Render(highlightField(entry.Module)),
		highlightField(entry.Message),
	)

	if width > 0 && lipgloss.Width(line) > width {
		line = lipgloss.NewStyle().MaxWidth(width).Render(line)
	}

	return line
}
