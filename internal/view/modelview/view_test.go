package modelview

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/nav"
	"github.com/bschimke95/jara/internal/ui"
	"github.com/bschimke95/jara/internal/view"
)

func TestUpdateDeployStartsInput(t *testing.T) {
	v := New(ui.DefaultKeyMap(), color.DefaultStyles(), func(string) error { return nil })
	v.SetSize(120, 30)

	_, cmd := v.Update(tea.KeyPressMsg{Text: "D", Code: 'D'})
	if cmd == nil {
		t.Fatal("expected focus command when opening modal")
	}
	if !v.deployModalOpen {
		t.Fatal("expected deploy modal to be open")
	}
}

func TestUpdateApplicationsNavShortcut(t *testing.T) {
	v := New(ui.DefaultKeyMap(), color.DefaultStyles(), func(string) error { return nil })
	v.SetSize(120, 30)

	_, cmd := v.Update(tea.KeyPressMsg{Text: "A", Code: 'A'})
	if cmd == nil {
		t.Fatal("expected navigation command for applications shortcut")
	}
	msg := cmd()
	navigate, ok := msg.(view.NavigateMsg)
	if !ok {
		t.Fatalf("msg type = %T, want view.NavigateMsg", msg)
	}
	if navigate.Target != nav.ApplicationsView {
		t.Fatalf("navigate target = %v, want %v", navigate.Target, nav.ApplicationsView)
	}
}
