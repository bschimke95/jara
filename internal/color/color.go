package color

import (
	"fmt"
	"image/color"

	"charm.land/lipgloss/v2"
)

// statusColors holds optional per-status color overrides set via ApplyTheme.
var statusColors map[string]color.Color

// StatusColor returns the appropriate color for a Juju status string.
// If custom status colors have been set via ApplyTheme, they are consulted first.
func StatusColor(status string) color.Color {
	if statusColors != nil {
		if c, ok := statusColors[status]; ok {
			return c
		}
	}
	switch status {
	case "active", "idle", "running", "started":
		return lipgloss.Color("#00ff00")
	case "blocked", "error", "lost", "down":
		return lipgloss.Color("#ff5555")
	case "waiting", "allocating", "pending":
		return lipgloss.Color("#ffff00")
	case "maintenance", "executing":
		return lipgloss.Color("#00bfff")
	case "terminated", "unknown":
		return lipgloss.Color("#808080")
	case "stopped":
		return lipgloss.Color("#ff4500")
	default:
		return lipgloss.Color("#c0c0c0")
	}
}

// StatusStyle returns a lipgloss style with the foreground set to the status color.
func StatusStyle(status string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(StatusColor(status))
}

// ForegroundText renders text with only a foreground color sequence and a
// foreground-only reset. This preserves any outer background highlight.
func ForegroundText(c color.Color, text string) string {
	if c == nil || text == "" {
		return text
	}
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm%s\x1b[39m", r>>8, g>>8, b>>8, text)
}

// StatusText renders a Juju status string with its semantic foreground color
// while preserving any parent-applied background style.
func StatusText(status string) string {
	return ForegroundText(StatusColor(status), status)
}

// Theme colors — k9s-inspired dark blue scheme.
var (
	// LogoColor is the bright accent for the logo / branding.
	LogoColor = lipgloss.Color("#00bfff")

	// Primary is used for table headers, active crumbs, key hints.
	Primary = lipgloss.Color("#00bfff")

	// Secondary is muted informational text.
	Secondary = lipgloss.Color("#6b7280")

	// Title is the default text color (light gray).
	Title = lipgloss.Color("#e5e7eb")

	// Subtle is used for borders, separators.
	Subtle = lipgloss.Color("#4b5563")

	// Highlight is the selected-row background.
	Highlight = lipgloss.Color("#1d4ed8")

	// Muted is for less-important text.
	Muted = lipgloss.Color("#6b7280")

	// HintKey is the color for key hint brackets and keys.
	HintKey = lipgloss.Color("#f0c674")

	// HintDesc is the color for key hint descriptions.
	HintDesc = lipgloss.Color("#6b7280")

	// CrumbFg is text inside crumb indicators.
	CrumbFg = lipgloss.Color("#ffffff")

	// CrumbBg is the crumb background.
	CrumbBg = lipgloss.Color("#1d4ed8")

	// Border is used for box borders around header and body.
	Border = lipgloss.Color("#4b5563")

	// BorderTitle is used for the title text embedded in a border.
	BorderTitle = lipgloss.Color("#00bfff")

	// InfoLabel is for dim labels like "Context:", "Cluster:".
	InfoLabel = lipgloss.Color("#6b7280")

	// InfoValue is for the values next to info labels.
	InfoValue = lipgloss.Color("#e5e7eb")

	// Error is the color used for error messages.
	Error = lipgloss.Color("#ff0000")

	// SearchHighlightFg is the foreground for search-match highlighting.
	SearchHighlightFg = lipgloss.Color("#000000")

	// SearchHighlightBg is the background for search-match highlighting.
	SearchHighlightBg = lipgloss.Color("#ffff00")

	// CrumbBgAlt is the background for secondary/context crumbs.
	CrumbBgAlt = lipgloss.Color("#374151")

	// CheckGreen is the color for positive check marks.
	CheckGreen = lipgloss.Color("#00ff00")

	// CheckRed is the color for negative/unchecked marks.
	CheckRed = lipgloss.Color("#ff5555")
)

// ApplyTheme overrides the package-level color variables from a resolved
// config.Theme. This must be called once, before the TUI starts, to apply
// user-configured colors. Nil fields in the theme are left at their defaults.
func ApplyTheme(t ThemeOverrides) {
	apply(&LogoColor, t.LogoColor)
	apply(&Primary, t.Primary)
	apply(&Secondary, t.Secondary)
	apply(&Title, t.Title)
	apply(&Subtle, t.Subtle)
	apply(&Highlight, t.Highlight)
	apply(&Muted, t.Muted)
	apply(&HintKey, t.HintKey)
	apply(&HintDesc, t.HintDesc)
	apply(&CrumbFg, t.CrumbFg)
	apply(&CrumbBg, t.CrumbBg)
	apply(&Border, t.Border)
	apply(&BorderTitle, t.BorderTitle)
	apply(&InfoLabel, t.InfoLabel)
	apply(&InfoValue, t.InfoValue)
	apply(&Error, t.Error)
	apply(&SearchHighlightFg, t.SearchHighlightFg)
	apply(&SearchHighlightBg, t.SearchHighlightBg)
	apply(&CrumbBgAlt, t.CrumbBgAlt)
	apply(&CheckGreen, t.CheckGreen)
	apply(&CheckRed, t.CheckRed)

	if t.StatusColors != nil {
		statusColors = t.StatusColors
	}
}

// ThemeOverrides carries the resolved theme colors. Non-nil fields override
// the corresponding package-level variable.
type ThemeOverrides struct {
	LogoColor         color.Color
	Primary           color.Color
	Secondary         color.Color
	Title             color.Color
	Subtle            color.Color
	Highlight         color.Color
	Muted             color.Color
	HintKey           color.Color
	HintDesc          color.Color
	CrumbFg           color.Color
	CrumbBg           color.Color
	Border            color.Color
	BorderTitle       color.Color
	InfoLabel         color.Color
	InfoValue         color.Color
	Error             color.Color
	SearchHighlightFg color.Color
	SearchHighlightBg color.Color
	CrumbBgAlt        color.Color
	CheckGreen        color.Color
	CheckRed          color.Color
	StatusColors      map[string]color.Color
}

func apply(target *color.Color, override color.Color) {
	if override != nil {
		*target = override
	}
}
