package config

import (
	"testing"

	"charm.land/lipgloss/v2"
)

func TestResolveStylesDefaults(t *testing.T) {
	s := ResolveStyles(SkinConfig{})

	// Default primary should match the Atom One Dark value.
	if s.Primary != lipgloss.Color("#61afef") {
		t.Errorf("Primary = %v, want %v", s.Primary, lipgloss.Color("#61afef"))
	}
	if s.Highlight != lipgloss.Color("#3e4451") {
		t.Errorf("Highlight = %v, want %v", s.Highlight, lipgloss.Color("#3e4451"))
	}
}

func TestResolveStylesOverrides(t *testing.T) {
	skin := SkinConfig{
		Primary:   "#ff0000",
		Highlight: "#00ff00",
		Status: StatusColorsConfig{
			Active:  "#aabbcc",
			Default: "#ddeeff",
		},
	}

	s := ResolveStyles(skin)

	if s.Primary != lipgloss.Color("#ff0000") {
		t.Errorf("Primary = %v, want %v", s.Primary, lipgloss.Color("#ff0000"))
	}
	if s.Highlight != lipgloss.Color("#00ff00") {
		t.Errorf("Highlight = %v, want %v", s.Highlight, lipgloss.Color("#00ff00"))
	}

	// Overridden status color.
	activeColor := s.StatusColor("active")
	if activeColor != lipgloss.Color("#aabbcc") {
		t.Errorf("StatusColor(active) = %v, want %v", activeColor, lipgloss.Color("#aabbcc"))
	}

	// Default status color for unknown statuses.
	defaultColor := s.StatusColor("some-unknown-status")
	if defaultColor != lipgloss.Color("#ddeeff") {
		t.Errorf("StatusColor(some-unknown-status) = %v, want %v", defaultColor, lipgloss.Color("#ddeeff"))
	}

	// Non-overridden fields should keep defaults.
	if s.BorderColor != lipgloss.Color("#4b5263") {
		t.Errorf("BorderColor = %v, want default", s.BorderColor)
	}
}

func TestResolveStylesStatusColor(t *testing.T) {
	s := ResolveStyles(SkinConfig{})

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
			got := s.StatusColor(tt.status)
			want := lipgloss.Color(tt.expected)
			if got != want {
				t.Errorf("StatusColor(%q) = %v, want %v", tt.status, got, want)
			}
		})
	}
}

func TestResolveStylesStatusColorFallback(t *testing.T) {
	s := ResolveStyles(SkinConfig{})
	got := s.StatusColor("unknown-garbage")
	want := lipgloss.Color("#c0c0c0")
	if got != want {
		t.Errorf("StatusColor(unknown-garbage) = %v, want %v", got, want)
	}
}

func TestResolveStylesStatusStyle(t *testing.T) {
	s := ResolveStyles(SkinConfig{})
	style := s.StatusStyle("active")
	if style.GetForeground() != lipgloss.Color("#00ff00") {
		t.Errorf("StatusStyle(active) foreground = %v, want %v", style.GetForeground(), lipgloss.Color("#00ff00"))
	}
}
