package ui

import (
	"sort"

	"charm.land/bubbles/v2/table"

	"github.com/bschimke95/jara/internal/color"
)

// ScaleColumns adjusts column widths proportionally so their total equals availableWidth.
// Each column's cell padding (2 chars) is accounted for.
func ScaleColumns(cols []table.Column, availableWidth int) []table.Column {
	// Calculate the sum of the original (desired) widths.
	var totalDesired int
	for _, c := range cols {
		totalDesired += c.Width
	}
	if totalDesired <= 0 {
		return cols
	}

	// Account for cell padding: each column has ~2 chars of padding from the table style.
	padding := len(cols) * 2
	usable := availableWidth - padding
	if usable < len(cols) {
		usable = len(cols) // at least 1 char per column
	}

	scaled := make([]table.Column, len(cols))
	var assigned int
	for i, c := range cols {
		w := c.Width * usable / totalDesired
		if w < 1 {
			w = 1
		}
		scaled[i] = table.Column{Title: c.Title, Width: w}
		assigned += w
	}
	// Distribute any leftover to the last column.
	if diff := usable - assigned; diff > 0 && len(scaled) > 0 {
		scaled[len(scaled)-1].Width += diff
	}
	return scaled
}

// StyledTable returns the standard table styles: primary-colored bold header,
// highlighted selected row with crumb foreground.
func StyledTable(s *color.Styles) table.Styles {
	ts := table.DefaultStyles()
	ts.Header = s.Header
	ts.Selected = s.SelectedRow
	ts.Cell = s.Cell
	return ts
}

// StyledTableHighlightOnly is like StyledTable but the selected row only sets a
// background highlight without overriding the cell foreground colour. This lets
// pre-coloured status values (workload, agent) remain readable when highlighted.
func StyledTableHighlightOnly(s *color.Styles) table.Styles {
	ts := table.DefaultStyles()
	ts.Header = s.Header
	ts.Selected = s.SelectedRowBgOnly
	ts.Cell = s.Cell
	return ts
}

// UnfocusedTableStyles returns dimmed table styles for inactive/read-only panes.
// The selected style is intentionally identical to the cell style so that
// the cursor row carries no highlight in a non-interactive pane.
func UnfocusedTableStyles(s *color.Styles) table.Styles {
	ts := table.DefaultStyles()
	ts.Header = s.UnfocusedHeader
	ts.Selected = s.UnfocusedSelected
	ts.Cell = s.Cell
	return ts
}

// SortedKeys returns the sorted keys of a map.
func SortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
