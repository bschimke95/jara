package app

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/view"
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
			return view.DebugLogErrMsg{Err: err}
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

// readNextLogBatch returns a Cmd that reads the next batch from the
// debug-log stream. The context and channel are passed through closures
// rather than stored on the model.
func readNextLogBatch(ctx context.Context, ch <-chan model.LogEntry) tea.Cmd {
	return func() tea.Msg {
		return readDebugLogBatch(ctx, ch)
	}
}

// readDebugLogBatch reads available log entries from the channel,
// batching them together before delivering to the view.
func readDebugLogBatch(ctx context.Context, ch <-chan model.LogEntry) tea.Msg {
	// Block on the first entry.
	select {
	case <-ctx.Done():
		return nil
	case entry, ok := <-ch:
		if !ok {
			return view.DebugLogErrMsg{Err: fmt.Errorf("log stream closed")}
		}
		batch := []model.LogEntry{entry}
		// Drain any additional immediately-available entries.
	drain:
		for {
			select {
			case e, ok := <-ch:
				if !ok {
					break drain
				}
				batch = append(batch, e)
				if len(batch) >= 50 {
					break drain
				}
			default:
				break drain
			}
		}
		return view.DebugLogMsg{Entries: batch, Ctx: ctx, Ch: ch}
	}
}
