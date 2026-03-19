// Package api defines the Client interface through which jara communicates with
// a Juju controller, along with message types used to propagate status updates
// to the TUI.
package api

import (
	"context"
	"time"

	"github.com/bschimke95/jara/internal/model"
)

// Client defines the interface for fetching Juju status.
type Client interface {
	Status(ctx context.Context) (*model.FullStatus, error)
	Controllers(ctx context.Context) ([]model.Controller, error)
	Models(ctx context.Context, controllerName string) ([]model.ModelSummary, error)
	DebugLog(ctx context.Context, filter model.DebugLogFilter) (<-chan model.LogEntry, error)
	// WatchStatus starts a background loop that pushes status snapshots onto
	// the returned channel at the given interval. The stream runs until the
	// context is cancelled. On transient errors the implementation should
	// reconnect with backoff rather than closing the channel.
	WatchStatus(ctx context.Context, interval time.Duration) (<-chan StatusUpdate, error)
	// ScaleApplication adjusts the unit count for an application by delta
	// (positive to scale up, negative to scale down).
	ScaleApplication(ctx context.Context, appName string, delta int) error
	// DeployApplication deploys a charm with deploy options in the current model.
	DeployApplication(ctx context.Context, opts model.DeployOptions) error
	// CharmhubSuggestions returns charm names from Charmhub for autocomplete.
	CharmhubSuggestions(ctx context.Context, query string, limit int) ([]string, error)
	// SelectController switches the client to target a different controller.
	SelectController(name string) error
	// SelectModel switches the client to target the given model within the
	// current controller.
	SelectModel(qualifiedName string) error
	// ControllerName returns the name of the currently targeted controller.
	ControllerName() string
	Close() error
}

// StatusUpdate carries either a successful status snapshot or an error from
// the watch loop. Consumers should check Err first.
type StatusUpdate struct {
	Status *model.FullStatus
	Err    error
}
