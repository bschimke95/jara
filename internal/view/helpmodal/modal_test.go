package helpmodal

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/ui"
)

func newTestModal() Modal {
	m := New(ui.DefaultKeyMap(), color.DefaultStyles())
	m.SetSize(120, 40)
	return m
}

func TestModal_ClosedOnEsc(t *testing.T) {
	m := newTestModal()

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("expected a command on ESC, got nil")
	}
	msg := cmd()
	if _, ok := msg.(ClosedMsg); !ok {
		t.Errorf("expected ClosedMsg, got %T", msg)
	}
	_ = updated
}

func TestModal_ClosedOnHelpKey(t *testing.T) {
	m := newTestModal()

	updated, cmd := m.Update(tea.KeyPressMsg{Code: '?'})
	if cmd == nil {
		t.Fatal("expected a command on '?', got nil")
	}
	msg := cmd()
	if _, ok := msg.(ClosedMsg); !ok {
		t.Errorf("expected ClosedMsg, got %T", msg)
	}
	_ = updated
}

func TestModal_NonKeyMsgIgnored(t *testing.T) {
	m := newTestModal()

	_, cmd := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if cmd != nil {
		t.Errorf("expected nil command for non-key msg, got %v", cmd)
	}
}

func TestModal_OtherKeyNotClosed(t *testing.T) {
	m := newTestModal()

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'j'})
	if cmd != nil {
		t.Errorf("expected nil command for non-close key, got %v", cmd)
	}
}

func TestModal_RenderBoxContainsSections(t *testing.T) {
	m := newTestModal()
	m.SetViewHints([]ui.KeyHint{
		{Key: "enter", Desc: "select"},
		{Key: "d", Desc: "deploy"},
	})

	out := m.renderBox()

	if !strings.Contains(out, "View") {
		t.Error("expected 'View' section in renderBox output")
	}
	if !strings.Contains(out, "General") {
		t.Error("expected 'General' section in renderBox output")
	}
	if !strings.Contains(out, "Key Bindings") {
		t.Error("expected 'Key Bindings' title in renderBox output")
	}
}

func TestModal_RenderBoxNoViewHints(t *testing.T) {
	m := newTestModal()
	m.SetViewHints(nil)

	out := m.renderBox()
	if !strings.Contains(out, "(none)") {
		t.Error("expected '(none)' when no view hints are set")
	}
}

func TestModal_RenderBoxContainsViewHintKeys(t *testing.T) {
	m := newTestModal()
	m.SetViewHints([]ui.KeyHint{
		{Key: "enter", Desc: "select"},
	})

	out := m.renderBox()
	if !strings.Contains(out, "enter") {
		end := 200
		if len(out) < end {
			end = len(out)
		}
		t.Errorf("expected view hint key 'enter' in renderBox output, got: %q", out[:end])
	}
	if !strings.Contains(out, "select") {
		t.Errorf("expected view hint desc 'select' in renderBox output")
	}
}

func TestModal_RenderBoxContainsGeneralHints(t *testing.T) {
	m := newTestModal()
	m.SetViewHints(nil)

	out := m.renderBox()
	// General hints always include quit and help
	if !strings.Contains(out, "quit") {
		t.Error("expected 'quit' in general hints section")
	}
	if !strings.Contains(out, "help") {
		t.Error("expected 'help' in general hints section")
	}
}

func TestRenderSection_Empty(t *testing.T) {
	s := color.DefaultStyles()
	out := renderSection("Test", nil, 60, s)
	if !strings.Contains(out, "Test") {
		t.Error("expected section title in output")
	}
	if !strings.Contains(out, "(none)") {
		t.Error("expected '(none)' for empty hints")
	}
}

func TestRenderSection_WithHints(t *testing.T) {
	s := color.DefaultStyles()
	hints := []ui.KeyHint{
		{Key: "k", Desc: "up"},
		{Key: "j", Desc: "down"},
		{Key: "g", Desc: "top"},
		{Key: "G", Desc: "bottom"},
	}
	out := renderSection("Navigation", hints, 60, s)
	if !strings.Contains(out, "Navigation") {
		t.Error("expected section title in output")
	}
	// Should contain all hint descriptions
	for _, h := range hints {
		if !strings.Contains(out, h.Desc) {
			t.Errorf("expected hint desc %q in output", h.Desc)
		}
	}
}
