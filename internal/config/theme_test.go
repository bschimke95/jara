package config

import (
	"testing"

	"charm.land/lipgloss/v2"
)

func TestResolveThemeDefaults(t *testing.T) {
	theme := ResolveTheme(SkinConfig{})
	def := DefaultTheme()

	if theme.Primary != def.Primary {
		t.Errorf("Primary = %v, want %v", theme.Primary, def.Primary)
	}
	if theme.Highlight != def.Highlight {
		t.Errorf("Highlight = %v, want %v", theme.Highlight, def.Highlight)
	}
	if len(theme.StatusColors) != len(def.StatusColors) {
		t.Errorf("StatusColors length = %d, want %d", len(theme.StatusColors), len(def.StatusColors))
	}
}

func TestResolveThemeOverrides(t *testing.T) {
	skin := SkinConfig{
		Primary:   "#ff0000",
		Highlight: "#00ff00",
		Status: StatusColorsConfig{
			Active:  "#aabbcc",
			Default: "#ddeeff",
		},
	}

	theme := ResolveTheme(skin)

	if theme.Primary != lipgloss.Color("#ff0000") {
		t.Errorf("Primary = %v, want %v", theme.Primary, lipgloss.Color("#ff0000"))
	}
	if theme.Highlight != lipgloss.Color("#00ff00") {
		t.Errorf("Highlight = %v, want %v", theme.Highlight, lipgloss.Color("#00ff00"))
	}

	// Overridden status color.
	activeColor := theme.StatusColor("active")
	if activeColor != lipgloss.Color("#aabbcc") {
		t.Errorf("StatusColor(active) = %v, want %v", activeColor, lipgloss.Color("#aabbcc"))
	}

	// Default status color for unknown statuses.
	defaultColor := theme.StatusColor("some-unknown-status")
	if defaultColor != lipgloss.Color("#ddeeff") {
		t.Errorf("StatusColor(some-unknown-status) = %v, want %v", defaultColor, lipgloss.Color("#ddeeff"))
	}

	// Non-overridden fields should keep defaults.
	def := DefaultTheme()
	if theme.Border != def.Border {
		t.Errorf("Border = %v, want %v (should keep default)", theme.Border, def.Border)
	}
}

func TestThemeStatusColor(t *testing.T) {
	theme := DefaultTheme()

	tests := []struct {
		status   string
		expected string
	}{
		{"active", "#00ff00"},
		{"blocked", "#ff5555"},
		{"waiting", "#ffff00"},
		{"maintenance", "#00bfff"},
		{"terminated", "#808080"},
		{"stopped", "#ff4500"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := theme.StatusColor(tt.status)
			want := lipgloss.Color(tt.expected)
			if got != want {
				t.Errorf("StatusColor(%q) = %v, want %v", tt.status, got, want)
			}
		})
	}
}

func TestThemeStatusColorFallback(t *testing.T) {
	theme := DefaultTheme()
	got := theme.StatusColor("unknown-garbage")
	want := lipgloss.Color("#c0c0c0")
	if got != want {
		t.Errorf("StatusColor(unknown-garbage) = %v, want %v", got, want)
	}
}

func TestThemeStatusStyle(t *testing.T) {
	theme := DefaultTheme()
	style := theme.StatusStyle("active")
	if style.GetForeground() != lipgloss.Color("#00ff00") {
		t.Errorf("StatusStyle(active) foreground = %v, want %v", style.GetForeground(), lipgloss.Color("#00ff00"))
	}
}
