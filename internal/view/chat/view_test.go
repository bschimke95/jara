package chat

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/llm"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/ui"
	"github.com/bschimke95/jara/internal/view"
)

func newTestView() *View {
	keys := ui.DefaultKeyMap()
	styles := color.DefaultStyles()
	client := llm.NewMockClient(0)
	v := New(keys, styles, client, llm.DefaultSystemPrompt, "")
	v.SetSize(80, 24)
	return v
}

func TestNew_InitialState(t *testing.T) {
	v := newTestView()
	if v.mode != modeInput {
		t.Error("expected initial mode to be modeInput")
	}
	if len(v.messages) != 0 {
		t.Error("expected no initial messages")
	}
	if v.streaming {
		t.Error("expected streaming to be false initially")
	}
}

func TestView_SetStatus(t *testing.T) {
	v := newTestView()
	status := &model.FullStatus{
		Model: model.ModelInfo{Name: "test"},
	}
	v.SetStatus(status)
	if v.status == nil || v.status.Model.Name != "test" {
		t.Error("SetStatus did not store the status")
	}
}

func TestView_Enter_Leave(t *testing.T) {
	v := newTestView()
	cmd, err := v.Enter(view.NavigateContext{})
	if err != nil {
		t.Fatalf("Enter returned error: %v", err)
	}
	if cmd != nil {
		t.Error("Enter should return nil cmd")
	}
	if v.mode != modeInput {
		t.Error("Enter should set mode to modeInput")
	}

	leaveCmd := v.Leave()
	if leaveCmd != nil {
		t.Error("Leave should return nil cmd")
	}
}

func TestView_InputBuffer(t *testing.T) {
	v := newTestView()

	// Type some characters.
	v.Update(tea.KeyPressMsg{Code: 'h', Text: "h"})
	v.Update(tea.KeyPressMsg{Code: 'i', Text: "i"})
	if v.inputBuf != "hi" {
		t.Errorf("expected inputBuf to be 'hi', got %q", v.inputBuf)
	}

	// Backspace removes last char.
	v.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	if v.inputBuf != "h" {
		t.Errorf("expected inputBuf to be 'h' after backspace, got %q", v.inputBuf)
	}
}

func TestView_SendCreatesMessages(t *testing.T) {
	v := newTestView()

	v.inputBuf = "what is broken?"
	updated, cmd := v.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	v = updated.(*View)

	if len(v.messages) != 2 {
		t.Fatalf("expected 2 messages (user + assistant placeholder), got %d", len(v.messages))
	}
	if v.messages[0].Role != llm.RoleUser {
		t.Error("first message should be from user")
	}
	if v.messages[0].Content != "what is broken?" {
		t.Errorf("user message content mismatch: %q", v.messages[0].Content)
	}
	if !v.messages[1].Streaming {
		t.Error("assistant message should be streaming")
	}
	if !v.streaming {
		t.Error("view should be in streaming state")
	}
	if cmd == nil {
		t.Error("send should return a command to start streaming")
	}
	if v.inputBuf != "" {
		t.Error("input buffer should be cleared after send")
	}
}

func TestView_EmptySendIgnored(t *testing.T) {
	v := newTestView()
	v.inputBuf = "   "
	updated, cmd := v.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	v = updated.(*View)

	if len(v.messages) != 0 {
		t.Error("empty send should not create messages")
	}
	if cmd != nil {
		t.Error("empty send should return nil cmd")
	}
}

func TestView_StreamChunk(t *testing.T) {
	v := newTestView()
	v.inputBuf = "test"
	updated, _ := v.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	v = updated.(*View)

	// Simulate a stream chunk.
	ch := make(chan llm.StreamEvent, 1)
	ch <- llm.StreamEvent{Done: true}

	ctx := context.Background()
	updated, _ = v.Update(chatStreamChunkMsg{
		delta: "hello ",
		ctx:   ctx,
		ch:    ch,
	})
	v = updated.(*View)

	if v.messages[1].Content != "hello " {
		t.Errorf("expected assistant content 'hello ', got %q", v.messages[1].Content)
	}
}

func TestView_StreamDone(t *testing.T) {
	v := newTestView()
	v.messages = []chatMessage{
		{Role: llm.RoleUser, Content: "q", Timestamp: time.Now()},
		{Role: llm.RoleAssistant, Content: "answer", Timestamp: time.Now(), Streaming: true},
	}
	v.streaming = true

	updated, _ := v.Update(chatStreamDoneMsg{})
	v = updated.(*View)

	if v.streaming {
		t.Error("streaming should be false after done")
	}
	if v.messages[1].Streaming {
		t.Error("assistant message should not be streaming after done")
	}
}

func TestView_StreamError(t *testing.T) {
	v := newTestView()
	v.messages = []chatMessage{
		{Role: llm.RoleUser, Content: "q", Timestamp: time.Now()},
		{Role: llm.RoleAssistant, Content: "", Timestamp: time.Now(), Streaming: true},
	}
	v.streaming = true

	updated, _ := v.Update(chatStreamErrMsg{err: errTest})
	v = updated.(*View)

	if v.streaming {
		t.Error("streaming should be false after error")
	}
	if v.messages[1].Streaming {
		t.Error("assistant message should not be streaming after error")
	}
	if !containsStr(v.messages[1].Content, "[Error:") {
		t.Error("error should be appended to assistant message")
	}
}

func TestView_NoClient(t *testing.T) {
	keys := ui.DefaultKeyMap()
	styles := color.DefaultStyles()
	v := New(keys, styles, nil, llm.DefaultSystemPrompt, "")
	v.SetSize(80, 24)

	output := v.View()
	if !containsStr(output.Content, "No AI provider configured") {
		t.Error("expected no-client message in view output")
	}
}

func TestView_KeyHints(t *testing.T) {
	v := newTestView()
	hints := v.KeyHints()
	if len(hints) == 0 {
		t.Error("expected at least one key hint")
	}
}

func TestView_OutputExactHeight(t *testing.T) {
	v := newTestView()
	output := v.View()
	lines := strings.Split(output.Content, "\n")
	if len(lines) != 24 {
		t.Errorf("expected exactly 24 lines, got %d", len(lines))
	}
	lastLine := lines[len(lines)-1]
	if !containsStr(lastLine, ">") {
		t.Error("last line should contain the input prompt")
	}
}

// newNoClientView creates a chat view with no AI provider (llmClient == nil).
func newNoClientView() *View {
	keys := ui.DefaultKeyMap()
	styles := color.DefaultStyles()
	v := New(keys, styles, nil, llm.DefaultSystemPrompt, "")
	v.SetSize(80, 24)
	return v
}

// --- Tests for modeInput key handling ---

func TestInputMode_LetterKeysTypeText(t *testing.T) {
	// Keys like j, k, g, G should be typed as text in modeInput,
	// not trigger scrolling (which uses the same key bindings globally).
	v := newTestView()

	letters := []struct {
		code rune
		text string
	}{
		{'j', "j"},
		{'k', "k"},
		{'g', "g"},
		{'G', "G"},
		{':', ":"},
		{'/', "/"},
		{'q', "q"},
		{'?', "?"},
	}

	for _, l := range letters {
		v.inputBuf = ""
		updated, cmd := v.Update(tea.KeyPressMsg{Code: l.code, Text: l.text})
		v = updated.(*View)
		if v.inputBuf != l.text {
			t.Errorf("pressing %q: expected inputBuf=%q, got %q", l.text, l.text, v.inputBuf)
		}
		if cmd == nil {
			t.Errorf("pressing %q: expected non-nil cmd (key consumed by view)", l.text)
		}
	}
}

func TestInputMode_ArrowKeysScroll(t *testing.T) {
	// Arrow keys (not j/k) should still scroll in modeInput.
	v := newTestView()
	// Add messages so there is something to scroll.
	v.messages = make([]chatMessage, 50)
	for i := range v.messages {
		v.messages[i] = chatMessage{Role: llm.RoleUser, Content: "line", Timestamp: time.Now()}
	}

	updated, cmd := v.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	v = updated.(*View)
	if v.scrollOffset == 0 {
		t.Error("up arrow should scroll in modeInput")
	}
	if cmd == nil {
		t.Error("up arrow should return non-nil cmd (consumed)")
	}

	prev := v.scrollOffset
	updated, _ = v.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	v = updated.(*View)
	if v.scrollOffset >= prev {
		t.Error("down arrow should scroll down in modeInput")
	}
}

func TestInputMode_EscGoesBack(t *testing.T) {
	// With an empty input buffer, Esc should emit GoBackMsg.
	v := newTestView()
	v.inputBuf = ""

	_, cmd := v.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("expected non-nil cmd for Esc with empty buffer")
	}
	msg := cmd()
	if _, ok := msg.(view.GoBackMsg); !ok {
		t.Fatalf("expected GoBackMsg, got %T", msg)
	}
}

func TestInputMode_EscClearsBuffer(t *testing.T) {
	// With text in the buffer, Esc should clear it (not navigate back).
	v := newTestView()
	v.inputBuf = "some text"

	updated, cmd := v.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	v = updated.(*View)
	if v.inputBuf != "" {
		t.Errorf("expected inputBuf to be cleared, got %q", v.inputBuf)
	}
	if cmd == nil {
		t.Error("expected non-nil cmd (key consumed)")
	}
	// Should NOT produce GoBackMsg.
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(view.GoBackMsg); ok {
			t.Error("Esc with non-empty buffer should not produce GoBackMsg")
		}
	}
}

// --- Tests for modeScroll key handling ---

func TestScrollMode_GlobalKeysBubbleUp(t *testing.T) {
	// In modeScroll, keys like ':', '/', 'q', '?' should NOT be consumed
	// by the chat view (cmd should be nil) so global handlers can act.
	v := newTestView()
	v.mode = modeScroll

	globalKeys := []struct {
		code rune
		text string
		desc string
	}{
		{':', ":", "command mode"},
		{'/', "/", "filter mode"},
		{'q', "q", "quit"},
		{'?', "?", "help"},
		{'S', "S", "secrets nav"},
		{'M', "M", "machines nav"},
		{'O', "O", "offers nav"},
		{'c', "c", "chat nav"},
	}

	for _, gk := range globalKeys {
		v.mode = modeScroll // Reset mode for each test.
		_, cmd := v.Update(tea.KeyPressMsg{Code: gk.code, Text: gk.text})
		if cmd != nil {
			t.Errorf("pressing %q (%s) in modeScroll: expected nil cmd (bubble up), got non-nil", gk.text, gk.desc)
		}
	}
}

func TestScrollMode_DoesNotCaptureText(t *testing.T) {
	// In modeScroll, printable characters should NOT be appended to inputBuf.
	v := newTestView()
	v.mode = modeScroll
	v.inputBuf = ""

	v.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	if v.inputBuf != "" {
		t.Errorf("expected inputBuf to remain empty in modeScroll, got %q", v.inputBuf)
	}
}

func TestScrollMode_ScrollKeysWork(t *testing.T) {
	// Scroll-specific keys (j, k, g, G) should still work in modeScroll.
	v := newTestView()
	v.mode = modeScroll
	// Add messages so there is content to scroll through.
	v.messages = make([]chatMessage, 50)
	for i := range v.messages {
		v.messages[i] = chatMessage{Role: llm.RoleUser, Content: "line", Timestamp: time.Now()}
	}

	// k (up) should scroll and return non-nil cmd.
	updated, cmd := v.Update(tea.KeyPressMsg{Code: 'k', Text: "k"})
	v = updated.(*View)
	if cmd == nil {
		t.Error("k in modeScroll should return non-nil cmd (consumed)")
	}
	if v.scrollOffset == 0 {
		t.Error("k in modeScroll should scroll up")
	}

	// j (down) should scroll down.
	_, cmd = v.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	if cmd == nil {
		t.Error("j in modeScroll should return non-nil cmd (consumed)")
	}
}

func TestScrollMode_EscGoesToInput(t *testing.T) {
	// Esc in modeScroll should return to modeInput.
	v := newTestView()
	v.mode = modeScroll
	v.scrollOffset = 5

	updated, cmd := v.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	v = updated.(*View)
	if v.mode != modeInput {
		t.Errorf("expected modeInput after Esc in modeScroll, got %d", v.mode)
	}
	if v.scrollOffset != 0 {
		t.Error("scrollOffset should be reset after Esc in modeScroll")
	}
	if cmd == nil {
		t.Error("Esc in modeScroll should return non-nil cmd (consumed)")
	}
}

func TestScrollMode_GGoesToBottom(t *testing.T) {
	// G (shift+g) in modeScroll should go to bottom and switch to modeInput.
	v := newTestView()
	v.mode = modeScroll
	v.scrollOffset = 10

	updated, cmd := v.Update(tea.KeyPressMsg{Code: 'G', Text: "G"})
	v = updated.(*View)
	if v.mode != modeInput {
		t.Errorf("expected modeInput after G in modeScroll, got %d", v.mode)
	}
	if v.scrollOffset != 0 {
		t.Error("scrollOffset should be 0 after G in modeScroll")
	}
	if cmd == nil {
		t.Error("G in modeScroll should return non-nil cmd (consumed)")
	}
}

// --- Tests for no-client mode ---

func TestNoClient_EscNavigatesBack(t *testing.T) {
	// When no AI provider is configured, Esc should emit GoBackMsg.
	v := newNoClientView()

	_, cmd := v.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("expected non-nil cmd for Esc with no client")
	}
	msg := cmd()
	if _, ok := msg.(view.GoBackMsg); !ok {
		t.Fatalf("expected GoBackMsg, got %T", msg)
	}
}

func TestNoClient_GlobalKeysBubbleUp(t *testing.T) {
	// When no AI provider is configured, all non-Esc keys should return
	// nil cmd so global handlers can process them.
	v := newNoClientView()

	keys := []struct {
		code rune
		text string
	}{
		{':', ":"},
		{'/', "/"},
		{'q', "q"},
		{'?', "?"},
		{'j', "j"},
		{'k', "k"},
	}

	for _, k := range keys {
		_, cmd := v.Update(tea.KeyPressMsg{Code: k.code, Text: k.text})
		if cmd != nil {
			t.Errorf("pressing %q with no client: expected nil cmd (bubble up), got non-nil", k.text)
		}
	}
}

func TestNoClient_DoesNotCaptureText(t *testing.T) {
	// When no AI provider is configured, text should not be captured.
	v := newNoClientView()

	v.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	if v.inputBuf != "" {
		t.Errorf("expected inputBuf to remain empty with no client, got %q", v.inputBuf)
	}
}

var errTest = &testError{}

type testError struct{}

func (e *testError) Error() string { return "test error" }

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
