package modelview

import (
	"testing"

	"github.com/bschimke95/jara/internal/model"
)

func TestApplicationColumns(t *testing.T) {
	cols := applicationColumns()
	if len(cols) != 8 {
		t.Fatalf("applicationColumns() returned %d, want 8", len(cols))
	}
}

func TestApplicationRows(t *testing.T) {
	tests := []struct {
		name     string
		apps     map[string]model.Application
		wantRows int
	}{
		{name: "nil", apps: nil, wantRows: 0},
		{name: "empty", apps: map[string]model.Application{}, wantRows: 0},
		{
			name: "two apps sorted",
			apps: map[string]model.Application{
				"zk":  {Name: "zk", Status: "active", Scale: 3},
				"app": {Name: "app", Status: "blocked", Scale: 1, Exposed: true},
			},
			wantRows: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applicationRows(tt.apps)
			if len(got) != tt.wantRows {
				t.Fatalf("applicationRows() len = %d, want %d", len(got), tt.wantRows)
			}
		})
	}
}

func TestApplicationRowsSortedOrder(t *testing.T) {
	apps := map[string]model.Application{
		"zk":  {Name: "zk"},
		"app": {Name: "app"},
		"mid": {Name: "mid"},
	}
	got := applicationRows(apps)
	want := []string{"app", "mid", "zk"}
	for i, row := range got {
		if row[0] != want[i] {
			t.Errorf("applicationRows()[%d][0] = %q, want %q", i, row[0], want[i])
		}
	}
}

func TestPadToHeight(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		height    int
		wantLines int
	}{
		{name: "pad short content", content: "a\nb", height: 5, wantLines: 5},
		{name: "truncate long content", content: "a\nb\nc\nd\ne", height: 3, wantLines: 3},
		{name: "exact fit", content: "a\nb\nc", height: 3, wantLines: 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := padToHeight(tt.content, tt.height)
			lines := 1
			for _, c := range got {
				if c == '\n' {
					lines++
				}
			}
			if lines != tt.wantLines {
				t.Errorf("padToHeight() produced %d lines, want %d", lines, tt.wantLines)
			}
		})
	}
}
