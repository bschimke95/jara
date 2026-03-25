package applications

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/ui"
)

func TestUpdateDeployStartsInput(t *testing.T) {
	v := New(ui.DefaultKeyMap(), color.DefaultStyles())
	v.SetSize(120, 30)
	v.SetStatus(&model.FullStatus{Applications: map[string]model.Application{
		"postgresql": {Name: "postgresql"},
	}})

	_, cmd := v.Update(tea.KeyPressMsg{Text: "D", Code: 'D'})
	if cmd == nil {
		t.Fatal("expected focus command when opening modal")
	}
	if !v.deployModalOpen {
		t.Fatal("expected deploy modal to be open")
	}
}
