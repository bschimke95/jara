// Package api defines the Client interface through which jara communicates with
// a Juju controller, along with message types used to propagate status updates
// to the TUI.
package api

import (
	"context"
	"errors"
	"time"

	"github.com/bschimke95/jara/internal/model"
)

// ErrNoSelectedModel is returned when a Juju operation requires a model but
// none is currently selected (e.g. the user has not yet run `juju switch`).
// Callers should check for this with errors.Is(err, ErrNoSelectedModel).
var ErrNoSelectedModel = errors.New("no selected model")

// StatusClient provides read-only access to Juju model status.
type StatusClient interface {
	// Status fetches a one-shot status snapshot of the current model.
	Status(ctx context.Context) (*model.FullStatus, error)
	// WatchStatus starts a background loop that pushes status snapshots onto
	// the returned channel at the given interval. The stream runs until the
	// context is cancelled. On transient errors the implementation should
	// reconnect with backoff rather than closing the channel.
	WatchStatus(ctx context.Context, interval time.Duration) (<-chan StatusUpdate, error)
}

// ApplicationClient groups operations that manage applications and their
// lifecycle (scale, deploy, config, actions).
type ApplicationClient interface {
	// ScaleApplication adjusts the unit count for an application by delta
	// (positive to scale up, negative to scale down).
	ScaleApplication(ctx context.Context, appName string, delta int) error
	// DeployApplication deploys a charm with deploy options in the current model.
	DeployApplication(ctx context.Context, opts model.DeployOptions) error
	// AppConfig returns the configuration key-value pairs for an application.
	AppConfig(ctx context.Context, appName string) ([]model.ConfigEntry, error)
	// ApplicationActions returns the available charm actions for an application.
	ApplicationActions(ctx context.Context, appName string) ([]model.ActionSpec, error)
	// RunAction executes a named action on a unit and waits for the result.
	RunAction(ctx context.Context, unitName, actionName string, params map[string]string) (*model.ActionResult, error)
}

// RelationClient groups operations for managing relations between applications.
type RelationClient interface {
	// RelateApplications adds a relation between two endpoints.
	// Each endpoint is either "appName" or "appName:endpointName".
	RelateApplications(ctx context.Context, endpointA, endpointB string) error
	// DestroyRelation removes a relation between two endpoints.
	DestroyRelation(ctx context.Context, endpointA, endpointB string) error
	// RelationData fetches the application and unit databag contents for the
	// given relation ID.
	RelationData(ctx context.Context, relationID int) (*model.RelationData, error)
}

// SecretClient provides access to Juju secrets.
type SecretClient interface {
	// ListSecrets returns the secrets for the current model.
	ListSecrets(ctx context.Context) ([]model.Secret, error)
	// RevealSecret returns the decoded key-value content of a secret.
	// When revision is 0 the latest revision is revealed.
	RevealSecret(ctx context.Context, uri string, revision int) (map[string]string, error)
}

// CharmhubClient provides lookups against the Charmhub store.
type CharmhubClient interface {
	// CharmhubSuggestions returns charm names from Charmhub for autocomplete.
	CharmhubSuggestions(ctx context.Context, query string, limit int) ([]string, error)
	// CharmRelationInfo returns endpoint metadata for a charm from Charmhub.
	CharmRelationInfo(ctx context.Context, charmName string) (map[string]model.CharmEndpoint, error)
}

// InfrastructureClient groups operations for models, controllers, storage,
// offers, and debug logging.
type InfrastructureClient interface {
	// Controllers returns details for all known controllers.
	Controllers(ctx context.Context) ([]model.Controller, error)
	// Models returns the list of models for a controller.
	Models(ctx context.Context, controllerName string) ([]model.ModelSummary, error)
	// DebugLog connects to the controller and streams debug log messages.
	DebugLog(ctx context.Context, filter model.DebugLogFilter) (<-chan model.LogEntry, error)
	// ListStorage returns all storage instances in the current model.
	ListStorage(ctx context.Context) ([]model.StorageInstance, error)
	// ListOffers returns application offers for the current model.
	ListOffers(ctx context.Context) ([]model.Offer, error)
	// SelectController switches the client to target a different controller.
	SelectController(name string) error
	// SelectModel switches the client to target the given model within the
	// current controller.
	SelectModel(qualifiedName string) error
	// ControllerName returns the name of the currently targeted controller.
	ControllerName() string
}

// Client defines the full interface for fetching Juju status and managing
// infrastructure. It composes smaller role interfaces so that consumers can
// depend on only the subset they need.
type Client interface {
	StatusClient
	ApplicationClient
	RelationClient
	SecretClient
	CharmhubClient
	InfrastructureClient
	Close() error
}

// StatusUpdate carries either a successful status snapshot or an error from
// the watch loop. Consumers should check Err first.
type StatusUpdate struct {
	Status *model.FullStatus
	Err    error
}
