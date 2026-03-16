package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/bschimke95/jara/internal/api"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/view"
)

// statusStreamConnectedMsg is sent when the status stream is established.
type statusStreamConnectedMsg struct {
	ctx context.Context
	ch  <-chan api.StatusUpdate
}

// statusStreamUpdateMsg delivers a status snapshot from the watch stream.
type statusStreamUpdateMsg struct {
	status *model.FullStatus
	ctx    context.Context
	ch     <-chan api.StatusUpdate
}

// statusStreamErrMsg delivers a transient error from the watch stream.
// The stream is still running; the next read is scheduled automatically.
type statusStreamErrMsg struct {
	err error
	ctx context.Context
	ch  <-chan api.StatusUpdate
}

type (
	errMsg     struct{ err error }
	noModelMsg struct{} // sent when no model is selected in the current controller
)

// startStatusStream begins streaming status updates from the API.
// It returns a Cmd that connects and sends a statusStreamConnectedMsg.
func (m *Model) startStatusStream() tea.Cmd {
	m.stopStatusStream() // cancel any existing stream

	ctx, cancel := context.WithCancel(context.Background())
	m.statusCancel = cancel

	client := m.client
	refreshDuration := m.cfg.RefreshDuration()
	return func() tea.Msg {
		ch, err := client.WatchStatus(ctx, refreshDuration)
		if err != nil {
			if strings.Contains(err.Error(), "No selected model") ||
				strings.Contains(err.Error(), "resolving current model") {
				return noModelMsg{}
			}
			return errMsg{err}
		}
		return statusStreamConnectedMsg{ctx: ctx, ch: ch}
	}
}

// stopStatusStream cancels the active status stream, if any.
func (m *Model) stopStatusStream() {
	if m.statusCancel != nil {
		m.statusCancel()
		m.statusCancel = nil
		m.statusCh = nil
	}
}

// readNextStatus returns a Cmd that blocks until the next StatusUpdate
// arrives from the watch channel.
func readNextStatus(ctx context.Context, ch <-chan api.StatusUpdate) tea.Cmd {
	return func() tea.Msg {
		select {
		case <-ctx.Done():
			return nil
		case update, ok := <-ch:
			if !ok {
				return errMsg{err: fmt.Errorf("status stream closed")}
			}
			if update.Err != nil {
				if strings.Contains(update.Err.Error(), "No selected model") {
					return noModelMsg{}
				}
				return statusStreamErrMsg{err: update.Err, ctx: ctx, ch: ch}
			}
			return statusStreamUpdateMsg{status: update.Status, ctx: ctx, ch: ch}
		}
	}
}

// pollControllers fetches the controller list from the local client store.
func (m Model) pollControllers() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		controllers, err := m.client.Controllers(ctx)
		if err != nil {
			return errMsg{err}
		}
		return view.ControllersUpdatedMsg{Controllers: controllers}
	}
}

// pollModels fetches the model list for the given controller from the local client store.
func (m Model) pollModels(controllerName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		models, err := m.client.Models(ctx, controllerName)
		if err != nil {
			return errMsg{err}
		}
		return view.ModelsUpdatedMsg{Models: models}
	}
}

// scaleApplication returns a Cmd that calls ScaleApplication on the API client.
func (m Model) scaleApplication(appName string, delta int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := m.client.ScaleApplication(ctx, appName, delta); err != nil {
			return errMsg{err}
		}
		return nil
	}
}
