package llm

import (
	"context"
	"fmt"

	"google.golang.org/genai"
)

const defaultGeminiModel = "gemini-2.0-flash"

// GeminiClient implements Client using the Google Gemini API.
type GeminiClient struct {
	client      *genai.Client
	model       string
	temperature float64
	maxTokens   int
}

// GeminiOption configures the Gemini client.
type GeminiOption func(*GeminiClient)

// WithGeminiModel overrides the default model.
func WithGeminiModel(m string) GeminiOption {
	return func(c *GeminiClient) {
		if m != "" {
			c.model = m
		}
	}
}

// WithGeminiTemperature sets the sampling temperature.
func WithGeminiTemperature(t float64) GeminiOption {
	return func(c *GeminiClient) { c.temperature = t }
}

// WithGeminiMaxTokens sets the maximum response tokens.
func WithGeminiMaxTokens(n int) GeminiOption {
	return func(c *GeminiClient) {
		if n > 0 {
			c.maxTokens = n
		}
	}
}

// NewGeminiClient creates a Gemini-backed LLM client.
func NewGeminiClient(ctx context.Context, apiKey string, opts ...GeminiOption) (*GeminiClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required for Gemini provider")
	}
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("creating Gemini client: %w", err)
	}
	g := &GeminiClient{
		client:      client,
		model:       defaultGeminiModel,
		temperature: 0.7,
		maxTokens:   4096,
	}
	for _, opt := range opts {
		opt(g)
	}
	return g, nil
}

// ChatStream implements Client.
func (g *GeminiClient) ChatStream(ctx context.Context, messages []Message) (<-chan StreamEvent, error) {
	config := &genai.GenerateContentConfig{
		Temperature: genai.Ptr(float32(g.temperature)),
	}
	if g.maxTokens > 0 {
		config.MaxOutputTokens = int32(g.maxTokens)
	}

	// Build system instruction, history, and the final user message.
	var history []*genai.Content
	var userParts []*genai.Part

	for _, msg := range messages {
		switch msg.Role {
		case RoleSystem:
			config.SystemInstruction = genai.NewContentFromText(msg.Content, "user")
		case RoleUser:
			if userParts != nil {
				history = append(history, genai.NewContentFromText(userParts[0].Text, "user"))
			}
			userParts = []*genai.Part{genai.NewPartFromText(msg.Content)}
		case RoleAssistant:
			if userParts != nil {
				history = append(history, genai.NewContentFromText(userParts[0].Text, "user"))
				userParts = nil
			}
			history = append(history, genai.NewContentFromText(msg.Content, "model"))
		}
	}

	// The last message must be from the user.
	if userParts == nil {
		return nil, fmt.Errorf("last message must be from the user")
	}

	chat, err := g.client.Chats.Create(ctx, g.model, config, history)
	if err != nil {
		return nil, fmt.Errorf("creating Gemini chat: %w", err)
	}

	stream := chat.SendStream(ctx, userParts...)

	ch := make(chan StreamEvent, 16)
	go func() {
		defer close(ch)
		for resp, err := range stream {
			if err != nil {
				ch <- StreamEvent{Err: fmt.Errorf("gemini stream: %w", err)}
				return
			}
			if text := resp.Text(); text != "" {
				ch <- StreamEvent{Delta: text}
			}
		}
		ch <- StreamEvent{Done: true}
	}()

	return ch, nil
}

// Close implements Client.
func (g *GeminiClient) Close() error {
	return nil
}
