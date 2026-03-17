package debuglog

import (
	"context"

	"charm.land/bubbles/v2/textinput"

	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/ui"
)

const maxLogLines = 1000

// Msg delivers a batch of new log entries to the view.
type Msg struct {
	Entries []model.LogEntry
	Ctx     context.Context
	Ch      <-chan model.LogEntry
}

// ErrMsg signals that the debug-log stream encountered an error.
type ErrMsg struct {
	Err error
}

// FilterChangedMsg is emitted when the user applies a new filter from
// inside the debug-log view. The app handles this by restarting the stream.
type FilterChangedMsg struct {
	Filter model.DebugLogFilter
}

// debugMode represents the active sub-mode inside the debug-log view.
type debugMode int

const (
	debugModeNormal debugMode = iota
	debugModeFilter
	debugModeSearch
)

// View is the Bubble Tea model for the debug-log streaming view.
type View struct {
	keys   ui.KeyMap
	width  int
	height int

	lines      []string
	rawEntries []model.LogEntry
	offset     int
	paused     bool

	mode         debugMode
	filterModal  FilterModal
	activeFilter model.DebugLogFilter

	status      *model.FullStatus
	seenModules map[string]struct{}

	searchInput   textinput.Model
	searchQuery   string
	searchMatches []int
	searchIdx     int
}
