package secrets

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/nav"
	"github.com/bschimke95/jara/internal/ui"
	"github.com/bschimke95/jara/internal/view"
)

func TestEnterDrillsDown(t *testing.T) {
	v := New(ui.DefaultKeyMap(), color.DefaultStyles())
	v.SetSize(120, 30)

	now := time.Now()
	v.SetStatus(&model.FullStatus{
		Secrets: []model.Secret{
			{URI: "secret:abc123", Label: "db-pass", Owner: "application-pg", UpdateTime: now},
			{URI: "secret:def456", Label: "api-key", Owner: "application-grafana", UpdateTime: now},
		},
	})

	_, cmd := v.Update(tea.KeyPressMsg{Text: "enter", Code: 0x0d})
	if cmd == nil {
		t.Fatal("expected navigation command on Enter")
	}
	msg := cmd()
	navMsg, ok := msg.(view.NavigateMsg)
	if !ok {
		t.Fatalf("msg type = %T, want view.NavigateMsg", msg)
	}
	if navMsg.Target != nav.SecretDetailView {
		t.Fatalf("navigate target = %v, want %v", navMsg.Target, nav.SecretDetailView)
	}
	if navMsg.Context != "secret:abc123" {
		t.Fatalf("navigate context = %q, want %q", navMsg.Context, "secret:abc123")
	}
}

func TestLogsNavigation(t *testing.T) {
	v := New(ui.DefaultKeyMap(), color.DefaultStyles())
	v.SetSize(120, 30)

	_, cmd := v.Update(tea.KeyPressMsg{Text: "L", Code: 'L'})
	if cmd == nil {
		t.Fatal("expected navigation command on 'l'")
	}
	msg := cmd()
	navMsg, ok := msg.(view.NavigateMsg)
	if !ok {
		t.Fatalf("msg type = %T, want view.NavigateMsg", msg)
	}
	if navMsg.Target != nav.DebugLogView {
		t.Fatalf("navigate target = %v, want %v", navMsg.Target, nav.DebugLogView)
	}
}
