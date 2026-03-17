package debuglog

import (
	"strings"
	"testing"

	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/ui"
)

// ── truncateLabel ─────────────────────────────────────────────────────────────

func TestTruncateLabel(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"empty string", "", 10, ""},
		{"under limit", "hello", 10, "hello"},
		{"exactly at limit", "hello", 5, "hello"},
		{"over limit", "hello world", 8, "hello w\u2026"},
		{"maxLen=1", "hello", 1, "\u2026"},
		{"unicode runes", "h\u00e9llo", 4, "h\u00e9l\u2026"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateLabel(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateLabel(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

// ── appendUnique ─────────────────────────────────────────────────────────────

func TestAppendUnique(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		v     string
		want  []string
	}{
		{"add to empty", nil, "a", []string{"a"}},
		{"add new element", []string{"a", "b"}, "c", []string{"a", "b", "c"}},
		{"duplicate is ignored", []string{"a", "b"}, "a", []string{"a", "b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := appendUnique(tt.slice, tt.v)
			if len(got) != len(tt.want) {
				t.Fatalf("appendUnique(%v, %q) = %v, want %v", tt.slice, tt.v, got, tt.want)
			}
			for i, g := range got {
				if g != tt.want[i] {
					t.Errorf("appendUnique: got[%d] = %q, want %q", i, g, tt.want[i])
				}
			}
		})
	}
}

// ── removeFromSlice ───────────────────────────────────────────────────────────

func TestRemoveFromSlice(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		v     string
		want  []string
	}{
		{"remove from empty", nil, "a", nil},
		{"remove present element", []string{"a", "b", "c"}, "b", []string{"a", "c"}},
		{"remove absent element", []string{"a", "b"}, "z", []string{"a", "b"}},
		{"remove first", []string{"a", "b", "c"}, "a", []string{"b", "c"}},
		{"remove last", []string{"a", "b", "c"}, "c", []string{"a", "b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// keep a copy to verify no aliasing
			orig := make([]string, len(tt.slice))
			copy(orig, tt.slice)
			got := removeFromSlice(tt.slice, tt.v)
			for i, v := range tt.slice {
				if v != orig[i] {
					t.Errorf("removeFromSlice mutated original slice at index %d: got %q, want %q", i, v, orig[i])
				}
			}
			if len(got) != len(tt.want) {
				t.Fatalf("removeFromSlice(%v, %q) = %v, want %v", tt.slice, tt.v, got, tt.want)
			}
			for i, g := range got {
				if g != tt.want[i] {
					t.Errorf("removeFromSlice: got[%d] = %q, want %q", i, g, tt.want[i])
				}
			}
		})
	}
}

// ── addLabelEntry ─────────────────────────────────────────────────────────────

func TestAddLabelEntry(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantKey    string
		wantVal    string
		wantAbsent bool
	}{
		{"valid key=value", "env=prod", "env", "prod", false},
		{"value with equals sign", "msg=a=b", "msg", "a=b", false},
		{"key only (no =)", "barekey", "barekey", "", false},
		{"empty string (no-op)", "", "", "", true},
		{"whitespace key (no-op)", "  ", "", "", true},
		{"key with spaces trimmed", " env = prod ", "env", "prod", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &FilterModal{leftCursor: leftPaneLabels}
			m.addLabelEntry(tt.input)
			if tt.wantAbsent {
				if len(m.filter.IncludeLabels) != 0 {
					t.Errorf("addLabelEntry(%q): expected no labels, got %v", tt.input, m.filter.IncludeLabels)
				}
				return
			}
			v, ok := m.filter.IncludeLabels[tt.wantKey]
			if !ok {
				t.Errorf("addLabelEntry(%q): key %q not found in %v", tt.input, tt.wantKey, m.filter.IncludeLabels)
				return
			}
			if v != tt.wantVal {
				t.Errorf("addLabelEntry(%q): label[%q] = %q, want %q", tt.input, tt.wantKey, v, tt.wantVal)
			}
		})
	}
	t.Run("nil map initialised on first add", func(t *testing.T) {
		m := &FilterModal{leftCursor: leftPaneLabels}
		if m.filter.IncludeLabels != nil {
			t.Fatal("expected nil IncludeLabels initially")
		}
		m.addLabelEntry("k=v")
		if m.filter.IncludeLabels == nil {
			t.Fatal("IncludeLabels should be non-nil after addLabelEntry")
		}
		if m.filter.IncludeLabels["k"] != "v" {
			t.Errorf("got %v, want k=v", m.filter.IncludeLabels)
		}
	})
}

// ── selectedItems ─────────────────────────────────────────────────────────────

func TestSelectedItems(t *testing.T) {
	filter := model.DebugLogFilter{
		Level:           "INFO",
		Applications:    []string{"mysql", "wordpress"},
		IncludeEntities: []string{"unit-mysql-0", "machine-0"},
		IncludeModules:  []string{"provider"},
		IncludeLabels:   map[string]string{"env": "prod", "region": "us"},
	}
	tests := []struct {
		name    string
		pane    leftPane
		wantLen int
		wantIn  []string
	}{
		{"applications", leftPaneApplications, 2, []string{"mysql", "wordpress"}},
		{"units excludes machine prefix", leftPaneUnits, 1, []string{"unit-mysql-0"}},
		{"machines only machine- prefix", leftPaneMachines, 1, []string{"machine-0"}},
		{"modules", leftPaneModules, 1, []string{"provider"}},
		{"labels as key=value", leftPaneLabels, 2, []string{"env=prod", "region=us"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &FilterModal{filter: filter, leftCursor: tt.pane}
			got := m.selectedItems()
			if len(got) != tt.wantLen {
				t.Errorf("selectedItems() len = %d, want %d (got %v)", len(got), tt.wantLen, got)
			}
			for _, want := range tt.wantIn {
				found := false
				for _, g := range got {
					if g == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("selectedItems(): %q not found in %v", want, got)
				}
			}
		})
	}
	t.Run("level pane returns nil", func(t *testing.T) {
		m := &FilterModal{filter: filter, leftCursor: leftPaneLevel}
		if got := m.selectedItems(); got != nil {
			t.Errorf("expected nil for leftPaneLevel, got %v", got)
		}
	})
}

// ── availableSuggestions ──────────────────────────────────────────────────────

func TestAvailableSuggestions(t *testing.T) {
	sugg := map[leftPane][]string{
		leftPaneApplications: {"mysql", "wordpress", "postgresql"},
	}
	filter := model.DebugLogFilter{Applications: []string{"mysql"}}

	t.Run("selected items excluded from suggestions", func(t *testing.T) {
		m := &FilterModal{filter: filter, suggestions: sugg, leftCursor: leftPaneApplications}
		got := m.availableSuggestions()
		for _, g := range got {
			if g == "mysql" {
				t.Errorf("availableSuggestions(): selected item 'mysql' should not appear")
			}
		}
		if len(got) != 2 {
			t.Errorf("availableSuggestions() len = %d, want 2 (got %v)", len(got), got)
		}
	})
	t.Run("result is sorted", func(t *testing.T) {
		m := &FilterModal{filter: model.DebugLogFilter{}, suggestions: sugg, leftCursor: leftPaneApplications}
		got := m.availableSuggestions()
		for i := 1; i < len(got); i++ {
			if got[i] < got[i-1] {
				t.Errorf("availableSuggestions() not sorted: %v", got)
				break
			}
		}
	})
	t.Run("nil suggestions returns nil", func(t *testing.T) {
		m := &FilterModal{leftCursor: leftPaneApplications}
		if got := m.availableSuggestions(); got != nil {
			t.Errorf("expected nil for empty suggestions, got %v", got)
		}
	})
}

// ── addSelectedItem / removeSelectedItem ──────────────────────────────────────

func TestAddRemoveSelectedItem(t *testing.T) {
	t.Run("add application", func(t *testing.T) {
		m := &FilterModal{leftCursor: leftPaneApplications}
		m.addSelectedItem("mysql")
		if len(m.filter.Applications) != 1 || m.filter.Applications[0] != "mysql" {
			t.Errorf("expected Applications=[mysql], got %v", m.filter.Applications)
		}
		// duplicate add is a no-op
		m.addSelectedItem("mysql")
		if len(m.filter.Applications) != 1 {
			t.Errorf("duplicate add should be no-op, got %v", m.filter.Applications)
		}
	})
	t.Run("add machine gets machine- prefix", func(t *testing.T) {
		m := &FilterModal{leftCursor: leftPaneMachines}
		m.addSelectedItem("0")
		found := false
		for _, e := range m.filter.IncludeEntities {
			if e == "machine-0" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected IncludeEntities to contain 'machine-0', got %v", m.filter.IncludeEntities)
		}
	})
	t.Run("add machine already prefixed", func(t *testing.T) {
		m := &FilterModal{leftCursor: leftPaneMachines}
		m.addSelectedItem("machine-1")
		if len(m.filter.IncludeEntities) != 1 || m.filter.IncludeEntities[0] != "machine-1" {
			t.Errorf("expected [machine-1], got %v", m.filter.IncludeEntities)
		}
	})
	t.Run("remove application", func(t *testing.T) {
		m := &FilterModal{
			leftCursor: leftPaneApplications,
			filter:     model.DebugLogFilter{Applications: []string{"mysql", "wordpress"}},
		}
		m.removeSelectedItem("mysql")
		if len(m.filter.Applications) != 1 || m.filter.Applications[0] != "wordpress" {
			t.Errorf("after remove, expected [wordpress], got %v", m.filter.Applications)
		}
	})
	t.Run("remove label deletes from map", func(t *testing.T) {
		m := &FilterModal{
			leftCursor: leftPaneLabels,
			filter: model.DebugLogFilter{
				IncludeLabels: map[string]string{"env": "prod", "region": "us"},
			},
		}
		m.removeSelectedItem("env=prod")
		if _, ok := m.filter.IncludeLabels["env"]; ok {
			t.Errorf("label 'env' should have been removed, map: %v", m.filter.IncludeLabels)
		}
		if _, ok := m.filter.IncludeLabels["region"]; !ok {
			t.Errorf("label 'region' should still be present, map: %v", m.filter.IncludeLabels)
		}
	})
}

// ── buildRightRows ────────────────────────────────────────────────────────────

func TestBuildRightRows(t *testing.T) {
	t.Run("level pane has 5 rowSelected rows matching logLevels", func(t *testing.T) {
		m := &FilterModal{leftCursor: leftPaneLevel}
		rows := m.buildRightRows()
		if len(rows) != len(logLevels) {
			t.Fatalf("expected %d rows, got %d", len(logLevels), len(rows))
		}
		for i, r := range rows {
			if r.kind != rowSelected {
				t.Errorf("row[%d] kind = %v, want rowSelected", i, r.kind)
			}
			if r.label != logLevels[i] {
				t.Errorf("row[%d] label = %q, want %q", i, r.label, logLevels[i])
			}
		}
	})
	t.Run("no selected no suggestions gives empty rows", func(t *testing.T) {
		m := &FilterModal{leftCursor: leftPaneApplications}
		rows := m.buildRightRows()
		if len(rows) != 0 {
			t.Errorf("expected empty rows, got %v", rows)
		}
	})
	t.Run("suggestions only no divider when nothing selected", func(t *testing.T) {
		m := &FilterModal{
			leftCursor:  leftPaneApplications,
			suggestions: map[leftPane][]string{leftPaneApplications: {"mysql"}},
		}
		rows := m.buildRightRows()
		for _, r := range rows {
			if r.kind == rowDivider {
				t.Errorf("divider should not appear when nothing is selected")
			}
		}
	})
	t.Run("selected and suggestions inserts divider between them", func(t *testing.T) {
		m := &FilterModal{
			leftCursor: leftPaneApplications,
			filter:     model.DebugLogFilter{Applications: []string{"mysql"}},
			suggestions: map[leftPane][]string{
				leftPaneApplications: {"mysql", "wordpress"},
			},
		}
		rows := m.buildRightRows()
		kinds := make([]rowKind, len(rows))
		for i, r := range rows {
			kinds[i] = r.kind
		}
		hasDivider := false
		for _, k := range kinds {
			if k == rowDivider {
				hasDivider = true
				break
			}
		}
		if !hasDivider {
			t.Errorf("expected a divider row when items are selected; kinds %v", kinds)
		}
		divIdx := -1
		for i, k := range kinds {
			if k == rowDivider {
				divIdx = i
				break
			}
		}
		if divIdx > 0 && kinds[divIdx-1] != rowSelected {
			t.Errorf("divider should follow rowSelected rows, got %v before it", kinds[divIdx-1])
		}
	})
	t.Run("labels pane always has rowAddLabel entry", func(t *testing.T) {
		m := &FilterModal{leftCursor: leftPaneLabels}
		rows := m.buildRightRows()
		found := false
		for _, r := range rows {
			if r.kind == rowAddLabel {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("labels pane should always have a rowAddLabel row; got %v", rows)
		}
	})
}

// ── scrollRightIntoView ───────────────────────────────────────────────────────

func TestScrollRightIntoView(t *testing.T) {
	t.Run("cursor below viewport scrolls offset down", func(t *testing.T) {
		m := &FilterModal{rightCursor: rightPaneVisibleRows + 2, rightViewOffset: 0}
		m.scrollRightIntoView(rightPaneVisibleRows + 5)
		if m.rightViewOffset <= 0 {
			t.Errorf("expected offset > 0 when cursor is below viewport, got %d", m.rightViewOffset)
		}
	})
	t.Run("cursor above viewport scrolls offset up", func(t *testing.T) {
		m := &FilterModal{rightCursor: 1, rightViewOffset: 5}
		m.scrollRightIntoView(20)
		if m.rightViewOffset != 1 {
			t.Errorf("expected offset = 1 (cursor), got %d", m.rightViewOffset)
		}
	})
	t.Run("total less than visibleRows clamps offset to zero", func(t *testing.T) {
		m := &FilterModal{rightCursor: 2, rightViewOffset: 3}
		m.scrollRightIntoView(rightPaneVisibleRows - 1)
		if m.rightViewOffset != 0 {
			t.Errorf("expected offset clamped to 0, got %d", m.rightViewOffset)
		}
	})
}

// ── containsIgnoreCase ────────────────────────────────────────────────────────

func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		s, substr string
		want      bool
	}{
		{"Hello World", "hello", true},
		{"Hello World", "WORLD", true},
		{"Hello World", "xyz", false},
		{"", "a", false},
		{"abc", "", true},
	}
	for _, tt := range tests {
		got := containsIgnoreCase(tt.s, tt.substr)
		if got != tt.want {
			t.Errorf("containsIgnoreCase(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
		}
	}
}

// ── DebugLog helpers ──────────────────────────────────────────────────────────

func TestDebugLog_SetFilterActiveFilter(t *testing.T) {
	d := New(ui.DefaultKeyMap())
	f := model.DebugLogFilter{Level: "DEBUG", Applications: []string{"mysql"}}
	d.SetFilter(f)
	got := d.ActiveFilter()
	if got.Level != f.Level || len(got.Applications) != 1 || got.Applications[0] != f.Applications[0] {
		t.Errorf("ActiveFilter() = %+v, want %+v", got, f)
	}
}

func TestDebugLog_IsModalOpen(t *testing.T) {
	d := New(ui.DefaultKeyMap())
	if d.IsModalOpen() {
		t.Error("expected IsModalOpen() = false on new DebugLog")
	}
	d.mode = debugModeFilter
	if !d.IsModalOpen() {
		t.Error("expected IsModalOpen() = true when mode is debugModeFilter")
	}
	d.mode = debugModeSearch
	if d.IsModalOpen() {
		t.Error("expected IsModalOpen() = false when mode is debugModeSearch")
	}
}

// stripANSI removes ANSI escape sequences for plain-text test assertions.
func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		switch {
		case r == '\x1b':
			inEsc = true
		case inEsc && r == 'm':
			inEsc = false
		case inEsc:
			// still inside escape sequence
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func TestDebugLog_FilterTitle(t *testing.T) {
	tests := []struct {
		name         string
		filter       model.DebugLogFilter
		wantEmpty    bool
		wantContains []string
	}{
		{"empty filter", model.DebugLogFilter{}, true, nil},
		{"level chip", model.DebugLogFilter{Level: "DEBUG"}, false, []string{"level=DEBUG"}},
		{"application chip", model.DebugLogFilter{Applications: []string{"mysql"}}, false, []string{"app=mysql"}},
		{"entity chip", model.DebugLogFilter{IncludeEntities: []string{"unit-mysql-0"}}, false, []string{"entity=unit-mysql-0"}},
		{"module chip", model.DebugLogFilter{IncludeModules: []string{"provider"}}, false, []string{"module=provider"}},
		{"label chip", model.DebugLogFilter{IncludeLabels: map[string]string{"env": "prod"}}, false, []string{"label="}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New(ui.DefaultKeyMap())
			d.SetFilter(tt.filter)
			got := d.FilterTitle()
			if tt.wantEmpty && got != "" {
				t.Errorf("FilterTitle() = %q, want empty", got)
			}
			if !tt.wantEmpty && got == "" {
				t.Errorf("FilterTitle() = empty, want non-empty")
			}
			plain := stripANSI(got)
			for _, sub := range tt.wantContains {
				if !strings.Contains(plain, sub) {
					t.Errorf("FilterTitle() does not contain %q in plain output %q", sub, plain)
				}
			}
		})
	}
}

func TestDebugLog_BuildSuggestions(t *testing.T) {
	t.Run("nil status returns empty map", func(t *testing.T) {
		d := New(ui.DefaultKeyMap())
		sugg := d.buildSuggestions()
		if len(sugg) != 0 {
			t.Errorf("expected empty suggestions with nil status, got %v", sugg)
		}
	})
	t.Run("applications and units populated from status", func(t *testing.T) {
		d := New(ui.DefaultKeyMap())
		d.status = &model.FullStatus{
			Applications: map[string]model.Application{
				"mysql": {
					Units: []model.Unit{
						{Name: "mysql/0"},
					},
				},
				"wordpress": {
					Units: []model.Unit{
						{Name: "wordpress/0"},
					},
				},
			},
			Machines: map[string]model.Machine{
				"0": {ID: "0"},
			},
		}
		sugg := d.buildSuggestions()
		if len(sugg[leftPaneApplications]) != 2 {
			t.Errorf("expected 2 app suggestions, got %v", sugg[leftPaneApplications])
		}
		if len(sugg[leftPaneUnits]) != 2 {
			t.Errorf("expected 2 unit suggestions, got %v", sugg[leftPaneUnits])
		}
		if len(sugg[leftPaneMachines]) != 1 {
			t.Errorf("expected 1 machine suggestion, got %v", sugg[leftPaneMachines])
		}
		if sugg[leftPaneMachines][0] != "machine-0" {
			t.Errorf("machine suggestion = %q, want 'machine-0'", sugg[leftPaneMachines][0])
		}
	})
	t.Run("seenModules included in module suggestions", func(t *testing.T) {
		d := New(ui.DefaultKeyMap())
		d.seenModules = map[string]struct{}{"provider": {}, "worker": {}}
		sugg := d.buildSuggestions()
		if len(sugg[leftPaneModules]) != 2 {
			t.Errorf("expected 2 module suggestions, got %v", sugg[leftPaneModules])
		}
	})
}

func TestDebugLog_ScrollHelpers(t *testing.T) {
	d := New(ui.DefaultKeyMap())
	for i := 0; i < 50; i++ {
		d.lines = append(d.lines, "line")
	}
	d.height = 20

	t.Run("scrollUp clamps at zero", func(t *testing.T) {
		d.offset = 0
		d.scrollUp(5)
		if d.offset != 0 {
			t.Errorf("scrollUp below 0 should clamp to 0, got %d", d.offset)
		}
	})
	t.Run("scrollDown clamps at bottomOffset", func(t *testing.T) {
		d.offset = 0
		d.scrollDown(1000)
		if d.offset != d.bottomOffset() {
			t.Errorf("scrollDown past end should clamp to %d, got %d", d.bottomOffset(), d.offset)
		}
	})
	t.Run("visibleLines fallback for zero height", func(t *testing.T) {
		d2 := New(ui.DefaultKeyMap())
		if d2.visibleLines() != 20 {
			t.Errorf("visibleLines() with height=0 should return 20, got %d", d2.visibleLines())
		}
	})
}

func TestDebugLog_RebuildSearchMatches(t *testing.T) {
	d := New(ui.DefaultKeyMap())
	d.rawEntries = []model.LogEntry{
		{Message: "Hello world", Entity: "unit-mysql-0", Module: "db"},
		{Message: "another entry", Entity: "unit-wp-0", Module: "web"},
		{Message: "HELLO again", Entity: "unit-mysql-1", Module: "db"},
	}
	d.lines = make([]string, len(d.rawEntries))

	t.Run("empty query clears matches", func(t *testing.T) {
		d.searchMatches = []int{0, 1}
		d.searchQuery = ""
		d.rebuildSearchMatches()
		if len(d.searchMatches) != 0 {
			t.Errorf("expected empty matches for empty query, got %v", d.searchMatches)
		}
	})
	t.Run("case-insensitive message match", func(t *testing.T) {
		d.searchQuery = "hello"
		d.rebuildSearchMatches()
		if len(d.searchMatches) != 2 {
			t.Errorf("expected 2 matches for 'hello', got %v", d.searchMatches)
		}
	})
	t.Run("entity match", func(t *testing.T) {
		d.searchQuery = "mysql"
		d.rebuildSearchMatches()
		if len(d.searchMatches) != 2 {
			t.Errorf("expected 2 matches for 'mysql' (entity), got %v", d.searchMatches)
		}
	})
	t.Run("no match returns empty", func(t *testing.T) {
		d.searchQuery = "zzznomatch"
		d.rebuildSearchMatches()
		if len(d.searchMatches) != 0 {
			t.Errorf("expected 0 matches, got %v", d.searchMatches)
		}
	})
}

func TestDebugLog_JumpToNextMatch(t *testing.T) {
	d := New(ui.DefaultKeyMap())
	d.lines = make([]string, 10)
	d.searchMatches = []int{2, 5, 8}

	t.Run("forward wraps around", func(t *testing.T) {
		d.searchIdx = 2
		d.jumpToNextMatch(1)
		if d.searchIdx != 0 {
			t.Errorf("expected wrap to 0, got %d", d.searchIdx)
		}
		if d.offset != d.searchMatches[0] {
			t.Errorf("offset should be searchMatches[0]=%d, got %d", d.searchMatches[0], d.offset)
		}
	})
	t.Run("backward wraps around", func(t *testing.T) {
		d.searchIdx = 0
		d.jumpToNextMatch(-1)
		if d.searchIdx != 2 {
			t.Errorf("expected wrap to 2, got %d", d.searchIdx)
		}
	})
	t.Run("empty matches is no-op", func(t *testing.T) {
		d2 := New(ui.DefaultKeyMap())
		d2.offset = 0
		d2.jumpToNextMatch(1)
		if d2.offset != 0 {
			t.Errorf("expected no change when no matches, offset = %d", d2.offset)
		}
	})
	t.Run("sets paused true", func(t *testing.T) {
		d.paused = false
		d.jumpToNextMatch(1)
		if !d.paused {
			t.Error("jumpToNextMatch should set paused=true")
		}
	})
}
