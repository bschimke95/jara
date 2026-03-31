package relations

import (
	"testing"

	"github.com/bschimke95/jara/internal/model"
)

func TestColumns(t *testing.T) {
	cols := Columns()
	if len(cols) != 6 {
		t.Fatalf("Columns() returned %d, want 6", len(cols))
	}
}

func TestCompactColumn(t *testing.T) {
	cols := CompactColumn()
	if len(cols) != 1 {
		t.Fatalf("CompactColumn() returned %d, want 1", len(cols))
	}
}

func TestRows(t *testing.T) {
	tests := []struct {
		name     string
		rels     []model.Relation
		wantRows int
	}{
		{name: "nil", rels: nil, wantRows: 0},
		{name: "empty", rels: []model.Relation{}, wantRows: 0},
		{
			name: "two relations",
			rels: []model.Relation{
				{ID: 1, Endpoints: []model.Endpoint{{ApplicationName: "pg", Name: "db"}, {ApplicationName: "app", Name: "db"}}, Interface: "pgsql", Scope: "global"},
				{ID: 2, Endpoints: []model.Endpoint{{ApplicationName: "app", Name: "juju-info"}, {ApplicationName: "nrpe", Name: "general-info"}}, Interface: "juju-info", Scope: "container"},
			},
			wantRows: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Rows(tt.rels)
			if len(got) != tt.wantRows {
				t.Fatalf("Rows() len = %d, want %d", len(got), tt.wantRows)
			}
		})
	}
}

func TestRowsForApp(t *testing.T) {
	rels := []model.Relation{
		{ID: 1, Endpoints: []model.Endpoint{{ApplicationName: "pg", Name: "db"}, {ApplicationName: "app", Name: "db"}}},
		{ID: 2, Endpoints: []model.Endpoint{{ApplicationName: "redis", Name: "cache"}, {ApplicationName: "worker", Name: "cache"}}},
	}
	got := RowsForApp(rels, "pg")
	if len(got) != 1 {
		t.Fatalf("RowsForApp(pg) len = %d, want 1", len(got))
	}
}

func TestCompactRowsForApp(t *testing.T) {
	rels := []model.Relation{
		{
			ID:        1,
			Key:       "pg:db app:db",
			Interface: "pgsql",
			Endpoints: []model.Endpoint{
				{ApplicationName: "pg", Name: "db"},
				{ApplicationName: "app", Name: "db"},
			},
		},
		{
			ID:        2,
			Key:       "redis:cache worker:cache",
			Interface: "redis",
			Endpoints: []model.Endpoint{
				{ApplicationName: "redis", Name: "cache"},
				{ApplicationName: "worker", Name: "cache"},
			},
		},
	}
	got := CompactRowsForApp(rels, "pg")
	if len(got) != 1 {
		t.Fatalf("CompactRowsForApp(pg) len = %d, want 1", len(got))
	}
}

func TestIsCrossModelRelation(t *testing.T) {
	tests := []struct {
		key       string
		remoteApp string
		want      bool
	}{
		{key: "pg:db app:db", remoteApp: "app", want: false},
		{key: "admin/other.pg:db app:db", remoteApp: "pg", want: true},
		{key: "pg:db admin/other.app:db", remoteApp: "app", want: true},
		{key: "pg:db app:db", remoteApp: "pg", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := isCrossModelRelation(tt.key, tt.remoteApp)
			if got != tt.want {
				t.Errorf("isCrossModelRelation(%q, %q) = %v, want %v", tt.key, tt.remoteApp, got, tt.want)
			}
		})
	}
}

func TestExtractModelPrefix(t *testing.T) {
	tests := []struct {
		key       string
		remoteApp string
		want      string
	}{
		{key: "admin/other.pg:db app:db", remoteApp: "pg", want: "admin/other"},
		{key: "pg:db admin/staging.app:db", remoteApp: "app", want: "admin/staging"},
		{key: "pg:db app:db", remoteApp: "app", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := extractModelPrefix(tt.key, tt.remoteApp)
			if got != tt.want {
				t.Errorf("extractModelPrefix(%q, %q) = %q, want %q", tt.key, tt.remoteApp, got, tt.want)
			}
		})
	}
}

func TestMatchesFilter(t *testing.T) {
	rel := model.Relation{
		ID:        1,
		Interface: "pgsql",
		Scope:     "global",
		Status:    "joined",
		Endpoints: []model.Endpoint{
			{ApplicationName: "postgresql", Name: "db"},
			{ApplicationName: "ubuntu-app", Name: "db"},
		},
	}

	tests := []struct {
		filter string
		want   bool
	}{
		{filter: "post", want: true},
		{filter: "ubuntu", want: true},
		{filter: "pgsql", want: true},
		{filter: "global", want: true},
		{filter: "joined", want: true},
		{filter: "redis", want: false},
		{filter: "db", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.filter, func(t *testing.T) {
			got := matchesFilter(rel, tt.filter)
			if got != tt.want {
				t.Errorf("matchesFilter(%q) = %v, want %v", tt.filter, got, tt.want)
			}
		})
	}
}
