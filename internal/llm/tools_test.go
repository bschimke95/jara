package llm

import (
	"testing"
)

func TestParseToolCall(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    *ToolCallRequest
	}{
		{
			name:    "valid tool call",
			content: `Some text <tool_call>{"tool":"get_status","args":{"model":"prod"}}</tool_call> more text`,
			want:    &ToolCallRequest{Tool: "get_status", Args: map[string]any{"model": "prod"}},
		},
		{
			name:    "tool call at start",
			content: `<tool_call>{"tool":"list_units","args":{}}</tool_call>`,
			want:    &ToolCallRequest{Tool: "list_units", Args: map[string]any{}},
		},
		{
			name:    "no tool call",
			content: "Hello, this is regular text.",
			want:    nil,
		},
		{
			name:    "incomplete tool call - no close tag",
			content: `<tool_call>{"tool":"get_status","args":{}}`,
			want:    nil,
		},
		{
			name:    "invalid JSON inside tags",
			content: `<tool_call>not json</tool_call>`,
			want:    nil,
		},
		{
			name:    "empty content",
			content: "",
			want:    nil,
		},
		{
			name:    "tool call with whitespace",
			content: `<tool_call>  {"tool":"scale","args":{"app":"nginx","delta":"1"}}  </tool_call>`,
			want:    &ToolCallRequest{Tool: "scale", Args: map[string]any{"app": "nginx", "delta": "1"}},
		},
		{
			name:    "only open tag",
			content: `<tool_call>`,
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseToolCall(tt.content)
			if tt.want == nil {
				if got != nil {
					t.Errorf("ParseToolCall() = %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("ParseToolCall() = nil, want non-nil")
			}
			if got.Tool != tt.want.Tool {
				t.Errorf("Tool = %q, want %q", got.Tool, tt.want.Tool)
			}
		})
	}
}

func TestStripToolCall(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "strip from middle",
			content: "Before\n<tool_call>{\"tool\":\"x\",\"args\":{}}</tool_call>\nAfter",
			want:    "Before\nAfter",
		},
		{
			name:    "strip from start",
			content: "<tool_call>{\"tool\":\"x\",\"args\":{}}</tool_call>\nAfter",
			want:    "After",
		},
		{
			name:    "strip from end",
			content: "Before\n<tool_call>{\"tool\":\"x\",\"args\":{}}</tool_call>",
			want:    "Before",
		},
		{
			name:    "no tool call",
			content: "Just regular text",
			want:    "Just regular text",
		},
		{
			name:    "incomplete tool call - strip to end",
			content: "Before\n<tool_call>partial json...",
			want:    "Before",
		},
		{
			name:    "empty string",
			content: "",
			want:    "",
		},
		{
			name:    "entire string is tool call",
			content: "<tool_call>{\"tool\":\"x\",\"args\":{}}</tool_call>",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripToolCall(tt.content)
			if got != tt.want {
				t.Errorf("StripToolCall() = %q, want %q", got, tt.want)
			}
		})
	}
}
