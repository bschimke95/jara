package modelview

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/bschimke95/jara/internal/ui"
)

func TestUpdateDeployStartsInput(t *testing.T) {
	v := New(ui.DefaultKeyMap())
	v.SetSize(120, 30)

	_, cmd := v.Update(tea.KeyPressMsg{Text: "D", Code: 'D'})
	if cmd == nil {
		t.Fatal("expected focus command when opening modal")
	}
	if !v.deployModalOpen {
		t.Fatal("expected deploy modal to be open")
	}
}
