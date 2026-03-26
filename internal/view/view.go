// Package view defines the View contract that all self-contained view packages
// must implement, along with shared types and the ViewConfig used for
// dependency injection of theme and key overrides.
package view

import (
	tea "charm.land/bubbletea/v2"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/nav"
	"github.com/bschimke95/jara/internal/ui"
)

// StatusUpdatedMsg is sent when fresh status data arrives from the API.
type StatusUpdatedMsg struct {
	Status *model.FullStatus
}

// NavigateMsg requests navigation to a different view.
type NavigateMsg struct {
	Target  nav.ViewID
	Context string
	// Filter is an optional debug-log filter applied when Target is DebugLogView.
	Filter *model.DebugLogFilter
	// ResetStack discards the navigation history and makes Target the sole entry.
	ResetStack bool
}

// GoBackMsg requests navigation back to the previous view.
type GoBackMsg struct{}

// ScaleRequestMsg requests that an application be scaled by the given delta.
type ScaleRequestMsg struct {
	AppName string
	Delta   int
}

// DeployRequestMsg requests deploying a charm with the provided options.
// ModelName is optional; when set, deployment should target that model.
type DeployRequestMsg struct {
	ModelName string
	Options   model.DeployOptions
}

// RelateRequestMsg requests adding a relation between two endpoints.
type RelateRequestMsg struct {
	EndpointA string
	EndpointB string
}

// DestroyRelationRequestMsg requests removing a relation between two endpoints.
type DestroyRelationRequestMsg struct {
	EndpointA string
	EndpointB string
}

// RevealSecretRequestMsg requests decoding a secret's content via the API.
// When Revision is 0 the latest revision is revealed.
type RevealSecretRequestMsg struct {
	URI      string
	Revision int
}

// RevealSecretResponseMsg carries the decoded key-value content of a secret.
type RevealSecretResponseMsg struct {
	URI    string
	Values map[string]string
}

// KeyHint represents a single key-description pair for the header hint bar.
type KeyHint = ui.KeyHint

// NavigateContext carries parameters passed to a view on Enter.
type NavigateContext struct {
	// Context is an optional string parameter (e.g. app name, controller name).
	Context string
	// Filter is an optional debug-log filter.
	Filter *model.DebugLogFilter
}

// View is the interface all resource views must implement.
// Each view is self-contained: it owns its own rendering, types, and messages.
type View interface {
	tea.Model

	// SetSize informs the view of the available content area dimensions.
	SetSize(width, height int)

	// KeyHints returns the view-specific key hints to display in the header.
	// These are merged on top of the global hints by the app chrome.
	KeyHints() []KeyHint

	// Enter is called when the view becomes active (navigated to or returned
	// to via back). Views use this to refresh data or reset internal state.
	// The returned command is batched with any app-level commands.
	// A non-nil error aborts the navigation.
	Enter(ctx NavigateContext) (tea.Cmd, error)

	// Leave is called when the view is about to become inactive (navigated
	// away from). Views use this to clean up transient state. The returned
	// command is batched with any app-level commands.
	Leave() tea.Cmd
}

// StopStatusStreamMsg requests the app stop the active status stream.
type StopStatusStreamMsg struct{}

// StartStatusStreamMsg requests the app start a new status stream.
type StartStatusStreamMsg struct{}

// StartDebugLogStreamMsg requests the app start a debug-log stream.
type StartDebugLogStreamMsg struct {
	Filter model.DebugLogFilter
}

// StopDebugLogStreamMsg requests the app stop the active debug-log stream.
type StopDebugLogStreamMsg struct{}

// ClearStatusMsg requests the app clear the cached status on all views.
type ClearStatusMsg struct{}

// StatusReceiver is implemented by views that consume model status updates.
// Views that don't need FullStatus (e.g. Controllers, Models) simply don't
// implement this interface.
type StatusReceiver interface {
	SetStatus(status *model.FullStatus)
}

// CharmSuggestionReceiver is implemented by views that can consume external
// charm name suggestions (e.g. from Charmhub) for deploy autocomplete.
type CharmSuggestionReceiver interface {
	SetCharmSuggestions(names []string)
}

// CharmEndpointReceiver is implemented by views that consume charm endpoint
// metadata from Charmhub (relation descriptions, interface info).
type CharmEndpointReceiver interface {
	SetCharmEndpoints(endpoints map[string]map[string]model.CharmEndpoint)
}
