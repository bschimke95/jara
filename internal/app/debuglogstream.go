package app

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/view/debuglog"
)

// debugLogConnectedMsg is sent when the debug-log stream is established.
type debugLogConnectedMsg struct {
	ctx context.Context
	ch  <-chan model.LogEntry
}

// startDebugLogStream begins streaming debug-log entries from the API.
// It returns a Cmd that connects and sends a debugLogConnectedMsg.
func (m *Model) startDebugLogStream(filter model.DebugLogFilter) tea.Cmd {
	m.stopDebugLogStream() // cancel any existing stream

	ctx, cancel := context.WithCancel(context.Background())
	m.logCancel = cancel

	client := m.client
	return func() tea.Msg {
		ch, err := client.DebugLog(ctx, filter)
		if err != nil {
			return debuglog.ErrMsg{Err: err}
		}
		return debugLogConnectedMsg{ctx: ctx, ch: ch}
	}
}

// stopDebugLogStream cancels the active debug-log context, if any.
func (m *Model) stopDebugLogStream() {
	if m.logCancel != nil {
		m.logCancel()
		m.logCancel = nil
	}
}

// readFirstLogBatch returns a Cmd that reads the first batch from the
// debug-log stream channel after the stream connects.
func readFirstLogBatch(ctx context.Context, ch <-chan model.LogEntry) tea.Cmd {
	return func() tea.Msg {
		return debuglog.ReadNextLogBatch(ctx, ch)
	}
}
