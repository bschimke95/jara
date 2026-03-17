package controllers

import (
	"testing"

	"github.com/bschimke95/jara/internal/model"
)

func TestColumns(t *testing.T) {
	cols := columns()
	if len(cols) != 10 {
		t.Fatalf("columns() returned %d, want 10", len(cols))
	}
}

func TestControllerRows(t *testing.T) {
	tests := []struct {
		name     string
		ctrls    []model.Controller
		wantRows int
	}{
		{name: "nil", ctrls: nil, wantRows: 0},
		{name: "empty", ctrls: []model.Controller{}, wantRows: 0},
		{
			name: "two controllers",
			ctrls: []model.Controller{
				{Name: "aws-ctrl", Cloud: "aws", Region: "us-east-1", Version: "3.4.0", Status: "available", HA: "3", Models: 5, Machines: 3, Access: "superuser", Addr: "10.0.0.1"},
				{Name: "lxd-ctrl", Cloud: "lxd", Region: "default", Version: "3.4.0", Status: "available", HA: "none", Models: 2, Machines: 1, Access: "admin", Addr: "10.0.0.2"},
			},
			wantRows: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := controllerRows(tt.ctrls)
			if len(got) != tt.wantRows {
				t.Fatalf("controllerRows() len = %d, want %d", len(got), tt.wantRows)
			}
		})
	}
}

func TestControllerRowFields(t *testing.T) {
	ctrl := model.Controller{
		Name: "myctrl", Cloud: "aws", Region: "us-east-1",
		Version: "3.4.0", Status: "available", HA: "3",
		Models: 5, Machines: 3, Access: "admin", Addr: "10.0.0.1",
	}
	got := controllerRows([]model.Controller{ctrl})
	if got[0][0] != "myctrl" {
		t.Errorf("name = %q, want %q", got[0][0], "myctrl")
	}
	if got[0][6] != "5" {
		t.Errorf("models = %q, want %q", got[0][6], "5")
	}
	if got[0][7] != "3" {
		t.Errorf("machines = %q, want %q", got[0][7], "3")
	}
}
