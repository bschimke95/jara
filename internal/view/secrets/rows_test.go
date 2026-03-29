package secrets

import (
	"testing"
	"time"

	"github.com/bschimke95/jara/internal/model"
)

func TestColumns(t *testing.T) {
	cols := Columns()
	if len(cols) != 6 {
		t.Fatalf("Columns() returned %d, want 6", len(cols))
	}
}

func TestRows(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		secrets  []model.Secret
		wantRows int
	}{
		{name: "nil", secrets: nil, wantRows: 0},
		{name: "empty", secrets: []model.Secret{}, wantRows: 0},
		{
			name: "two secrets",
			secrets: []model.Secret{
				{URI: "secret:abc123", Label: "db-pass", Owner: "application-pg", RotatePolicy: "monthly", Revision: 3, UpdateTime: now},
				{URI: "secret:def456", Label: "api-key", Owner: "application-grafana", RotatePolicy: "never", Revision: 1, UpdateTime: now},
			},
			wantRows: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Rows(tt.secrets)
			if len(got) != tt.wantRows {
				t.Fatalf("Rows() len = %d, want %d", len(got), tt.wantRows)
			}
		})
	}
}

func TestRowsForApp(t *testing.T) {
	now := time.Now()
	secrets := []model.Secret{
		{URI: "secret:abc123", Label: "db-pass", Owner: "application-postgresql", Revision: 3, UpdateTime: now},
		{URI: "secret:def456", Label: "api-key", Owner: "application-grafana", Revision: 1, UpdateTime: now},
		{URI: "secret:ghi789", Label: "tls-cert", Owner: "application-postgresql", Revision: 2, UpdateTime: now},
	}
	got := RowsForApp(secrets, "postgresql")
	if len(got) != 2 {
		t.Fatalf("RowsForApp(postgresql) len = %d, want 2", len(got))
	}
}

func TestRowsForApp_noMatch(t *testing.T) {
	now := time.Now()
	secrets := []model.Secret{
		{URI: "secret:abc123", Owner: "application-postgresql", UpdateTime: now},
	}
	got := RowsForApp(secrets, "grafana")
	if len(got) != 0 {
		t.Fatalf("RowsForApp(grafana) len = %d, want 0", len(got))
	}
}

func TestFormatOwner(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "application-postgresql", want: "postgresql"},
		{input: "unit-mysql-0", want: "mysql-0"},
		{input: "model-admin", want: "admin"},
		{input: "unknown-entity", want: "unknown-entity"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := formatOwner(tt.input)
			if got != tt.want {
				t.Errorf("formatOwner(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
