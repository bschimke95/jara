package applications

import (
	"testing"

	"github.com/bschimke95/jara/internal/model"
)

func TestColumns(t *testing.T) {
	cols := columns()
	if len(cols) != 8 {
		t.Fatalf("columns() returned %d columns, want 8", len(cols))
	}
	want := []string{"NAME", "STATUS", "CHARM", "CHANNEL", "REV", "SCALE", "EXPOSED", "MESSAGE"}
	for i, col := range cols {
		if col.Title != want[i] {
			t.Errorf("columns()[%d].Title = %q, want %q", i, col.Title, want[i])
		}
	}
}

func TestRows(t *testing.T) {
	tests := []struct {
		name     string
		apps     map[string]model.Application
		wantRows int
	}{
		{
			name:     "nil map",
			apps:     nil,
			wantRows: 0,
		},
		{
			name:     "empty map",
			apps:     map[string]model.Application{},
			wantRows: 0,
		},
		{
			name: "single app",
			apps: map[string]model.Application{
				"postgresql": {Name: "postgresql", Status: "active", Charm: "postgresql", CharmChannel: "14/stable", CharmRev: 363, Scale: 1, Exposed: false},
			},
			wantRows: 1,
		},
		{
			name: "multiple apps sorted",
			apps: map[string]model.Application{
				"zookeeper":  {Name: "zookeeper", Status: "active"},
				"alpine":     {Name: "alpine", Status: "blocked"},
				"postgresql": {Name: "postgresql", Status: "waiting"},
			},
			wantRows: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rows(tt.apps)
			if len(got) != tt.wantRows {
				t.Fatalf("rows() returned %d rows, want %d", len(got), tt.wantRows)
			}
		})
	}
}

func TestRowsSortedOrder(t *testing.T) {
	apps := map[string]model.Application{
		"zookeeper":  {Name: "zookeeper"},
		"alpine":     {Name: "alpine"},
		"postgresql": {Name: "postgresql"},
	}
	got := rows(apps)
	want := []string{"alpine", "postgresql", "zookeeper"}
	for i, row := range got {
		if row[0] != want[i] {
			t.Errorf("rows()[%d][0] = %q, want %q", i, row[0], want[i])
		}
	}
}

func TestRowsExposedField(t *testing.T) {
	tests := []struct {
		name    string
		exposed bool
		want    string
	}{
		{"exposed true", true, "yes"},
		{"exposed false", false, "no"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apps := map[string]model.Application{
				"app": {Name: "app", Exposed: tt.exposed},
			}
			got := rows(apps)
			if got[0][6] != tt.want {
				t.Errorf("exposed field = %q, want %q", got[0][6], tt.want)
			}
		})
	}
}
