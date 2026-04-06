package storage

import (
	"testing"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
)

func TestColumns(t *testing.T) {
	cols := Columns()
	if len(cols) != 6 {
		t.Fatalf("Columns() returned %d columns, want 6", len(cols))
	}

	want := []string{"ID", "Kind", "Owner", "Status", "Persistent", "Pool/Location"}
	for i, c := range cols {
		if c.Title != want[i] {
			t.Errorf("Columns()[%d].Title = %q, want %q", i, c.Title, want[i])
		}
		if c.Width <= 0 {
			t.Errorf("Columns()[%d].Width = %d, want > 0", i, c.Width)
		}
	}
}

func TestRows(t *testing.T) {
	styles := color.DefaultStyles()

	tests := []struct {
		name      string
		instances []model.StorageInstance
		wantLen   int
		// Check specific cell values by [row][col] for non-styled fields.
		// Styled fields (Status) are checked separately.
		checkCells map[[2]int]string
	}{
		{
			name:      "empty slice",
			instances: nil,
			wantLen:   0,
		},
		{
			name: "single instance persistent",
			instances: []model.StorageInstance{
				{
					ID:         "data/0",
					Kind:       "filesystem",
					Owner:      "mysql/0",
					Status:     "attached",
					Persistent: true,
					Pool:       "rootfs",
				},
			},
			wantLen: 1,
			checkCells: map[[2]int]string{
				{0, 0}: "data/0",
				{0, 1}: "filesystem",
				{0, 2}: "mysql/0",
				// col 3 is styled status — skip exact match
				{0, 4}: "yes",
				{0, 5}: "rootfs",
			},
		},
		{
			name: "single instance not persistent",
			instances: []model.StorageInstance{
				{
					ID:         "logs/1",
					Kind:       "block",
					Owner:      "app/1",
					Status:     "detaching",
					Persistent: false,
					Pool:       "ebs",
				},
			},
			wantLen: 1,
			checkCells: map[[2]int]string{
				{0, 0}: "logs/1",
				{0, 1}: "block",
				{0, 2}: "app/1",
				{0, 4}: "no",
				{0, 5}: "ebs",
			},
		},
		{
			name: "multiple instances",
			instances: []model.StorageInstance{
				{ID: "a/0", Kind: "filesystem", Owner: "x/0", Status: "attached", Persistent: true, Pool: "p1"},
				{ID: "b/1", Kind: "block", Owner: "y/1", Status: "error", Persistent: false, Pool: "p2"},
			},
			wantLen: 2,
			checkCells: map[[2]int]string{
				{0, 0}: "a/0",
				{0, 4}: "yes",
				{1, 0}: "b/1",
				{1, 4}: "no",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows := Rows(tt.instances, styles)
			if len(rows) != tt.wantLen {
				t.Fatalf("Rows() returned %d rows, want %d", len(rows), tt.wantLen)
			}
			for pos, want := range tt.checkCells {
				r, c := pos[0], pos[1]
				if r >= len(rows) || c >= len(rows[r]) {
					t.Errorf("cell [%d][%d] out of bounds", r, c)
					continue
				}
				if rows[r][c] != want {
					t.Errorf("row[%d][%d] = %q, want %q", r, c, rows[r][c], want)
				}
			}
			// Verify each row has exactly 6 columns.
			for i, row := range rows {
				if len(row) != 6 {
					t.Errorf("row[%d] has %d columns, want 6", i, len(row))
				}
			}
		})
	}
}
