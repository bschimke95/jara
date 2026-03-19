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

// KeyHint represents a single key-description pair for the header hint bar.
type KeyHint = ui.KeyHint

// View is the interface all resource views must implement.
// Each view is self-contained: it owns its own rendering, types, and messages.
type View interface {
	tea.Model

	// SetSize informs the view of the available content area dimensions.
	SetSize(width, height int)

	// KeyHints returns the view-specific key hints to display in the header.
	// These are merged on top of the global hints by the app chrome.
	KeyHints() []KeyHint
}

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
