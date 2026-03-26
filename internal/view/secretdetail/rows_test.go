package secretdetail

import (
	"testing"
	"time"

	"github.com/bschimke95/jara/internal/model"
)

func TestRevisionColumns(t *testing.T) {
	cols := RevisionColumns()
	if len(cols) != 4 {
		t.Fatalf("RevisionColumns() returned %d, want 4", len(cols))
	}
}

func TestAccessColumns(t *testing.T) {
	cols := AccessColumns()
	if len(cols) != 3 {
		t.Fatalf("AccessColumns() returned %d, want 3", len(cols))
	}
}

func TestRevisionRows(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)
	tests := []struct {
		name     string
		revs     []model.SecretRevision
		wantRows int
	}{
		{name: "nil", revs: nil, wantRows: 0},
		{name: "empty", revs: []model.SecretRevision{}, wantRows: 0},
		{
			name: "two revisions",
			revs: []model.SecretRevision{
				{Revision: 1, CreatedAt: past, ExpiredAt: &now, Backend: "internal"},
				{Revision: 2, CreatedAt: now, Backend: "internal"},
			},
			wantRows: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RevisionRows(tt.revs)
			if len(got) != tt.wantRows {
				t.Fatalf("RevisionRows() len = %d, want %d", len(got), tt.wantRows)
			}
		})
	}
}

func TestRevisionRows_expiredFormatting(t *testing.T) {
	now := time.Now()
	rows := RevisionRows([]model.SecretRevision{
		{Revision: 1, CreatedAt: now, Backend: "vault"},
	})
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0][2] != "-" {
		t.Errorf("expired column = %q, want %q", rows[0][2], "-")
	}
}

func TestAccessRows(t *testing.T) {
	tests := []struct {
		name     string
		access   []model.SecretAccessInfo
		wantRows int
	}{
		{name: "nil", access: nil, wantRows: 0},
		{name: "empty", access: []model.SecretAccessInfo{}, wantRows: 0},
		{
			name: "one entry",
			access: []model.SecretAccessInfo{
				{Target: "application-ubuntu-app", Scope: "relation-1", Role: "consume"},
			},
			wantRows: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AccessRows(tt.access)
			if len(got) != tt.wantRows {
				t.Fatalf("AccessRows() len = %d, want %d", len(got), tt.wantRows)
			}
		})
	}
}
