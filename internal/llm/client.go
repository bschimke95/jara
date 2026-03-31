// Package llm provides an abstraction layer for interacting with large language
// models. It defines a provider-agnostic Client interface and concrete
// implementations for GitHub Copilot and Google Gemini.
package llm

import "context"

// Role identifies the sender of a chat message.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Message is a single entry in a chat conversation.
type Message struct {
	Role    Role
	Content string
}

// StreamEvent carries a single delta from a streaming chat completion.
type StreamEvent struct {
	// Delta contains the new token(s) in this chunk.
	Delta string
	// Done is true when the stream has finished.
	Done bool
	// Err is non-nil when the stream encountered an error.
	Err error
}

// Client is the provider-agnostic interface for LLM chat completions.
type Client interface {
	// ChatStream sends the conversation and returns a channel of streaming
	// events. The caller must read from the channel until it is closed or
	// a StreamEvent with Done==true or Err!=nil is received.
	ChatStream(ctx context.Context, messages []Message) (<-chan StreamEvent, error)

	// Close releases any resources held by the client.
	Close() error
}
