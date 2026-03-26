package app

import (
	"context"
	"testing"

	"charm.land/bubbles/v2/key"

	"github.com/bschimke95/jara/internal/api"
	"github.com/bschimke95/jara/internal/config"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/ui"
)

func TestDeployApplicationReadOnly(t *testing.T) {
	cfg := config.NewDefault()
	cfg.Jara.ReadOnly = true

	m := Model{
		client: api.NewMockClient(),
		cfg:    cfg,
	}

	msg := m.deployApplication("", model.DeployOptions{CharmName: "redis-k8s", ApplicationName: "redis"})()
	if msg == nil {
		t.Fatal("expected error message in read-only mode")
	}
	if _, ok := msg.(errMsg); !ok {
		t.Fatalf("msg type = %T, want errMsg", msg)
	}
}

func TestScaleApplicationReadOnly(t *testing.T) {
	cfg := config.NewDefault()
	cfg.Jara.ReadOnly = true

	client := api.NewMockClient()
	_ = client.SelectModel("admin/default")

	m := Model{
		client: client,
		cfg:    cfg,
	}

	msg := m.scaleApplication("postgresql", 1)()
	if msg == nil {
		t.Fatal("expected error message in read-only mode")
	}
	if _, ok := msg.(errMsg); !ok {
		t.Fatalf("msg type = %T, want errMsg", msg)
	}
}

func TestDeployApplicationTargetsModel(t *testing.T) {
	client := api.NewMockClient()
	m := Model{
		client: client,
		cfg:    config.NewDefault(),
	}

	msg := m.deployApplication("admin/default", model.DeployOptions{CharmName: "redis-k8s", ApplicationName: "redis"})()
	if msg != nil {
		t.Fatalf("deploy command returned unexpected message: %T", msg)
	}

	status, err := client.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() failed: %v", err)
	}
	if _, exists := status.Applications["redis"]; !exists {
		t.Fatal("expected redis application after deploy")
	}
}

// TestBuildHeaderHints verifies the 6-hint cap, view-specific priority, and
// that the help hint is always the last element.
func TestBuildHeaderHints(t *testing.T) {
	m := Model{keys: ui.DefaultKeyMap()}

	bk := func(b key.Binding) string { return b.Help().Key }
	helpHintKey := bk(m.keys.Help)

	tests := []struct {
		name      string
		viewHints []ui.KeyHint
		wantLen   int
	}{
		{
			name:      "no view hints",
			viewHints: nil,
			wantLen:   3, // cmd + quit + help
		},
		{
			name:      "one view hint",
			viewHints: []ui.KeyHint{{Key: "enter", Desc: "select"}},
			wantLen:   4, // view + cmd + quit + help
		},
		{
			name: "five view hints leaves only help as general",
			viewHints: []ui.KeyHint{
				{Key: "a", Desc: "1"},
				{Key: "b", Desc: "2"},
				{Key: "c", Desc: "3"},
				{Key: "d", Desc: "4"},
				{Key: "e", Desc: "5"},
			},
			wantLen: 6, // 5 view + help
		},
		{
			name: "more than five view hints truncated to 5 + help",
			viewHints: []ui.KeyHint{
				{Key: "a", Desc: "1"},
				{Key: "b", Desc: "2"},
				{Key: "c", Desc: "3"},
				{Key: "d", Desc: "4"},
				{Key: "e", Desc: "5"},
				{Key: "f", Desc: "6"},
				{Key: "g", Desc: "7"},
			},
			wantLen: 6, // capped at 6
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hints := m.buildHeaderHints(tt.viewHints)
			if len(hints) != tt.wantLen {
				t.Errorf("buildHeaderHints() len = %d, want %d; hints = %v", len(hints), tt.wantLen, hints)
			}
			// Help must always be the last element.
			last := hints[len(hints)-1]
			if last.Key != helpHintKey {
				t.Errorf("last hint key = %q, want help key %q", last.Key, helpHintKey)
			}
		})
	}
}
