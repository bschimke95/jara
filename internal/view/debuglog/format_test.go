package debuglog

import (
	"testing"
)

func TestHighlightSearchMatch_NoMatch(t *testing.T) {
	line := "some log line"
	got := highlightSearchMatch(line, "xyz")
	if got != line {
		t.Errorf("expected no change for non-matching query, got %q", got)
	}
}

func TestHighlightSearchMatch_Found(t *testing.T) {
	line := "error in module"
	got := highlightSearchMatch(line, "error")
	if got == line {
		t.Error("expected highlighted output to differ from input")
	}
	if len(got) <= len(line) {
		t.Error("highlighted output should be longer (contains ANSI codes)")
	}
}

func TestHighlightSearchMatch_CaseInsensitive(t *testing.T) {
	line := "Error in module"
	got := highlightSearchMatch(line, "error")
	if got == line {
		t.Error("expected case-insensitive match to produce highlighted output")
	}
}
