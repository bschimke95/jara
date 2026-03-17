package ui

import (
	"testing"

	"charm.land/bubbles/v2/table"
)

func TestScaleColumns(t *testing.T) {
	tests := []struct {
		name           string
		cols           []table.Column
		availableWidth int
	}{
		{
			name:           "basic proportional scaling",
			cols:           []table.Column{{Title: "A", Width: 20}, {Title: "B", Width: 10}, {Title: "C", Width: 10}},
			availableWidth: 100,
		},
		{
			name:           "very small width",
			cols:           []table.Column{{Title: "A", Width: 20}, {Title: "B", Width: 10}},
			availableWidth: 10,
		},
		{
			name:           "single column",
			cols:           []table.Column{{Title: "A", Width: 50}},
			availableWidth: 100,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scaled := ScaleColumns(tt.cols, tt.availableWidth)
			if len(scaled) != len(tt.cols) {
				t.Errorf("got %d columns, want %d", len(scaled), len(tt.cols))
			}
			for i, c := range scaled {
				if c.Width < 1 {
					t.Errorf("column %d width = %d, want >= 1", i, c.Width)
				}
			}
		})
	}
}

func TestScaleColumns_TotalWidth(t *testing.T) {
	cols := []table.Column{
		{Title: "A", Width: 20},
		{Title: "B", Width: 30},
		{Title: "C", Width: 50},
	}
	available := 120
	scaled := ScaleColumns(cols, available)
	padding := len(scaled) * 2
	usable := available - padding

	var total int
	for _, c := range scaled {
		total += c.Width
	}
	if total != usable {
		t.Errorf("total column width = %d, want %d (usable)", total, usable)
	}
}

func TestSortedKeys(t *testing.T) {
	tests := []struct {
		name string
		in   map[string]int
		want []string
	}{
		{"nil map", nil, nil},
		{"empty map", map[string]int{}, nil},
		{"single key", map[string]int{"a": 1}, []string{"a"}},
		{"multiple keys sorted", map[string]int{"c": 3, "a": 1, "b": 2}, []string{"a", "b", "c"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SortedKeys(tt.in)
			if len(got) != len(tt.want) {
				t.Fatalf("SortedKeys() len = %d, want %d", len(got), len(tt.want))
			}
			for i, g := range got {
				if g != tt.want[i] {
					t.Errorf("SortedKeys()[%d] = %q, want %q", i, g, tt.want[i])
				}
			}
		})
	}
}
