// Package chat implements the AI-powered cluster analysis chat view.
// It provides a full-screen conversational interface where users can ask
// questions about their Juju cluster and receive streaming LLM responses.
package chat

import (
	"context"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/llm"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/ui"
	"github.com/bschimke95/jara/internal/view"
)

// New creates a new chat view.
func New(keys ui.KeyMap, styles *color.Styles, client llm.Client, systemPrompt, initErr string) *View {
	return &View{
		keys:         keys,
		styles:       styles,
		llmClient:    client,
		systemPrompt: systemPrompt,
		initErr:      initErr,
		messages:     make([]chatMessage, 0),
		mode:         modeInput,
	}
}

func (v *View) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// SetStatus implements view.StatusReceiver.
func (v *View) SetStatus(status *model.FullStatus) {
	v.status = status
}

// KeyHints returns the view-specific key hints for the header.
func (v *View) KeyHints() []view.KeyHint {
	bk := func(b key.Binding) string { return b.Help().Key }
	return []view.KeyHint{
		{Key: "enter", Desc: "send"},
		{Key: bk(v.keys.Up) + "/" + bk(v.keys.Down), Desc: "scroll"},
		{Key: bk(v.keys.Back), Desc: "back"},
	}
}

// Enter is called when the chat view becomes active.
func (v *View) Enter(_ view.NavigateContext) (tea.Cmd, error) {
	v.mode = modeInput
	return nil, nil
}

// Leave is called when navigating away from the chat view.
func (v *View) Leave() tea.Cmd {
	if v.streamCancel != nil {
		v.streamCancel()
		v.streamCancel = nil
	}
	v.streaming = false
	// Mark any streaming message as complete.
	if len(v.messages) > 0 && v.messages[len(v.messages)-1].Streaming {
		v.messages[len(v.messages)-1].Streaming = false
	}
	return nil
}

func (v *View) Init() tea.Cmd { return nil }

func (v *View) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case chatStreamChunkMsg:
		if msg.ctx.Err() != nil {
			return v, nil
		}
		if len(v.messages) > 0 {
			last := &v.messages[len(v.messages)-1]
			if last.Streaming {
				last.Content += msg.delta
			}
		}
		// Auto-scroll to bottom on new content.
		v.scrollOffset = 0
		return v, readNextStreamEvent(msg.ctx, msg.ch)

	case chatStreamDoneMsg:
		v.streaming = false
		if len(v.messages) > 0 {
			v.messages[len(v.messages)-1].Streaming = false
		}
		v.streamCancel = nil
		return v, nil

	case chatStreamErrMsg:
		v.streaming = false
		if len(v.messages) > 0 {
			last := &v.messages[len(v.messages)-1]
			if last.Streaming {
				last.Content += "\n\n[Error: " + msg.err.Error() + "]"
				last.Streaming = false
			}
		}
		v.streamCancel = nil
		return v, nil
	}

	kp, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return v, nil
	}

	// Handle key presses based on mode.
	switch {
	case key.Matches(kp, v.keys.Back):
		if v.mode == modeScroll {
			v.mode = modeInput
			v.scrollOffset = 0
			return v, noopCmd
		}
		// In input mode with empty buffer, navigate back.
		if v.inputBuf == "" {
			return v, func() tea.Msg { return view.GoBackMsg{} }
		}
		// Clear input buffer.
		v.inputBuf = ""
		return v, noopCmd

	case kp.String() == "enter":
		return v.handleSend()

	case kp.String() == "backspace":
		if len(v.inputBuf) > 0 {
			v.inputBuf = v.inputBuf[:len(v.inputBuf)-1]
		}
		return v, noopCmd

	case key.Matches(kp, v.keys.Up):
		return v.handleScroll(1)

	case key.Matches(kp, v.keys.Down):
		return v.handleScroll(-1)

	case key.Matches(kp, v.keys.PageUp):
		return v.handleScroll(v.messageAreaHeight() / 2)

	case key.Matches(kp, v.keys.PageDown):
		return v.handleScroll(-v.messageAreaHeight() / 2)

	case key.Matches(kp, v.keys.Top):
		v.scrollOffset = v.maxScroll()
		v.mode = modeScroll
		return v, noopCmd

	case key.Matches(kp, v.keys.Bottom):
		v.scrollOffset = 0
		v.mode = modeInput
		return v, noopCmd

	default:
		// Append printable characters to input buffer.
		if kp.Text != "" {
			v.inputBuf += kp.Text
			return v, noopCmd
		}
	}

	return v, nil
}

func (v *View) handleSend() (*View, tea.Cmd) {
	text := strings.TrimSpace(v.inputBuf)
	if text == "" || v.streaming {
		return v, nil
	}

	if v.llmClient == nil {
		return v, nil
	}

	v.inputBuf = ""
	v.scrollOffset = 0

	// Append user message.
	v.messages = append(v.messages, chatMessage{
		Role:      llm.RoleUser,
		Content:   text,
		Timestamp: time.Now(),
	})

	// Append placeholder for assistant response.
	v.messages = append(v.messages, chatMessage{
		Role:      llm.RoleAssistant,
		Timestamp: time.Now(),
		Streaming: true,
	})

	v.streaming = true

	// Build the LLM payload.
	llmMessages := v.buildLLMMessages()

	client := v.llmClient
	ctx, cancel := context.WithCancel(context.Background())
	v.streamCancel = cancel

	return v, func() tea.Msg {
		ch, err := client.ChatStream(ctx, llmMessages)
		if err != nil {
			return chatStreamErrMsg{err: err}
		}
		// Read the first event to kick off the streaming.
		return readStreamEvent(ctx, ch)
	}
}

func (v *View) buildLLMMessages() []llm.Message {
	var msgs []llm.Message

	// System prompt with status context.
	systemContent := v.systemPrompt
	if v.status != nil {
		statusCtx := llm.FormatStatusContext(v.status)
		systemContent += "\n\n--- Current Cluster Status ---\n" + statusCtx
	}
	msgs = append(msgs, llm.Message{Role: llm.RoleSystem, Content: systemContent})

	// Conversation history.
	for _, m := range v.messages {
		if m.Role == llm.RoleUser || (m.Role == llm.RoleAssistant && !m.Streaming) {
			msgs = append(msgs, llm.Message{Role: m.Role, Content: m.Content})
		}
	}

	return msgs
}

func (v *View) handleScroll(delta int) (*View, tea.Cmd) {
	v.scrollOffset += delta
	maxOffset := v.maxScroll()
	if v.scrollOffset > maxOffset {
		v.scrollOffset = maxOffset
	}
	if v.scrollOffset < 0 {
		v.scrollOffset = 0
	}
	if v.scrollOffset > 0 {
		v.mode = modeScroll
	}
	return v, noopCmd
}

func (v *View) maxScroll() int {
	rendered := v.renderMessages()
	lines := strings.Split(rendered, "\n")
	totalLines := len(lines)
	viewable := v.messageAreaHeight()
	if totalLines > viewable {
		return totalLines - viewable
	}
	return 0
}

// noopCmd is a non-nil command that signals the key was consumed by this
// view, preventing the global key handler from also acting on it.
var noopCmd = func() tea.Msg { return nil }

func (v *View) messageAreaHeight() int {
	// The last line is the input prompt; everything above is for messages.
	h := v.height - 1
	if h < 1 {
		h = 1
	}
	return h
}

func (v *View) View() tea.View {
	if v.llmClient == nil {
		return tea.NewView(v.renderNoClient())
	}

	h := v.height
	if h < 2 {
		h = 2
	}

	// Build exactly h lines: [0..h-2] messages, [h-1] input prompt.
	output := make([]string, h)

	// Fill message area.
	msgLines := strings.Split(v.renderMessages(), "\n")
	msgH := h - 1

	// Apply scroll offset.
	if len(msgLines) > msgH {
		end := len(msgLines) - v.scrollOffset
		if end < msgH {
			end = msgH
		}
		start := end - msgH
		if start < 0 {
			start = 0
			end = msgH
		}
		if end > len(msgLines) {
			end = len(msgLines)
		}
		msgLines = msgLines[start:end]
	}

	// Place message lines bottom-aligned in the message area.
	offset := msgH - len(msgLines)
	for i, line := range msgLines {
		output[offset+i] = line
	}

	// Input prompt is always the last line.
	output[h-1] = v.renderInput()

	return tea.NewView(strings.Join(output, "\n"))
}

func (v *View) renderNoClient() string {
	noClient := lipgloss.NewStyle().Foreground(v.styles.Muted)
	title := lipgloss.NewStyle().Foreground(v.styles.Primary).Bold(true)
	errStyle := lipgloss.NewStyle().Foreground(v.styles.ErrorColor)

	var b strings.Builder
	b.WriteString(title.Render("AI Chat"))
	b.WriteString("\n\n")

	// When initialisation failed (e.g. Copilot CLI not found), show the
	// specific error so the user has an actionable message.
	if v.initErr != "" {
		b.WriteString(errStyle.Render("AI provider error:"))
		b.WriteString("\n")
		for _, line := range strings.Split(v.initErr, "\n") {
			b.WriteString(errStyle.Render("  " + line))
			b.WriteString("\n")
		}
		return b.String()
	}

	b.WriteString(noClient.Render("No AI provider configured."))
	b.WriteString("\n\n")
	b.WriteString(noClient.Render("To enable AI analysis, set one of the following:"))
	b.WriteString("\n\n")
	b.WriteString(noClient.Render("  GitHub Copilot:"))
	b.WriteString("\n")
	b.WriteString(noClient.Render("    export GITHUB_TOKEN=<your-token>"))
	b.WriteString("\n\n")
	b.WriteString(noClient.Render("  Google Gemini:"))
	b.WriteString("\n")
	b.WriteString(noClient.Render("    export GOOGLE_AI_API_KEY=<your-key>"))
	b.WriteString("\n\n")
	b.WriteString(noClient.Render("  Or configure in ~/.config/jara/config.yaml:"))
	b.WriteString("\n")
	b.WriteString(noClient.Render("    jara:"))
	b.WriteString("\n")
	b.WriteString(noClient.Render("      ai:"))
	b.WriteString("\n")
	b.WriteString(noClient.Render("        provider: copilot  # or gemini"))
	return b.String()
}

func (v *View) renderInput() string {
	promptStyle := lipgloss.NewStyle().Foreground(v.styles.Primary).Bold(true)
	textStyle := lipgloss.NewStyle().Foreground(v.styles.Title)
	cursorStyle := lipgloss.NewStyle().Foreground(v.styles.Primary)

	prompt := promptStyle.Render("> ")

	if v.streaming {
		mutedStyle := lipgloss.NewStyle().Foreground(v.styles.Muted)
		return prompt + mutedStyle.Render("receiving response...")
	}
	return prompt + textStyle.Render(v.inputBuf) + cursorStyle.Render("\u2588")
}

func (v *View) renderMessages() string {
	if len(v.messages) == 0 {
		welcome := lipgloss.NewStyle().Foreground(v.styles.Muted)
		return welcome.Render("Ask a question about your cluster. Type and press Enter to send.")
	}

	userStyle := lipgloss.NewStyle().Foreground(v.styles.Primary).Bold(true)
	assistantLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("#98c379")).Bold(true)
	contentStyle := lipgloss.NewStyle().Foreground(v.styles.Title)
	mutedStyle := lipgloss.NewStyle().Foreground(v.styles.Muted)

	maxWidth := v.width - 2
	if maxWidth < 20 {
		maxWidth = 20
	}

	var parts []string
	for _, msg := range v.messages {
		switch msg.Role {
		case llm.RoleUser:
			label := userStyle.Render("You:")
			content := contentStyle.Render(wrapText(msg.Content, maxWidth))
			parts = append(parts, label+"\n"+content)

		case llm.RoleAssistant:
			label := assistantLabel.Render("AI:")
			content := msg.Content
			if msg.Streaming {
				content += "▌"
			}
			rendered := contentStyle.Render(wrapText(content, maxWidth))
			parts = append(parts, label+"\n"+rendered)

		case llm.RoleSystem:
			parts = append(parts, mutedStyle.Render("[system context loaded]"))
		}
	}

	return strings.Join(parts, "\n\n")
}

// wrapText wraps long lines to the given width.
func wrapText(s string, width int) string {
	if width <= 0 {
		return s
	}
	var result strings.Builder
	for _, line := range strings.Split(s, "\n") {
		if lipgloss.Width(line) <= width {
			if result.Len() > 0 {
				result.WriteString("\n")
			}
			result.WriteString(line)
			continue
		}
		// Word-wrap long lines.
		words := strings.Fields(line)
		currentLine := ""
		for _, word := range words {
			if currentLine == "" {
				currentLine = word
				continue
			}
			candidate := currentLine + " " + word
			if lipgloss.Width(candidate) > width {
				if result.Len() > 0 {
					result.WriteString("\n")
				}
				result.WriteString(currentLine)
				currentLine = word
			} else {
				currentLine = candidate
			}
		}
		if currentLine != "" {
			if result.Len() > 0 {
				result.WriteString("\n")
			}
			result.WriteString(currentLine)
		}
	}
	return result.String()
}

// readNextStreamEvent returns a Cmd that reads the next event from the stream.
func readNextStreamEvent(ctx context.Context, ch <-chan llm.StreamEvent) tea.Cmd {
	return func() tea.Msg {
		return readStreamEvent(ctx, ch)
	}
}

// readStreamEvent blocks until the next stream event is available.
func readStreamEvent(ctx context.Context, ch <-chan llm.StreamEvent) tea.Msg {
	select {
	case <-ctx.Done():
		return chatStreamDoneMsg{}
	case ev, ok := <-ch:
		if !ok {
			return chatStreamDoneMsg{}
		}
		if ev.Err != nil {
			return chatStreamErrMsg{err: ev.Err}
		}
		if ev.Done {
			return chatStreamDoneMsg{}
		}
		return chatStreamChunkMsg{delta: ev.Delta, ctx: ctx, ch: ch}
	}
}
