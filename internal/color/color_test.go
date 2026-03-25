package color

import (
	"image/color"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestDefaultStylesStatusColor(t *testing.T) {
	s := DefaultStyles()

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
			got := s.StatusColor(tt.status)
			expected := lipgloss.Color(tt.expected)
			if got != expected {
				t.Errorf("StatusColor(%q) = %v, want %v", tt.status, got, expected)
			}
		})
	}
}

func TestDefaultStylesStatusStyle(t *testing.T) {
	s := DefaultStyles()

	statuses := []string{"active", "blocked", "idle", "error", "terminated", "unknown", "waiting", "maintenance"}

	for _, status := range statuses {
		t.Run(status, func(t *testing.T) {
			style := s.StatusStyle(status)
			expectedColor := s.StatusColor(status)

			if style.GetForeground() != expectedColor {
				t.Errorf("StatusStyle(%q) foreground = %v, want %v", status, style.GetForeground(), expectedColor)
			}
		})
	}
}

func TestDefaultStylesStatusText(t *testing.T) {
	s := DefaultStyles()

	statuses := []string{"active", "blocked", "idle", "error", "terminated", "unknown"}
	for _, status := range statuses {
		rendered := s.StatusText(status)
		if rendered == "" {
			t.Errorf("StatusText(%q) returned empty string", status)
		}
		if len(rendered) < len(status) {
			t.Errorf("StatusText(%q) too short: %q", status, rendered)
		}
	}
}

func TestDefaultStylesColors(t *testing.T) {
	s := DefaultStyles()

	colors := map[string]color.Color{
		"LogoColor":    s.LogoColor,
		"Primary":      s.Primary,
		"Secondary":    s.Secondary,
		"Title":        s.Title,
		"Subtle":       s.Subtle,
		"Highlight":    s.Highlight,
		"Muted":        s.Muted,
		"HintKeyColor": s.HintKeyColor,
		"CrumbFgColor": s.CrumbFgColor,
		"CrumbBgColor": s.CrumbBgColor,
		"ErrorColor":   s.ErrorColor,
		"BorderColor":  s.BorderColor,
	}

	for name, clr := range colors {
		if clr == nil {
			t.Errorf("Color %s is nil", name)
		}
	}
}

func TestSetStatusColor(t *testing.T) {
	s := DefaultStyles()

	s.SetStatusColor("custom", lipgloss.Color("#abcdef"))
	got := s.StatusColor("custom")
	want := lipgloss.Color("#abcdef")
	if got != want {
		t.Errorf("StatusColor(custom) = %v, want %v", got, want)
	}
}

func TestRebuildStyles(t *testing.T) {
	s := DefaultStyles()

	// Override a color and rebuild.
	s.Primary = lipgloss.Color("#ff0000")
	s.RebuildStyles()

	// The Header style should now use the overridden primary color.
	if s.Header.GetForeground() != lipgloss.Color("#ff0000") {
		t.Errorf("Header foreground after rebuild = %v, want #ff0000", s.Header.GetForeground())
	}
}

func TestForegroundText(t *testing.T) {
	result := ForegroundText(lipgloss.Color("#ff0000"), "hello")
	if result == "" {
		t.Error("ForegroundText returned empty string")
	}
	if len(result) < 5 {
		t.Errorf("ForegroundText too short: %q", result)
	}
}

func TestForegroundTextNil(t *testing.T) {
	result := ForegroundText(nil, "hello")
	if result != "hello" {
		t.Errorf("ForegroundText(nil) = %q, want %q", result, "hello")
	}
}

func TestForegroundTextEmpty(t *testing.T) {
	result := ForegroundText(lipgloss.Color("#ff0000"), "")
	if result != "" {
		t.Errorf("ForegroundText empty = %q, want empty", result)
	}
}
