// Package juju provides a client for interacting with Juju models and controllers.
package juju

import (
	"context"

	"github.com/bschimke95/jara/pkg/types/juju"
)

// JujuClient provides methods to interact with Juju API
type JujuClient interface {
	// CurrentController returns the name of the current controller.
	CurrentController(ctx context.Context) (juju.Controller, error)
	// Models returns a list of models for the current controller.
	Models(ctx context.Context) ([]juju.Model, error)
	// CurrentModel returns the current model for the current controller.
	CurrentModel(ctx context.Context) (juju.Model, error)
}
