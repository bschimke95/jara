package config

import (
	imgcolor "image/color"

	"charm.land/lipgloss/v2"

	"github.com/bschimke95/jara/internal/color"
)

// ResolveStyles builds a color.Styles by merging SkinConfig overrides on top
// of the compiled-in defaults. This replaces the former Theme/ThemeOverrides
// indirection: SkinConfig → Styles in one step.
func ResolveStyles(skin SkinConfig) *color.Styles {
	// Start from the default Atom One Dark styles.
	s := color.DefaultStyles()

	// Apply palette-level overrides from config hex strings.
	applyColorOverride(&s.LogoColor, skin.LogoColor)
	applyColorOverride(&s.Primary, skin.Primary)
	applyColorOverride(&s.Secondary, skin.Secondary)
	applyColorOverride(&s.Title, skin.Title)
	applyColorOverride(&s.Subtle, skin.Subtle)
	applyColorOverride(&s.Highlight, skin.Highlight)
	applyColorOverride(&s.Muted, skin.Muted)
	applyColorOverride(&s.HintKeyColor, skin.HintKey)
	applyColorOverride(&s.HintDescColor, skin.HintDesc)
	applyColorOverride(&s.CrumbFgColor, skin.CrumbFg)
	applyColorOverride(&s.CrumbBgColor, skin.CrumbBg)
	applyColorOverride(&s.CrumbBgAltColor, skin.CrumbBgAlt)
	applyColorOverride(&s.SearchHighlightFgColor, skin.SearchHighlightFg)
	applyColorOverride(&s.SearchHighlightBgColor, skin.SearchHighlightBg)
	applyColorOverride(&s.ErrorColor, skin.Error)
	applyColorOverride(&s.CheckGreenColor, skin.CheckGreen)
	applyColorOverride(&s.CheckRedColor, skin.CheckRed)
	applyColorOverride(&s.AssistantLabelColor, skin.AssistantLabel)
	applyColorOverride(&s.InfoLabelColor, skin.InfoLabel)
	applyColorOverride(&s.InfoValueColor, skin.InfoValue)
	applyColorOverride(&s.BorderColor, skin.Border)
	applyColorOverride(&s.BorderTitleColor, skin.BorderTitle)

	// Rebuild all styles that depend on overridden colors.
	s.RebuildStyles()

	// Apply status color overrides.
	applyStatusColorOverride(s, "active", skin.Status.Active)
	applyStatusColorOverride(s, "idle", skin.Status.Idle)
	applyStatusColorOverride(s, "running", skin.Status.Running)
	applyStatusColorOverride(s, "started", skin.Status.Started)
	applyStatusColorOverride(s, "blocked", skin.Status.Blocked)
	applyStatusColorOverride(s, "error", skin.Status.Error)
	applyStatusColorOverride(s, "lost", skin.Status.Lost)
	applyStatusColorOverride(s, "down", skin.Status.Down)
	applyStatusColorOverride(s, "waiting", skin.Status.Waiting)
	applyStatusColorOverride(s, "allocating", skin.Status.Allocating)
	applyStatusColorOverride(s, "pending", skin.Status.Pending)
	applyStatusColorOverride(s, "maintenance", skin.Status.Maintenance)
	applyStatusColorOverride(s, "executing", skin.Status.Executing)
	applyStatusColorOverride(s, "terminated", skin.Status.Terminated)
	applyStatusColorOverride(s, "unknown", skin.Status.Unknown)
	applyStatusColorOverride(s, "stopped", skin.Status.Stopped)
	if skin.Status.Default != "" {
		s.SetStatusColor("", lipgloss.Color(skin.Status.Default))
	}

	return s
}

func applyStatusColorOverride(s *color.Styles, key, override string) {
	if override != "" {
		s.SetStatusColor(key, lipgloss.Color(override))
	}
}

func applyColorOverride(target *imgcolor.Color, override string) {
	if override != "" {
		*target = lipgloss.Color(override)
	}
}
