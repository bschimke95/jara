package color

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// StatusColor returns the appropriate color for a Juju status string.
func StatusColor(status string) color.Color {
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
)
