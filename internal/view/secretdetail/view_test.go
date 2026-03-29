package secretdetail

import (
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/ui"
	"github.com/bschimke95/jara/internal/view"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string { return ansiRE.ReplaceAllString(s, "") }

func TestEnterLoadsSecret(t *testing.T) {
	v := New(ui.DefaultKeyMap(), color.DefaultStyles())
	v.SetSize(120, 30)

	now := time.Now()
	status := &model.FullStatus{
		Secrets: []model.Secret{
			{
				URI: "secret:abc123", Label: "db-pass", Owner: "application-pg", Revision: 2, CreateTime: now, UpdateTime: now,
				Revisions: []model.SecretRevision{
					{Revision: 1, CreatedAt: now, Backend: "internal"},
					{Revision: 2, CreatedAt: now, Backend: "internal"},
				},
				Access: []model.SecretAccessInfo{
					{Target: "application-app", Scope: "relation-1", Role: "consume"},
				},
			},
		},
	}
	v.SetStatus(status)

	_, err := v.Enter(view.NavigateContext{Context: "secret:abc123"})
	if err != nil {
		t.Fatalf("Enter() error = %v", err)
	}
	if v.secret == nil {
		t.Fatal("expected secret to be loaded after Enter")
	}
	if v.secret.Label != "db-pass" {
		t.Fatalf("secret label = %q, want %q", v.secret.Label, "db-pass")
	}
}

func TestEnterUnknownSecret(t *testing.T) {
	v := New(ui.DefaultKeyMap(), color.DefaultStyles())
	v.SetSize(120, 30)

	v.SetStatus(&model.FullStatus{
		Secrets: []model.Secret{},
	})

	_, err := v.Enter(view.NavigateContext{Context: "secret:nonexistent"})
	if err != nil {
		t.Fatalf("Enter() error = %v", err)
	}
	if v.secret != nil {
		t.Fatal("expected nil secret for unknown URI")
	}
}

func TestTabSwitchesFocus(t *testing.T) {
	v := New(ui.DefaultKeyMap(), color.DefaultStyles())
	v.SetSize(120, 30)

	if v.focusAccess {
		t.Fatal("expected initial focus on revisions")
	}

	v.Update(tea.KeyPressMsg{Text: "tab", Code: 0x09})
	if !v.focusAccess {
		t.Fatal("expected focus on access after tab")
	}

	v.Update(tea.KeyPressMsg{Text: "tab", Code: 0x09})
	if v.focusAccess {
		t.Fatal("expected focus back on revisions after second tab")
	}
}

func TestDecodeKeyEmitsRequest(t *testing.T) {
	v := New(ui.DefaultKeyMap(), color.DefaultStyles())
	v.SetSize(120, 30)

	now := time.Now()
	v.SetStatus(&model.FullStatus{
		Secrets: []model.Secret{
			{
				URI: "secret:abc123", Label: "test", Revision: 2, CreateTime: now, UpdateTime: now,
				Revisions: []model.SecretRevision{
					{Revision: 1, CreatedAt: now, Backend: "internal"},
					{Revision: 2, CreatedAt: now, Backend: "internal"},
				},
			},
		},
	})
	_, err := v.Enter(view.NavigateContext{Context: "secret:abc123"})
	if err != nil {
		t.Fatalf("Enter() error = %v", err)
	}

	_, cmd := v.Update(tea.KeyPressMsg{Text: "d"})
	if cmd == nil {
		t.Fatal("expected a command from decode key press")
	}

	msg := cmd()
	req, ok := msg.(view.RevealSecretRequestMsg)
	if !ok {
		t.Fatalf("expected RevealSecretRequestMsg, got %T", msg)
	}
	if req.URI != "secret:abc123" {
		t.Fatalf("request URI = %q, want %q", req.URI, "secret:abc123")
	}
	// Cursor starts at row 0 → newest revision = 2.
	if req.Revision != 2 {
		t.Fatalf("request Revision = %d, want 2", req.Revision)
	}
}

func TestRevealResponseOpensModal(t *testing.T) {
	v := New(ui.DefaultKeyMap(), color.DefaultStyles())
	v.SetSize(120, 30)

	now := time.Now()
	v.SetStatus(&model.FullStatus{
		Secrets: []model.Secret{
			{URI: "secret:abc123", Label: "test", CreateTime: now, UpdateTime: now},
		},
	})
	_, _ = v.Enter(view.NavigateContext{Context: "secret:abc123"})

	if v.revealOpen {
		t.Fatal("modal should not be open initially")
	}

	v.Update(view.RevealSecretResponseMsg{
		URI:    "secret:abc123",
		Values: map[string]string{"key": "value"},
	})

	if !v.revealOpen {
		t.Fatal("expected modal to be open after reveal response")
	}
}

func TestRevealModalClosesOnEsc(t *testing.T) {
	v := New(ui.DefaultKeyMap(), color.DefaultStyles())
	v.SetSize(120, 30)

	now := time.Now()
	v.SetStatus(&model.FullStatus{
		Secrets: []model.Secret{
			{URI: "secret:abc123", Label: "test", CreateTime: now, UpdateTime: now},
		},
	})
	_, _ = v.Enter(view.NavigateContext{Context: "secret:abc123"})

	// Open modal.
	v.Update(view.RevealSecretResponseMsg{
		URI:    "secret:abc123",
		Values: map[string]string{"key": "value"},
	})

	if !v.revealOpen {
		t.Fatal("expected modal to be open")
	}

	// Press esc — the modal emits ClosedMsg.
	_, cmd := v.Update(tea.KeyPressMsg{Text: "esc", Code: 0x1b})
	if cmd != nil {
		msg := cmd()
		// Feed the ClosedMsg back.
		v.Update(msg)
	}

	if v.revealOpen {
		t.Fatal("expected modal to be closed after esc")
	}
}

func TestLeaveClosesRevealModal(t *testing.T) {
	v := New(ui.DefaultKeyMap(), color.DefaultStyles())
	v.SetSize(120, 30)

	v.revealOpen = true
	v.Leave()

	if v.revealOpen {
		t.Fatal("expected modal to be closed after Leave")
	}
}

func TestSelectedRevisionReflectsCursor(t *testing.T) {
	v := New(ui.DefaultKeyMap(), color.DefaultStyles())
	v.SetSize(120, 30)

	now := time.Now()
	earlier := now.Add(-1 * time.Hour)
	v.SetStatus(&model.FullStatus{
		Secrets: []model.Secret{
			{
				URI: "secret:abc123", Label: "test", Revision: 3, CreateTime: earlier, UpdateTime: now,
				Revisions: []model.SecretRevision{
					{Revision: 1, CreatedAt: earlier, Backend: "internal"},
					{Revision: 2, CreatedAt: earlier.Add(30 * time.Minute), Backend: "internal"},
					{Revision: 3, CreatedAt: now, Backend: "vault"},
				},
			},
		},
	})
	_, _ = v.Enter(view.NavigateContext{Context: "secret:abc123"})

	// Cursor starts at 0 → newest revision = 3.
	rev := v.selectedRevision()
	if rev == nil {
		t.Fatal("expected non-nil revision")
	}
	if rev.Revision != 3 {
		t.Fatalf("selected revision = %d, want 3", rev.Revision)
	}

	// Capture the rendered view before moving.
	viewBefore := v.View().Content

	// Move cursor down → revision 2.
	v.Update(tea.KeyPressMsg{Text: "j"})
	rev = v.selectedRevision()
	if rev == nil {
		t.Fatal("expected non-nil revision after j")
	}
	if rev.Revision != 2 {
		t.Fatalf("selected revision = %d, want 2", rev.Revision)
	}

	// The rendered view must change to reflect the new revision.
	viewAfter := v.View().Content
	if viewBefore == viewAfter {
		t.Fatal("expected View() output to change after cursor move, but it stayed the same")
	}

	// The metadata should contain "Revision: 2".
	if !strings.Contains(stripANSI(viewAfter), "Revision: 2") {
		t.Fatalf("expected View output to contain 'Revision: 2', got:\n%s", viewAfter)
	}
}

func TestMetadataUpdatesWithSmallTerminal(t *testing.T) {
	v := New(ui.DefaultKeyMap(), color.DefaultStyles())
	v.SetSize(80, 20) // small terminal

	now := time.Now()
	earlier := now.Add(-1 * time.Hour)
	v.SetStatus(&model.FullStatus{
		Secrets: []model.Secret{
			{
				URI: "secret:abc123", Label: "test", Revision: 3, CreateTime: earlier, UpdateTime: now,
				Revisions: []model.SecretRevision{
					{Revision: 1, CreatedAt: earlier, Backend: "internal"},
					{Revision: 2, CreatedAt: earlier.Add(30 * time.Minute), Backend: "vault"},
					{Revision: 3, CreatedAt: now, Backend: "vault"},
				},
			},
		},
	})
	_, _ = v.Enter(view.NavigateContext{Context: "secret:abc123"})

	output1 := v.View().Content
	if !strings.Contains(stripANSI(output1), "Revision: 3") {
		t.Fatalf("expected initial view to contain 'Revision: 3', got:\n%s", output1)
	}

	v.Update(tea.KeyPressMsg{Text: "j"})
	output2 := v.View().Content
	if !strings.Contains(stripANSI(output2), "Revision: 2") {
		t.Fatalf("expected view after j to contain 'Revision: 2', got:\n%s", output2)
	}
}

func TestCursorPreservedAcrossStatusRefresh(t *testing.T) {
	v := New(ui.DefaultKeyMap(), color.DefaultStyles())
	v.SetSize(120, 30)

	now := time.Now()
	earlier := now.Add(-1 * time.Hour)
	status := &model.FullStatus{
		Secrets: []model.Secret{
			{
				URI: "secret:abc123", Label: "test", Revision: 3, CreateTime: earlier, UpdateTime: now,
				Revisions: []model.SecretRevision{
					{Revision: 1, CreatedAt: earlier, Backend: "internal"},
					{Revision: 2, CreatedAt: earlier.Add(30 * time.Minute), Backend: "internal"},
					{Revision: 3, CreatedAt: now, Backend: "vault"},
				},
			},
		},
	}
	v.SetStatus(status)
	_, _ = v.Enter(view.NavigateContext{Context: "secret:abc123"})

	// Move cursor to revision 1 (oldest, two steps down from newest).
	v.Update(tea.KeyPressMsg{Text: "j"})
	v.Update(tea.KeyPressMsg{Text: "j"})
	if rev := v.selectedRevision(); rev == nil || rev.Revision != 1 {
		t.Fatalf("expected revision 1, got %v", rev)
	}

	// Simulate a status stream refresh (same data).
	v.SetStatus(status)

	// Cursor must remain on revision 1 after the refresh.
	rev := v.selectedRevision()
	if rev == nil || rev.Revision != 1 {
		t.Fatalf("expected cursor to stay on revision 1 after SetStatus, got %v", rev)
	}

	// The rendered view must still show revision 1 in the metadata.
	output := v.View().Content
	if !strings.Contains(stripANSI(output), "Revision: 1") {
		t.Fatalf("expected View to contain 'Revision: 1' after refresh, got:\n%s", output)
	}
}
