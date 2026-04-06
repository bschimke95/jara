package appconfig

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/ui"
	"github.com/bschimke95/jara/internal/view"
)

func newTestView() *View {
	keys := ui.DefaultKeyMap()
	styles := color.DefaultStyles()
	v := New(keys, styles)
	v.SetSize(80, 24)
	return v
}

func TestEnter_EmptyContext_ReturnsError(t *testing.T) {
	v := newTestView()
	cmd, err := v.Enter(view.NavigateContext{})
	if err == nil {
		t.Fatal("expected error when entering with empty context")
	}
	if cmd != nil {
		t.Error("expected nil cmd when entering with empty context")
	}
}

func TestEnter_WithContext_SetsAppName(t *testing.T) {
	v := newTestView()
	cmd, err := v.Enter(view.NavigateContext{Context: "postgresql"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.appName != "postgresql" {
		t.Errorf("expected appName=%q, got %q", "postgresql", v.appName)
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd to fetch config")
	}
	msg := cmd()
	fetchMsg, ok := msg.(FetchAppConfigMsg)
	if !ok {
		t.Fatalf("expected FetchAppConfigMsg, got %T", msg)
	}
	if fetchMsg.AppName != "postgresql" {
		t.Errorf("expected FetchAppConfigMsg.AppName=%q, got %q", "postgresql", fetchMsg.AppName)
	}
}

func TestEnter_ClearsPreviousState(t *testing.T) {
	v := newTestView()
	// Simulate previous state.
	v.appName = "old-app"
	v.entries = []model.ConfigEntry{{Key: "k", Value: "v"}}

	_, err := v.Enter(view.NavigateContext{Context: "new-app"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.appName != "new-app" {
		t.Errorf("expected appName=%q, got %q", "new-app", v.appName)
	}
	if v.entries != nil {
		t.Error("expected entries to be cleared on Enter")
	}
}

func TestUpdate_AppConfigMsg_SetsEntries(t *testing.T) {
	v := newTestView()
	v.appName = "myapp"

	entries := []model.ConfigEntry{
		{Key: "port", Value: "8080", Default: "80", Source: "user", Type: "int"},
		{Key: "debug", Value: "true", Default: "false", Source: "user", Type: "bool"},
	}

	updated, _ := v.Update(AppConfigMsg{AppName: "myapp", Entries: entries})
	v = updated.(*View)

	if len(v.entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(v.entries))
	}
	if v.entries[0].Key != "port" {
		t.Errorf("expected first entry key=%q, got %q", "port", v.entries[0].Key)
	}
}

func TestUpdate_AppConfigMsg_WrongApp_Ignored(t *testing.T) {
	v := newTestView()
	v.appName = "myapp"

	updated, _ := v.Update(AppConfigMsg{
		AppName: "other-app",
		Entries: []model.ConfigEntry{{Key: "k"}},
	})
	v = updated.(*View)

	if len(v.entries) != 0 {
		t.Error("entries from a different app should be ignored")
	}
}

func TestView_NoAppSelected(t *testing.T) {
	v := newTestView()
	v.appName = ""
	output := v.View()
	if output.Content != "No application selected" {
		t.Errorf("expected 'No application selected', got %q", output.Content)
	}
}

func TestView_Loading(t *testing.T) {
	v := newTestView()
	v.appName = "myapp"
	v.entries = nil
	output := v.View()
	expected := "Loading config for myapp..."
	if output.Content != expected {
		t.Errorf("expected %q, got %q", expected, output.Content)
	}
}

func TestView_ShowsTable(t *testing.T) {
	v := newTestView()
	v.appName = "myapp"
	v.entries = []model.ConfigEntry{
		{Key: "port", Value: "8080", Default: "80", Source: "user", Type: "int"},
	}
	v.table.SetRows(rows(v.entries))
	output := v.View()
	if output.Content == "No application selected" || output.Content == "Loading config for myapp..." {
		t.Error("expected table output, got placeholder text")
	}
}

func TestKeyHints(t *testing.T) {
	v := newTestView()
	hints := v.KeyHints()
	if len(hints) == 0 {
		t.Error("expected at least one key hint")
	}
}

func TestLeave(t *testing.T) {
	v := newTestView()
	cmd := v.Leave()
	if cmd != nil {
		t.Error("Leave should return nil cmd")
	}
}

// Test that the C key in the applications view navigates with context.
// This is an integration-style test using the applications view directly.
func TestApplicationsView_ConfigNavPassesContext(t *testing.T) {
	keys := ui.DefaultKeyMap()
	styles := color.DefaultStyles()

	// We can't import applications here (circular), so we test the
	// appconfig side: verify Enter rejects empty and accepts non-empty.
	v := New(keys, styles)
	v.SetSize(80, 24)

	// Simulate command-bar navigation (empty context).
	_, err := v.Enter(view.NavigateContext{Context: ""})
	if err == nil {
		t.Error("expected error for empty context (command-bar path)")
	}

	// Simulate C-key navigation (with context).
	_, err = v.Enter(view.NavigateContext{Context: "postgresql"})
	if err != nil {
		t.Errorf("unexpected error for non-empty context: %v", err)
	}
}

// Test that non-key messages don't crash or change state.
func TestUpdate_NonKeyMsg_Ignored(t *testing.T) {
	v := newTestView()
	v.appName = "myapp"

	// A random message type should be handled gracefully.
	updated, _ := v.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	v = updated.(*View)

	if v.appName != "myapp" {
		t.Error("appName should not change on unrelated message")
	}
}
