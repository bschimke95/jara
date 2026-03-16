package config

import (
	"image/color"
	"time"

	"charm.land/lipgloss/v2"

	jaracolor "github.com/bschimke95/jara/internal/color"
)

// Theme holds the resolved color theme with concrete color values.
// It is computed from the SkinConfig by applying overrides on top of defaults.
type Theme struct {
	LogoColor   color.Color
	Primary     color.Color
	Secondary   color.Color
	Title       color.Color
	Subtle      color.Color
	Highlight   color.Color
	Muted       color.Color
	HintKey     color.Color
	HintDesc    color.Color
	CrumbFg     color.Color
	CrumbBg     color.Color
	Border      color.Color
	BorderTitle color.Color
	InfoLabel   color.Color
	InfoValue   color.Color

	// Error is the color for error messages.
	Error color.Color

	// SearchHighlightFg is the foreground for search-match highlighting.
	SearchHighlightFg color.Color

	// SearchHighlightBg is the background for search-match highlighting.
	SearchHighlightBg color.Color

	// CrumbBgAlt is the background for secondary/context crumbs.
	CrumbBgAlt color.Color

	// CheckGreen is the color for positive check marks.
	CheckGreen color.Color

	// CheckRed is the color for negative/unchecked marks.
	CheckRed color.Color

	// StatusColors maps Juju status strings to colors.
	StatusColors map[string]color.Color
}

// DefaultTheme returns the compiled-in k9s-inspired dark blue theme.
func DefaultTheme() *Theme {
	return &Theme{
		LogoColor:   lipgloss.Color("#00bfff"),
		Primary:     lipgloss.Color("#00bfff"),
		Secondary:   lipgloss.Color("#6b7280"),
		Title:       lipgloss.Color("#e5e7eb"),
		Subtle:      lipgloss.Color("#4b5563"),
		Highlight:   lipgloss.Color("#1d4ed8"),
		Muted:       lipgloss.Color("#6b7280"),
		HintKey:     lipgloss.Color("#f0c674"),
		HintDesc:    lipgloss.Color("#6b7280"),
		CrumbFg:     lipgloss.Color("#ffffff"),
		CrumbBg:     lipgloss.Color("#1d4ed8"),
		Border:      lipgloss.Color("#4b5563"),
		BorderTitle: lipgloss.Color("#00bfff"),
		InfoLabel:   lipgloss.Color("#6b7280"),
		InfoValue:   lipgloss.Color("#e5e7eb"),
		Error:             lipgloss.Color("#ff0000"),
		SearchHighlightFg: lipgloss.Color("#000000"),
		SearchHighlightBg: lipgloss.Color("#ffff00"),
		CrumbBgAlt:        lipgloss.Color("#374151"),
		CheckGreen:        lipgloss.Color("#00ff00"),
		CheckRed:          lipgloss.Color("#ff5555"),
		StatusColors: map[string]color.Color{
			"active":      lipgloss.Color("#00ff00"),
			"idle":        lipgloss.Color("#00ff00"),
			"running":     lipgloss.Color("#00ff00"),
			"started":     lipgloss.Color("#00ff00"),
			"blocked":     lipgloss.Color("#ff5555"),
			"error":       lipgloss.Color("#ff5555"),
			"lost":        lipgloss.Color("#ff5555"),
			"down":        lipgloss.Color("#ff5555"),
			"waiting":     lipgloss.Color("#ffff00"),
			"allocating":  lipgloss.Color("#ffff00"),
			"pending":     lipgloss.Color("#ffff00"),
			"maintenance": lipgloss.Color("#00bfff"),
			"executing":   lipgloss.Color("#00bfff"),
			"terminated":  lipgloss.Color("#808080"),
			"unknown":     lipgloss.Color("#808080"),
			"stopped":     lipgloss.Color("#ff4500"),
		},
	}
}

// ResolveTheme builds a Theme by merging SkinConfig overrides on top of defaults.
func ResolveTheme(skin SkinConfig) *Theme {
	t := DefaultTheme()

	applyColor(&t.LogoColor, skin.LogoColor)
	applyColor(&t.Primary, skin.Primary)
	applyColor(&t.Secondary, skin.Secondary)
	applyColor(&t.Title, skin.Title)
	applyColor(&t.Subtle, skin.Subtle)
	applyColor(&t.Highlight, skin.Highlight)
	applyColor(&t.Muted, skin.Muted)
	applyColor(&t.HintKey, skin.HintKey)
	applyColor(&t.HintDesc, skin.HintDesc)
	applyColor(&t.CrumbFg, skin.CrumbFg)
	applyColor(&t.CrumbBg, skin.CrumbBg)
	applyColor(&t.Border, skin.Border)
	applyColor(&t.BorderTitle, skin.BorderTitle)
	applyColor(&t.InfoLabel, skin.InfoLabel)
	applyColor(&t.InfoValue, skin.InfoValue)
	applyColor(&t.Error, skin.Error)
	applyColor(&t.SearchHighlightFg, skin.SearchHighlightFg)
	applyColor(&t.SearchHighlightBg, skin.SearchHighlightBg)
	applyColor(&t.CrumbBgAlt, skin.CrumbBgAlt)
	applyColor(&t.CheckGreen, skin.CheckGreen)
	applyColor(&t.CheckRed, skin.CheckRed)

	// Apply status color overrides.
	applyStatusColor(t.StatusColors, "active", skin.Status.Active)
	applyStatusColor(t.StatusColors, "idle", skin.Status.Idle)
	applyStatusColor(t.StatusColors, "running", skin.Status.Running)
	applyStatusColor(t.StatusColors, "started", skin.Status.Started)
	applyStatusColor(t.StatusColors, "blocked", skin.Status.Blocked)
	applyStatusColor(t.StatusColors, "error", skin.Status.Error)
	applyStatusColor(t.StatusColors, "lost", skin.Status.Lost)
	applyStatusColor(t.StatusColors, "down", skin.Status.Down)
	applyStatusColor(t.StatusColors, "waiting", skin.Status.Waiting)
	applyStatusColor(t.StatusColors, "allocating", skin.Status.Allocating)
	applyStatusColor(t.StatusColors, "pending", skin.Status.Pending)
	applyStatusColor(t.StatusColors, "maintenance", skin.Status.Maintenance)
	applyStatusColor(t.StatusColors, "executing", skin.Status.Executing)
	applyStatusColor(t.StatusColors, "terminated", skin.Status.Terminated)
	applyStatusColor(t.StatusColors, "unknown", skin.Status.Unknown)
	applyStatusColor(t.StatusColors, "stopped", skin.Status.Stopped)
	if skin.Status.Default != "" {
		t.StatusColors[""] = lipgloss.Color(skin.Status.Default)
	}

	return t
}

// Apply writes the resolved theme into the color package's global variables.
// This must be called once before the TUI starts.
func (t *Theme) Apply() {
	jaracolor.ApplyTheme(jaracolor.ThemeOverrides{
		LogoColor:    t.LogoColor,
		Primary:      t.Primary,
		Secondary:    t.Secondary,
		Title:        t.Title,
		Subtle:       t.Subtle,
		Highlight:    t.Highlight,
		Muted:        t.Muted,
		HintKey:      t.HintKey,
		HintDesc:     t.HintDesc,
		CrumbFg:      t.CrumbFg,
		CrumbBg:      t.CrumbBg,
		Border:       t.Border,
		BorderTitle:  t.BorderTitle,
		InfoLabel:         t.InfoLabel,
		InfoValue:         t.InfoValue,
		Error:             t.Error,
		SearchHighlightFg: t.SearchHighlightFg,
		SearchHighlightBg: t.SearchHighlightBg,
		CrumbBgAlt:        t.CrumbBgAlt,
		CheckGreen:        t.CheckGreen,
		CheckRed:          t.CheckRed,
		StatusColors:      t.StatusColors,
	})
}

// StatusColor returns the appropriate color for a Juju status string.
func (t *Theme) StatusColor(status string) color.Color {
	if c, ok := t.StatusColors[status]; ok {
		return c
	}
	if c, ok := t.StatusColors[""]; ok {
		return c
	}
	return lipgloss.Color("#c0c0c0")
}

// StatusStyle returns a lipgloss style with the foreground set to the status color.
func (t *Theme) StatusStyle(status string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.StatusColor(status))
}

// RefreshDuration returns the config's refresh rate as a time.Duration.
func (c *Config) RefreshDuration() time.Duration {
	rate := c.Jara.RefreshRate
	if rate <= 0 {
		rate = DefaultRefreshRate
	}
	return time.Duration(rate * float64(time.Second))
}

func applyColor(target *color.Color, override string) {
	if override != "" {
		*target = lipgloss.Color(override)
	}
}

func applyStatusColor(m map[string]color.Color, key, override string) {
	if override != "" {
		m[key] = lipgloss.Color(override)
	}
}
