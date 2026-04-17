package units

import (
	"testing"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
)

func TestCompactColumns(t *testing.T) {
	cols := CompactColumns()
	if len(cols) != 4 {
		t.Fatalf("CompactColumns() returned %d, want 4", len(cols))
	}
}

func TestDetailColumns(t *testing.T) {
	cols := DetailColumns()
	if len(cols) != 7 {
		t.Fatalf("DetailColumns() returned %d, want 7", len(cols))
	}
}

func TestCompactRowsForApp(t *testing.T) {
	tests := []struct {
		name     string
		app      model.Application
		wantRows int
	}{
		{name: "no units", app: model.Application{}, wantRows: 0},
		{
			name: "two units",
			app: model.Application{
				Units: []model.Unit{
					{Name: "app/0", WorkloadStatus: "active", AgentStatus: "idle"},
					{Name: "app/1", WorkloadStatus: "waiting", AgentStatus: "executing"},
				},
			},
			wantRows: 2,
		},
		{
			name: "unit with subordinate",
			app: model.Application{
				Units: []model.Unit{
					{
						Name:           "app/0",
						WorkloadStatus: "active",
						AgentStatus:    "idle",
						Subordinates: []model.Unit{
							{Name: "nrpe/0", WorkloadStatus: "active", AgentStatus: "idle"},
						},
					},
				},
			},
			wantRows: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CompactRowsForApp(tt.app, color.DefaultStyles())
			if len(got) != tt.wantRows {
				t.Fatalf("CompactRowsForApp() len = %d, want %d", len(got), tt.wantRows)
			}
		})
	}
}

func TestDetailRowsForApp(t *testing.T) {
	app := model.Application{
		Units: []model.Unit{
			{Name: "pg/0", WorkloadStatus: "active", AgentStatus: "idle", Machine: "0", PublicAddress: "10.0.0.1", Ports: []string{"5432/tcp"}},
			{Name: "pg/1", WorkloadStatus: "active", AgentStatus: "idle", Machine: "1"},
		},
	}
	got := DetailRowsForApp(app, color.DefaultStyles())
	if len(got) != 2 {
		t.Fatalf("DetailRowsForApp() len = %d, want 2", len(got))
	}
}

func TestDetailRows(t *testing.T) {
	apps := map[string]model.Application{
		"b-app": {Units: []model.Unit{{Name: "b-app/0", WorkloadStatus: "active", AgentStatus: "idle"}}},
		"a-app": {Units: []model.Unit{{Name: "a-app/0", WorkloadStatus: "active", AgentStatus: "idle"}}},
	}
	got := DetailRows(apps, color.DefaultStyles())
	if len(got) != 2 {
		t.Fatalf("DetailRows() len = %d, want 2", len(got))
	}
}

func TestPendingCompactRows(t *testing.T) {
	tests := []struct {
		name     string
		units    []model.Unit
		delta    int
		wantRows int
	}{
		{name: "scale up by 2", units: nil, delta: 2, wantRows: 2},
		{name: "scale down by 1", units: []model.Unit{{Name: "app/0"}, {Name: "app/1"}}, delta: -1, wantRows: 0},
		{name: "no change", units: nil, delta: 0, wantRows: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PendingCompactRows("app", tt.units, tt.delta, color.DefaultStyles())
			if len(got) != tt.wantRows {
				t.Fatalf("PendingCompactRows() len = %d, want %d", len(got), tt.wantRows)
			}
		})
	}
}

func TestPendingDetailRows(t *testing.T) {
	tests := []struct {
		name     string
		units    []model.Unit
		delta    int
		wantRows int
	}{
		{name: "scale up by 3", units: nil, delta: 3, wantRows: 3},
		{name: "scale down by 2", units: []model.Unit{{Name: "app/0"}, {Name: "app/1"}, {Name: "app/2"}}, delta: -2, wantRows: 0},
		{name: "no change", units: nil, delta: 0, wantRows: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PendingDetailRows("app", tt.units, tt.delta, color.DefaultStyles())
			if len(got) != tt.wantRows {
				t.Fatalf("PendingDetailRows() len = %d, want %d", len(got), tt.wantRows)
			}
		})
	}
}
