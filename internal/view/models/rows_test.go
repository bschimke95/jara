package models

import (
	"testing"

	"github.com/bschimke95/jara/internal/model"
)

func TestColumns(t *testing.T) {
	cols := columns()
	if len(cols) != 4 {
		t.Fatalf("columns() returned %d, want 4", len(cols))
	}
	want := []string{"NAME", "OWNER", "TYPE", "UUID"}
	for i, c := range cols {
		if c.Title != want[i] {
			t.Errorf("columns()[%d].Title = %q, want %q", i, c.Title, want[i])
		}
	}
}

func TestModelRows(t *testing.T) {
	tests := []struct {
		name     string
		models   []model.ModelSummary
		wantRows int
	}{
		{name: "nil", models: nil, wantRows: 0},
		{name: "empty", models: []model.ModelSummary{}, wantRows: 0},
		{
			name: "two models",
			models: []model.ModelSummary{
				{Name: "admin/default", ShortName: "default", Owner: "admin", Type: "iaas", UUID: "abc-123"},
				{Name: "admin/prod", ShortName: "prod", Owner: "admin", Type: "iaas", UUID: "def-456"},
			},
			wantRows: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := modelRows(tt.models)
			if len(got) != tt.wantRows {
				t.Fatalf("modelRows() len = %d, want %d", len(got), tt.wantRows)
			}
		})
	}
}

func TestModelRowsCurrentMarker(t *testing.T) {
	tests := []struct {
		name    string
		current bool
		want    string
	}{
		{name: "not current", current: false, want: "default"},
		{name: "current model", current: true, want: "default *"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mdls := []model.ModelSummary{{ShortName: "default", Current: tt.current}}
			got := modelRows(mdls)
			if got[0][0] != tt.want {
				t.Errorf("name field = %q, want %q", got[0][0], tt.want)
			}
		})
	}
}
