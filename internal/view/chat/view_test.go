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
