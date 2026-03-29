package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/bschimke95/jara/internal/api"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/nav"
	"github.com/bschimke95/jara/internal/view"
	"github.com/bschimke95/jara/internal/view/controllers"
	"github.com/bschimke95/jara/internal/view/models"
	"github.com/bschimke95/jara/internal/view/modelview"
	"github.com/bschimke95/jara/internal/view/relations"
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

type errMsg struct{ err error }

type charmhubSuggestionsMsg struct {
	Names []string
}

type charmEndpointsMsg struct {
	// Endpoints maps charm name → endpoint name → CharmEndpoint.
	Endpoints map[string]map[string]model.CharmEndpoint
}

type secretsMsg struct {
	Secrets []model.Secret
}

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
				return modelview.NoModelMsg{}
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
					return modelview.NoModelMsg{}
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
		ctrlList, err := m.client.Controllers(ctx)
		if err != nil {
			return errMsg{err}
		}
		return controllers.UpdatedMsg{Controllers: ctrlList}
	}
}

// pollModels fetches the model list for the given controller from the local client store.
func (m Model) pollModels(controllerName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		modelList, err := m.client.Models(ctx, controllerName)
		if err != nil {
			return errMsg{err}
		}
		return models.UpdatedMsg{Models: modelList}
	}
}

// pollCharmhubSuggestions fetches charm names for deploy autocomplete.
func (m Model) pollCharmhubSuggestions() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		names, err := m.client.CharmhubSuggestions(ctx, "", 200)
		if err != nil {
			// Suggestions are best-effort; keep UI functional without them.
			return nil
		}
		return charmhubSuggestionsMsg{Names: names}
	}
}

// pollCharmEndpoints fetches endpoint metadata for all unique charms
// in the current model status from Charmhub.
func (m Model) pollCharmEndpoints() tea.Cmd {
	if m.status == nil {
		return nil
	}
	// Collect unique charm names.
	charms := make(map[string]struct{})
	for _, app := range m.status.Applications {
		if app.Charm != "" {
			charms[app.Charm] = struct{}{}
		}
	}
	if len(charms) == 0 {
		return nil
	}
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		result := make(map[string]map[string]model.CharmEndpoint, len(charms))
		for name := range charms {
			eps, err := client.CharmRelationInfo(ctx, name)
			if err != nil || eps == nil {
				continue
			}
			result[name] = eps
		}
		if len(result) == 0 {
			return nil
		}
		return charmEndpointsMsg{Endpoints: result}
	}
}

// pollSecrets fetches the secrets for the current model from the API.
func (m Model) pollSecrets() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		secs, err := client.ListSecrets(ctx)
		if err != nil {
			return errMsg{fmt.Errorf("listing secrets: %w", err)}
		}
		return secretsMsg{Secrets: secs}
	}
}

// revealSecret returns a Cmd that fetches the decoded content of a secret.
func (m Model) revealSecret(uri string, revision int) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		values, err := client.RevealSecret(ctx, uri, revision)
		if err != nil {
			return errMsg{fmt.Errorf("revealing secret: %w", err)}
		}
		return view.RevealSecretResponseMsg{URI: uri, Values: values}
	}
}

// scaleApplication returns a Cmd that calls ScaleApplication on the API client.
func (m Model) scaleApplication(appName string, delta int) tea.Cmd {
	return func() tea.Msg {
		if m.cfg != nil && m.cfg.Jara.ReadOnly {
			return errMsg{fmt.Errorf("write operations are disabled in read-only mode")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := m.client.ScaleApplication(ctx, appName, delta); err != nil {
			return errMsg{err}
		}
		return nil
	}
}

// deployApplication returns a Cmd that deploys a new application charm.
// If modelName is set, deployment is targeted to that model first.
func (m Model) deployApplication(modelName string, opts model.DeployOptions) tea.Cmd {
	return func() tea.Msg {
		if m.cfg != nil && m.cfg.Jara.ReadOnly {
			return errMsg{fmt.Errorf("write operations are disabled in read-only mode")}
		}
		if strings.TrimSpace(opts.CharmName) == "" {
			return errMsg{fmt.Errorf("charm name is required")}
		}
		if modelName == "" {
			current := m.stack.Current()
			if current.View == nav.ModelView && current.Context != "" {
				modelName = current.Context
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if modelName != "" {
			if err := m.client.SelectModel(modelName); err != nil {
				return errMsg{fmt.Errorf("selecting model %q: %w", modelName, err)}
			}
		}
		if err := m.client.DeployApplication(ctx, opts); err != nil {
			return errMsg{err}
		}
		return nil
	}
}

// relateApplications returns a Cmd that creates a relation between two endpoints.
func (m Model) relateApplications(endpointA, endpointB string) tea.Cmd {
	return func() tea.Msg {
		if m.cfg != nil && m.cfg.Jara.ReadOnly {
			return errMsg{fmt.Errorf("write operations are disabled in read-only mode")}
		}
		if strings.TrimSpace(endpointA) == "" || strings.TrimSpace(endpointB) == "" {
			return errMsg{fmt.Errorf("both endpoints are required for a relation")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := m.client.RelateApplications(ctx, endpointA, endpointB); err != nil {
			return errMsg{err}
		}
		return nil
	}
}

// destroyRelation returns a Cmd that removes a relation between two endpoints.
func (m Model) destroyRelation(endpointA, endpointB string) tea.Cmd {
	return func() tea.Msg {
		if m.cfg != nil && m.cfg.Jara.ReadOnly {
			return errMsg{fmt.Errorf("write operations are disabled in read-only mode")}
		}
		if strings.TrimSpace(endpointA) == "" || strings.TrimSpace(endpointB) == "" {
			return errMsg{fmt.Errorf("both endpoints are required to remove a relation")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := m.client.DestroyRelation(ctx, endpointA, endpointB); err != nil {
			return errMsg{err}
		}
		return nil
	}
}

// fetchRelationData returns a Cmd that fetches the databag contents for a relation.
func (m Model) fetchRelationData(relationID int) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		data, err := client.RelationData(ctx, relationID)
		if err != nil {
			return errMsg{err}
		}
		return relations.RelationDataMsg{RelationID: relationID, Data: data}
	}
}
