package view

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
)

func TestBindingKey(t *testing.T) {
	tests := []struct {
		name string
		b    key.Binding
		want string
	}{
		{
			name: "simple key",
			b:    key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
			want: "q",
		},
		{
			name: "multi-key uses first help text",
			b:    key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/up", "up")),
			want: "k/up",
		},
		{
			name: "special key",
			b:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
			want: "enter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BindingKey(tt.b)
			if got != tt.want {
				t.Errorf("BindingKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPadToHeight(t *testing.T) {
	tests := []struct {
		name    string
		content string
		height  int
		want    int // expected number of lines
	}{
		{
			name:    "pad empty to 5",
			content: "",
			height:  5,
			want:    5,
		},
		{
			name:    "content already exact",
			content: "a\nb\nc",
			height:  3,
			want:    3,
		},
		{
			name:    "content shorter than height",
			content: "line1\nline2",
			height:  5,
			want:    5,
		},
		{
			name:    "content taller than height - truncate",
			content: "a\nb\nc\nd\ne",
			height:  3,
			want:    3,
		},
		{
			name:    "height zero",
			content: "a\nb",
			height:  0,
			want:    0,
		},
		{
			name:    "single line padded",
			content: "only one line",
			height:  3,
			want:    3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PadToHeight(tt.content, tt.height)

			var gotLines int
			if tt.height == 0 {
				// PadToHeight with height=0 returns "" (truncated to 0 lines)
				if got != "" {
					t.Errorf("PadToHeight() = %q, want empty", got)
				}
				return
			}

			gotLines = len(strings.Split(got, "\n"))
			if gotLines != tt.want {
				t.Errorf("PadToHeight() produced %d lines, want %d", gotLines, tt.want)
			}
		})
	}
}

func TestPadToHeight_preservesContent(t *testing.T) {
	content := "hello\nworld"
	got := PadToHeight(content, 5)
	lines := strings.Split(got, "\n")

	if lines[0] != "hello" {
		t.Errorf("first line = %q, want %q", lines[0], "hello")
	}
	if lines[1] != "world" {
		t.Errorf("second line = %q, want %q", lines[1], "world")
	}
	// Padded lines should be empty.
	for i := 2; i < 5; i++ {
		if lines[i] != "" {
			t.Errorf("padded line[%d] = %q, want empty", i, lines[i])
		}
	}
}

func TestPadToHeight_truncatesContent(t *testing.T) {
	content := "a\nb\nc\nd\ne"
	got := PadToHeight(content, 2)
	lines := strings.Split(got, "\n")

	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2", len(lines))
	}
	if lines[0] != "a" {
		t.Errorf("line[0] = %q, want %q", lines[0], "a")
	}
	if lines[1] != "b" {
		t.Errorf("line[1] = %q, want %q", lines[1], "b")
	}
}
