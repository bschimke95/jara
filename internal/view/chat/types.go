package chat

import (
	"context"
	"time"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/llm"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/ui"
)

// chatMode tracks whether the user is typing or scrolling.
type chatMode int

const (
	modeInput chatMode = iota
	modeScroll
)

// chatMessage is a single message in the conversation.
type chatMessage struct {
	Role      llm.Role
	Content   string
	Timestamp time.Time
	Streaming bool // true while the assistant response is being received
}

// chatStreamChunkMsg delivers a token delta from the LLM stream.
type chatStreamChunkMsg struct {
	delta string
	ctx   context.Context
	ch    <-chan llm.StreamEvent
}

// chatStreamDoneMsg signals end of the LLM response stream.
type chatStreamDoneMsg struct{}

// chatStreamErrMsg carries an error from the LLM stream.
type chatStreamErrMsg struct {
	err error
}

// View is the Bubble Tea model for the AI chat view.
type View struct {
	keys      ui.KeyMap
	styles    *color.Styles
	width     int
	height    int
	status    *model.FullStatus
	llmClient llm.Client

	systemPrompt string
	messages     []chatMessage
	mode         chatMode
	streaming    bool

	// inputBuf accumulates typed characters.
	inputBuf string

	// scrollOffset is the number of lines scrolled up from the bottom.
	scrollOffset int

	// streamCancel stops the current LLM stream.
	streamCancel context.CancelFunc
}
