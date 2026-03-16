package render

import (
	"testing"
	"time"

	"charm.land/bubbles/v2/table"
	"github.com/bschimke95/jara/internal/model"
)

func TestModelRows(t *testing.T) {
	models := []model.ModelSummary{
		{Name: "admin/default", ShortName: "default", Owner: "admin", Type: "iaas", UUID: "uuid-1", Current: true},
		{Name: "admin/staging", ShortName: "staging", Owner: "admin", Type: "iaas", UUID: "uuid-2"},
	}
	rows := ModelRows(models)
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	// Current model should have a star suffix.
	if rows[0][0] != "default *" {
		t.Errorf("row[0][0] = %q, want %q", rows[0][0], "default *")
	}
	if rows[1][0] != "staging" {
		t.Errorf("row[1][0] = %q, want %q", rows[1][0], "staging")
	}
}

func TestApplicationRows_Sorted(t *testing.T) {
	now := time.Now()
	apps := map[string]model.Application{
		"zulu":  {Name: "zulu", Status: "active", Charm: "zulu-charm", CharmChannel: "stable", CharmRev: 1, Scale: 1, Exposed: false, StatusMessage: "ok", Since: &now},
		"alpha": {Name: "alpha", Status: "blocked", Charm: "alpha-charm", CharmChannel: "edge", CharmRev: 2, Scale: 3, Exposed: true, StatusMessage: "blocked", Since: &now},
	}
	rows := ApplicationRows(apps)
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	// Should be sorted by name.
	if rows[0][0] != "alpha" {
		t.Errorf("first app = %q, want %q", rows[0][0], "alpha")
	}
	if rows[1][0] != "zulu" {
		t.Errorf("second app = %q, want %q", rows[1][0], "zulu")
	}
	// Check exposed field.
	if rows[0][6] != "yes" {
		t.Errorf("alpha exposed = %q, want %q", rows[0][6], "yes")
	}
	if rows[1][6] != "no" {
		t.Errorf("zulu exposed = %q, want %q", rows[1][6], "no")
	}
}

func TestControllerRows(t *testing.T) {
	controllers := []model.Controller{
		{Name: "c1", Cloud: "aws", Region: "us-east-1", Version: "3.6", Status: "available", HA: "3", Models: 4, Machines: 12, Access: "admin", Addr: "10.0.0.1:17070"},
	}
	rows := ControllerRows(controllers)
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	if rows[0][0] != "c1" {
		t.Errorf("name = %q, want %q", rows[0][0], "c1")
	}
	if rows[0][6] != "4" {
		t.Errorf("models = %q, want %q", rows[0][6], "4")
	}
}

func TestMachineRows_Sorted(t *testing.T) {
	now := time.Now()
	machines := map[string]model.Machine{
		"2": {ID: "2", Status: "started", DNSName: "dns-2", InstanceID: "i-2", Base: "ubuntu@22.04", Hardware: "cores=2", Since: &now},
		"0": {ID: "0", Status: "started", DNSName: "dns-0", InstanceID: "i-0", Base: "ubuntu@22.04", Hardware: "cores=4", Since: &now},
		"1": {ID: "1", Status: "started", DNSName: "dns-1", InstanceID: "i-1", Base: "ubuntu@22.04", Hardware: "cores=2", Since: &now,
			Containers: []model.Machine{
				{ID: "1/lxd/0", Status: "started", DNSName: "dns-lxd", InstanceID: "juju-lxd-0", Base: "ubuntu@22.04", Hardware: "cores=1"},
			},
		},
	}
	rows := MachineRows(machines)
	// 3 machines + 1 container = 4 rows.
	if len(rows) != 4 {
		t.Fatalf("got %d rows, want 4", len(rows))
	}
	if rows[0][0] != "0" {
		t.Errorf("first machine = %q, want %q", rows[0][0], "0")
	}
	if rows[1][0] != "1" {
		t.Errorf("second machine = %q, want %q", rows[1][0], "1")
	}
	if rows[2][0] != "1/lxd/0" {
		t.Errorf("container = %q, want %q", rows[2][0], "1/lxd/0")
	}
	if rows[3][0] != "2" {
		t.Errorf("third machine = %q, want %q", rows[3][0], "2")
	}
}

func TestRelationRows(t *testing.T) {
	relations := []model.Relation{
		{
			ID: 1, Key: "pg:db app:db", Interface: "pgsql", Status: "joined", Scope: "global",
			Endpoints: []model.Endpoint{
				{ApplicationName: "pg", Name: "db", Role: "provider"},
				{ApplicationName: "app", Name: "db", Role: "requirer"},
			},
		},
	}
	rows := RelationRows(relations)
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	if rows[0][0] != "1" {
		t.Errorf("id = %q, want %q", rows[0][0], "1")
	}
	if rows[0][1] != "pg:db" {
		t.Errorf("ep1 = %q, want %q", rows[0][1], "pg:db")
	}
	if rows[0][2] != "app:db" {
		t.Errorf("ep2 = %q, want %q", rows[0][2], "app:db")
	}
}

func TestRelationRowsForApp(t *testing.T) {
	relations := []model.Relation{
		{ID: 1, Endpoints: []model.Endpoint{{ApplicationName: "pg", Name: "db"}, {ApplicationName: "app", Name: "db"}}},
		{ID: 2, Endpoints: []model.Endpoint{{ApplicationName: "prom", Name: "src"}, {ApplicationName: "grafana", Name: "src"}}},
	}
	rows := RelationRowsForApp(relations, "pg")
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
}

func TestScaleColumns(t *testing.T) {
	tests := []struct {
		name           string
		cols           []table.Column
		availableWidth int
	}{
		{
			name:           "basic proportional scaling",
			cols:           []table.Column{{Title: "A", Width: 20}, {Title: "B", Width: 10}, {Title: "C", Width: 10}},
			availableWidth: 100,
		},
		{
			name:           "very small width",
			cols:           []table.Column{{Title: "A", Width: 20}, {Title: "B", Width: 10}},
			availableWidth: 10,
		},
		{
			name:           "single column",
			cols:           []table.Column{{Title: "A", Width: 50}},
			availableWidth: 100,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scaled := ScaleColumns(tt.cols, tt.availableWidth)
			if len(scaled) != len(tt.cols) {
				t.Errorf("got %d columns, want %d", len(scaled), len(tt.cols))
			}
			// Every column must have width >= 1.
			for i, c := range scaled {
				if c.Width < 1 {
					t.Errorf("column %d width = %d, want >= 1", i, c.Width)
				}
			}
		})
	}
}

func TestScaleColumns_TotalWidth(t *testing.T) {
	cols := []table.Column{
		{Title: "A", Width: 20},
		{Title: "B", Width: 30},
		{Title: "C", Width: 50},
	}
	available := 120
	scaled := ScaleColumns(cols, available)
	padding := len(scaled) * 2
	usable := available - padding

	var total int
	for _, c := range scaled {
		total += c.Width
	}
	if total != usable {
		t.Errorf("total scaled width = %d, want %d (usable)", total, usable)
	}
}

func TestIsCrossModelRelation(t *testing.T) {
	tests := []struct {
		key       string
		remoteApp string
		want      bool
	}{
		{"pg:db app:db", "app", false},
		{"admin/other.pg:db app:db", "pg", true},
		{"pg:db admin/other.app:db", "app", true},
		{"pg:db app:db", "pg", false},
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
		{"admin/other.pg:db app:db", "pg", "admin/other"},
		{"pg:db admin/staging.app:db", "app", "admin/staging"},
		{"pg:db app:db", "app", ""},
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

func TestUnitDetailRows(t *testing.T) {
	now := time.Now()
	apps := map[string]model.Application{
		"myapp": {
			Name: "myapp",
			Units: []model.Unit{
				{Name: "myapp/0", WorkloadStatus: "active", AgentStatus: "idle", Machine: "0", PublicAddress: "10.0.0.1", Ports: []string{"80/tcp"}, Leader: true, Since: &now},
				{Name: "myapp/1", WorkloadStatus: "active", AgentStatus: "idle", Machine: "1", PublicAddress: "10.0.0.2", Since: &now},
			},
		},
	}
	rows := UnitDetailRows(apps)
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
}

func TestUnitRowsForApp_IncludesMessage(t *testing.T) {
	app := model.Application{
		Name: "myapp",
		Units: []model.Unit{
			{Name: "myapp/0", WorkloadStatus: "active", WorkloadMessage: "Live master (14.12)", AgentStatus: "idle", Leader: true},
			{Name: "myapp/1", WorkloadStatus: "waiting", WorkloadMessage: "Waiting for relation", AgentStatus: "allocating"},
		},
	}
	rows := UnitRowsForApp(app)
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	wantCols := len(UnitColumns())
	for i, row := range rows {
		if len(row) != wantCols {
			t.Errorf("row %d: got %d columns, want %d", i, len(row), wantCols)
		}
	}
	// The MESSAGE column (index 3) should carry the workload message.
	if rows[1][3] != "Waiting for relation" {
		t.Errorf("row 1 message = %q, want %q", rows[1][3], "Waiting for relation")
	}
}

func TestPendingUnitRows_ScaleUp(t *testing.T) {
	units := []model.Unit{
		{Name: "app/0"}, {Name: "app/1"},
	}
	rows := PendingUnitRows("app", units, 2)
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	wantCols := len(UnitColumns())
	for i, row := range rows {
		if len(row) != wantCols {
			t.Errorf("pending row %d: got %d columns, want %d", i, len(row), wantCols)
		}
	}
}

func TestPendingUnitRows_ScaleDown(t *testing.T) {
	units := []model.Unit{
		{Name: "app/0"}, {Name: "app/1"}, {Name: "app/2"},
	}
	rows := PendingUnitRows("app", units, -2)
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
}

func TestPendingUnitRows_Zero(t *testing.T) {
	units := []model.Unit{{Name: "app/0"}}
	rows := PendingUnitRows("app", units, 0)
	if len(rows) != 0 {
		t.Fatalf("got %d rows, want 0", len(rows))
	}
}
