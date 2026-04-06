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

func highlightSearchMatch(line, query string, s *color.Styles) string {
	lower := strings.ToLower(line)
	lq := strings.ToLower(query)
	ql := len(query)
	hl := lipgloss.NewStyle().
		Foreground(s.SearchHighlightFgColor).
		Background(s.SearchHighlightBgColor)

	var b strings.Builder
	pos := 0
	for {
		idx := strings.Index(lower[pos:], lq)
		if idx < 0 {
			b.WriteString(line[pos:])
			break
		}
		b.WriteString(line[pos : pos+idx])
		b.WriteString(hl.Render(line[pos+idx : pos+idx+ql]))
		pos += idx + ql
	}
	return b.String()
}

func severityColor(severity string, s *color.Styles) lipgloss.Style {
	switch strings.ToUpper(severity) {
	case "ERROR", "CRITICAL":
		return lipgloss.NewStyle().Foreground(s.CheckRedColor)
	case "WARNING":
		return lipgloss.NewStyle().Foreground(s.StatusColor("waiting"))
	case "INFO":
		return lipgloss.NewStyle().Foreground(s.Primary)
	case "DEBUG":
		return lipgloss.NewStyle().Foreground(s.Muted)
	case "TRACE":
		return lipgloss.NewStyle().Foreground(s.Subtle)
	default:
		return lipgloss.NewStyle().Foreground(s.Subtle)
	}
}

func formatLogEntry(entry model.LogEntry, width int, s *color.Styles) string {
	entityStyle := lipgloss.NewStyle().Foreground(s.Primary)
	tsStyle := lipgloss.NewStyle().Foreground(s.Muted)
	sevStyle := severityColor(entry.Severity, s)
	moduleStyle := lipgloss.NewStyle().Foreground(s.Secondary)

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

func formatLogEntryHighlighted(entry model.LogEntry, width int, query string, s *color.Styles) string {
	entityStyle := lipgloss.NewStyle().Foreground(s.Primary)
	tsStyle := lipgloss.NewStyle().Foreground(s.Muted)
	sevStyle := severityColor(entry.Severity, s)
	moduleStyle := lipgloss.NewStyle().Foreground(s.Secondary)

	ts := entry.Timestamp.Format(time.TimeOnly)

	highlightField := func(str string) string {
		return highlightSearchMatch(str, query, s)
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
