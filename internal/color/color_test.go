package color

import (
	"image/color"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestStatusColor(t *testing.T) {
	tests := []struct {
		status   string
		expected string // hex color string
	}{
		{"active", "#00ff00"},
		{"idle", "#00ff00"},
		{"running", "#00ff00"},
		{"started", "#00ff00"},
		{"blocked", "#ff5555"},
		{"error", "#ff5555"},
		{"lost", "#ff5555"},
		{"down", "#ff5555"},
		{"waiting", "#ffff00"},
		{"allocating", "#ffff00"},
		{"pending", "#ffff00"},
		{"maintenance", "#00bfff"},
		{"executing", "#00bfff"},
		{"terminated", "#808080"},
		{"unknown", "#808080"},
		{"stopped", "#ff4500"},
		{"some-unknown-status", "#c0c0c0"},
		{"", "#c0c0c0"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := StatusColor(tt.status)
			expected := lipgloss.Color(tt.expected)
			if got != expected {
				t.Errorf("StatusColor(%q) = %v, want %v", tt.status, got, expected)
			}
		})
	}
}

func TestStatusStyle(t *testing.T) {
	tests := []string{"active", "blocked", "idle", "error", "terminated", "unknown", "waiting", "maintenance"}

	for _, status := range tests {
		t.Run(status, func(t *testing.T) {
			style := StatusStyle(status)
			expectedColor := StatusColor(status)

			// Check that the style has the correct foreground color
			if style.GetForeground() != expectedColor {
				t.Errorf("StatusStyle(%q) foreground = %v, want %v", status, style.GetForeground(), expectedColor)
			}
		})
	}
}

func TestStatusStyle_Rendering(t *testing.T) {
	statuses := []string{"active", "blocked", "idle", "error", "terminated", "unknown"}
	for _, s := range statuses {
		style := StatusStyle(s)
		rendered := style.Render("test")
		if rendered == "" {
			t.Errorf("StatusStyle(%q).Render(\"test\") returned empty string", s)
		}
		// Should contain the text we passed
		if len(rendered) < 4 {
			t.Errorf("StatusStyle(%q).Render(\"test\") too short: %q", s, rendered)
		}
	}
}

func TestThemeColors(t *testing.T) {
	// Test that our theme color constants are defined and not empty
	colors := map[string]color.Color{
		"LogoColor":   LogoColor,
		"Primary":     Primary,
		"Secondary":   Secondary,
		"Title":       Title,
		"Subtle":      Subtle,
		"Highlight":   Highlight,
		"Muted":       Muted,
		"HintKey":     HintKey,
		"HintDesc":    HintDesc,
		"CrumbFg":     CrumbFg,
		"CrumbBg":     CrumbBg,
		"Border":      Border,
		"BorderTitle": BorderTitle,
		"InfoLabel":   InfoLabel,
		"InfoValue":   InfoValue,
	}

	for name, clr := range colors {
		if clr == nil {
			t.Errorf("Theme color %s is nil", name)
		}
	}
}

func TestColorConsistency(t *testing.T) {
	// Test that StatusStyle and StatusColor return consistent results
	statuses := []string{"active", "blocked", "idle", "error", "terminated", "unknown", "waiting", "maintenance"}
	for _, status := range statuses {
		styleColor := StatusStyle(status).GetForeground()
		directColor := StatusColor(status)
		if styleColor != directColor {
			t.Errorf("StatusStyle(%q) color %v != StatusColor(%q) color %v", status, styleColor, status, directColor)
		}
	}
}

func TestStatusColorType(t *testing.T) {
	// Test that StatusColor returns the correct type
	result := StatusColor("active")
	if result == nil {
		t.Error("StatusColor should not return nil")
	}

	// Should be able to use as color.Color interface
	if result == nil {
		t.Error("StatusColor result should implement color.Color interface")
	}
}
