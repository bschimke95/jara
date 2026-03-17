package machines

import (
	"testing"

	"github.com/bschimke95/jara/internal/model"
)

func TestColumns(t *testing.T) {
	cols := columns()
	if len(cols) != 6 {
		t.Fatalf("columns() returned %d, want 6", len(cols))
	}
}

func TestMachineRows(t *testing.T) {
	tests := []struct {
		name     string
		machines map[string]model.Machine
		wantRows int
	}{
		{name: "nil", machines: nil, wantRows: 0},
		{name: "empty", machines: map[string]model.Machine{}, wantRows: 0},
		{
			name: "single machine",
			machines: map[string]model.Machine{
				"0": {ID: "0", Status: "started", DNSName: "10.0.0.1"},
			},
			wantRows: 1,
		},
		{
			name: "machine with containers",
			machines: map[string]model.Machine{
				"0": {
					ID: "0", Status: "started",
					Containers: []model.Machine{
						{ID: "0/lxd/0", Status: "started"},
						{ID: "0/lxd/1", Status: "started"},
					},
				},
			},
			wantRows: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := machineRows(tt.machines)
			if len(got) != tt.wantRows {
				t.Fatalf("machineRows() len = %d, want %d", len(got), tt.wantRows)
			}
		})
	}
}

func TestMachineRowsSortedByID(t *testing.T) {
	machines := map[string]model.Machine{
		"2": {ID: "2"},
		"0": {ID: "0"},
		"1": {ID: "1"},
	}
	got := machineRows(machines)
	want := []string{"0", "1", "2"}
	for i, row := range got {
		if row[0] != want[i] {
			t.Errorf("machineRows()[%d][0] = %q, want %q", i, row[0], want[i])
		}
	}
}
