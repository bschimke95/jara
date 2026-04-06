package ui

import (
	"testing"

	"github.com/bschimke95/jara/internal/color"
)

func TestLogoHeight(t *testing.T) {
	h := LogoHeight()
	if h <= 0 {
		t.Errorf("LogoHeight() = %d, want > 0", h)
	}
	// The ASCII art logo has 7 lines (1 empty + 6 character lines).
	if h != 7 {
		t.Errorf("LogoHeight() = %d, want 7", h)
	}
}

func TestStatusBar(t *testing.T) {
	s := color.DefaultStyles()
	tests := []struct {
		name          string
		resourceCount int
		resourceType  string
		width         int
	}{
		{
			name:          "basic",
			resourceCount: 5,
			resourceType:  "applications",
			width:         40,
		},
		{
			name:          "zero count",
			resourceCount: 0,
			resourceType:  "units",
			width:         30,
		},
		{
			name:          "large count",
			resourceCount: 1000,
			resourceType:  "machines",
			width:         50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StatusBar(tt.resourceCount, tt.resourceType, tt.width, s)
			if got == "" {
				t.Error("StatusBar() returned empty string")
			}
		})
	}
}

func TestHintsForView(t *testing.T) {
	keys := DefaultKeyMap()

	tests := []struct {
		viewName string
		minHints int // at least this many hints expected
	}{
		{"Controllers", 4},  // select + 3 common
		{"Models", 5},       // select + back + 3 common
		{"Model", 8},        // units + relations + logs(app) + logs + scale + 3 common
		{"Applications", 6}, // enter + logs(app) + logs + 3 common
		{"Units", 7},        // back + logs(unit) + logs + scale + 3 common
		{"Machines", 6},     // back + logs(machine) + logs + 3 common
		{"Relations", 5},    // back + logs + 3 common
		{"Debug Log", 10},   // back + bottom + top + filter + clear + search + next/prev + 3 common
		{"Unknown View", 5}, // select + back + 3 common (default case)
	}

	for _, tt := range tests {
		t.Run(tt.viewName, func(t *testing.T) {
			hints := HintsForView(tt.viewName, keys)
			if len(hints) < tt.minHints {
				t.Errorf("HintsForView(%q) returned %d hints, want >= %d", tt.viewName, len(hints), tt.minHints)
			}

			// Every hint should have non-empty Key and Desc.
			for i, h := range hints {
				if h.Key == "" {
					t.Errorf("hint[%d].Key is empty for view %q", i, tt.viewName)
				}
				if h.Desc == "" {
					t.Errorf("hint[%d].Desc is empty for view %q", i, tt.viewName)
				}
			}
		})
	}
}

func TestCrumbBar(t *testing.T) {
	s := color.DefaultStyles()

	tests := []struct {
		name   string
		crumbs []string
		width  int
	}{
		{
			name:   "empty crumbs",
			crumbs: nil,
			width:  80,
		},
		{
			name:   "single crumb",
			crumbs: []string{"Controllers"},
			width:  80,
		},
		{
			name:   "multiple crumbs",
			crumbs: []string{"Controllers", "Models", "Applications"},
			width:  120,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CrumbBar(tt.crumbs, tt.width, s)
			// Should not panic, should return a string.
			if tt.crumbs == nil && got == "" {
				// Empty crumbs returning empty-ish is acceptable.
				return
			}
			if got == "" && len(tt.crumbs) > 0 {
				t.Error("CrumbBar() returned empty string for non-empty crumbs")
			}
		})
	}
}

func TestFooter(t *testing.T) {
	s := color.DefaultStyles()

	tests := []struct {
		name       string
		hints      []KeyHint
		filterText string
		width      int
	}{
		{
			name:       "no hints no filter",
			hints:      nil,
			filterText: "",
			width:      80,
		},
		{
			name: "hints only",
			hints: []KeyHint{
				{Key: "q", Desc: "quit"},
				{Key: "?", Desc: "help"},
			},
			filterText: "",
			width:      80,
		},
		{
			name: "hints with filter",
			hints: []KeyHint{
				{Key: "q", Desc: "quit"},
			},
			filterText: "mysql",
			width:      80,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Footer(tt.hints, tt.filterText, tt.width, s)
			// Should not panic.
			_ = got
		})
	}
}
