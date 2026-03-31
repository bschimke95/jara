package llm

import (
	"encoding/json"
	"strings"
)

const (
	toolCallOpenTag  = "<tool_call>"
	toolCallCloseTag = "</tool_call>"
)

// ToolCallRequest is a structured request from the LLM to invoke a tool.
type ToolCallRequest struct {
	Tool string         `json:"tool"`
	Args map[string]any `json:"args"`
}

// ParseToolCall scans content for a complete <tool_call>…</tool_call> block
// and returns the parsed request. Returns nil if no complete block is found or
// parsing fails.
func ParseToolCall(content string) *ToolCallRequest {
	start := strings.Index(content, toolCallOpenTag)
	if start == -1 {
		return nil
	}
	rest := content[start+len(toolCallOpenTag):]
	end := strings.Index(rest, toolCallCloseTag)
	if end == -1 {
		return nil // incomplete — still streaming
	}
	raw := strings.TrimSpace(rest[:end])
	var req ToolCallRequest
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		return nil
	}
	return &req
}

// StripToolCall removes the first complete <tool_call>…</tool_call> block from
// content and returns the cleaned string (suitable for display).
func StripToolCall(content string) string {
	start := strings.Index(content, toolCallOpenTag)
	if start == -1 {
		return content
	}
	rest := content[start+len(toolCallOpenTag):]
	end := strings.Index(rest, toolCallCloseTag)
	if end == -1 {
		// Incomplete tag — strip from <tool_call> to end so streaming doesn't
		// show the raw XML.
		return strings.TrimRight(content[:start], "\n ")
	}
	before := strings.TrimRight(content[:start], "\n ")
	after := strings.TrimLeft(rest[end+len(toolCallCloseTag):], "\n ")
	if before == "" {
		return after
	}
	if after == "" {
		return before
	}
	return before + "\n" + after
}
