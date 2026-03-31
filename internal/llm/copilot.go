package llm

import (
	"context"
	"fmt"
	"os/exec"
	"sync"

	copilot "github.com/github/copilot-sdk/go"
)

const copilotDefaultModel = "gpt-4o"

// CopilotClient implements Client using the official GitHub Copilot SDK.
// It communicates with the Copilot CLI via JSON-RPC, with the SDK managing
// the CLI process lifecycle automatically.
//
// The Copilot CLI binary must be installed separately and available in PATH.
// NewCopilotClient returns an error if the binary cannot be found.
type CopilotClient struct {
	sdkClient   *copilot.Client
	model       string
	githubToken string

	mu      sync.Mutex
	started bool
}

// CopilotOption configures the Copilot client.
type CopilotOption func(*CopilotClient)

// WithCopilotModel overrides the default model.
func WithCopilotModel(m string) CopilotOption {
	return func(c *CopilotClient) {
		if m != "" {
			c.model = m
		}
	}
}

// WithCopilotGitHubToken sets an explicit GitHub token for authentication.
// When provided, this takes priority over environment variables and gh CLI.
func WithCopilotGitHubToken(token string) CopilotOption {
	return func(c *CopilotClient) {
		c.githubToken = token
	}
}

// NewCopilotClient creates a Copilot-backed LLM client using the official SDK.
// The Copilot CLI binary must be installed on the system and available in PATH.
// Authentication is handled via environment variables (GITHUB_TOKEN, GH_TOKEN,
// COPILOT_GITHUB_TOKEN) or gh CLI credentials, or an explicit token provided
// via WithCopilotGitHubToken.
//
// Returns an error if the "copilot" binary is not found in PATH.
func NewCopilotClient(opts ...CopilotOption) (*CopilotClient, error) {
	cliPath, err := exec.LookPath("copilot")
	if err != nil {
		return nil, fmt.Errorf(
			"GitHub Copilot CLI not found in PATH.\n" +
				"Install it from: https://github.com/github/copilot-sdk/releases\n" +
				"Ensure the 'copilot' binary is accessible before using the Copilot provider.",
		)
	}

	c := &CopilotClient{
		model: copilotDefaultModel,
	}
	for _, opt := range opts {
		opt(c)
	}

	clientOpts := &copilot.ClientOptions{
		LogLevel: "error",
		CLIPath:  cliPath,
	}
	if c.githubToken != "" {
		clientOpts.GitHubToken = c.githubToken
	}
	c.sdkClient = copilot.NewClient(clientOpts)

	return c, nil
}

// ensureStarted lazily starts the Copilot CLI server on first use.
func (c *CopilotClient) ensureStarted(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.started {
		return nil
	}
	if err := c.sdkClient.Start(ctx); err != nil {
		return fmt.Errorf("starting copilot client: %w", err)
	}
	c.started = true
	return nil
}

// ChatStream implements Client.
func (c *CopilotClient) ChatStream(ctx context.Context, messages []Message) (<-chan StreamEvent, error) {
	if err := c.ensureStarted(ctx); err != nil {
		return nil, err
	}

	// Separate system prompt from conversation history.
	var systemPrompt string
	var lastUserMsg string
	var history []Message
	for i, m := range messages {
		switch m.Role {
		case RoleSystem:
			systemPrompt = m.Content
		case RoleUser:
			lastUserMsg = m.Content
			// All but the last user message are history.
			if i < len(messages)-1 {
				history = append(history, m)
			}
		case RoleAssistant:
			history = append(history, m)
		}
	}

	if lastUserMsg == "" {
		ch := make(chan StreamEvent, 1)
		ch <- StreamEvent{Done: true}
		close(ch)
		return ch, nil
	}

	// Build system content: prompt + conversation history as context.
	systemContent := systemPrompt
	if len(history) > 0 {
		systemContent += "\n\n--- Conversation History ---\n"
		for _, m := range history {
			systemContent += fmt.Sprintf("[%s]: %s\n", m.Role, m.Content)
		}
	}

	session, err := c.sdkClient.CreateSession(ctx, &copilot.SessionConfig{
		Model:     c.model,
		Streaming: true,
		SystemMessage: &copilot.SystemMessageConfig{
			Mode:    "replace",
			Content: systemContent,
		},
		OnPermissionRequest: copilot.PermissionHandler.ApproveAll,
		InfiniteSessions: &copilot.InfiniteSessionConfig{
			Enabled: copilot.Bool(false),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("creating copilot session: %w", err)
	}

	ch := make(chan StreamEvent, 64)
	var once sync.Once
	sessionDone := make(chan struct{})

	endSession := func(ev StreamEvent) {
		once.Do(func() {
			close(sessionDone)
			select {
			case ch <- ev:
			default:
			}
			close(ch)
			go func() { _ = session.Disconnect() }()
		})
	}

	// Disconnect on context cancellation so the caller is never blocked.
	go func() {
		select {
		case <-ctx.Done():
			endSession(StreamEvent{})
		case <-sessionDone:
			// Session ended normally; this goroutine exits cleanly.
		}
	}()

	session.On(func(event copilot.SessionEvent) {
		switch event.Type {
		case "assistant.message_delta":
			if event.Data.DeltaContent != nil {
				select {
				case ch <- StreamEvent{Delta: *event.Data.DeltaContent}:
				case <-ctx.Done():
				}
			}
		case "session.idle":
			endSession(StreamEvent{Done: true})
		case "session.error":
			errMsg := "copilot session error"
			if event.Data.Content != nil {
				errMsg = *event.Data.Content
			}
			endSession(StreamEvent{Err: fmt.Errorf("%s", errMsg)})
		}
	})

	_, err = session.Send(ctx, copilot.MessageOptions{
		Prompt: lastUserMsg,
	})
	if err != nil {
		endSession(StreamEvent{Err: fmt.Errorf("sending message: %w", err)})
	}

	return ch, nil
}

// Close implements Client.
func (c *CopilotClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.started {
		return nil
	}
	c.started = false
	return c.sdkClient.Stop()
}
