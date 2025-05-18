// Package juju provides a client for interacting with Juju models and controllers.
package juju

import (
	"context"

	"github.com/bschimke95/jara/pkg/types/juju"
	"github.com/juju/errors"
)

// JujuClient provides methods to interact with Juju models and controllers.
type JujuClient struct {
	store JujuStore
}

// NewJujuClient creates a new JujuClient with the given store.
func NewJujuClient(store JujuStore) *JujuClient {
	return &JujuClient{
		store: store,
	}
}

// JujuStore defines the interface for Juju store operations.
type JujuStore interface {
	// CurrentController returns the name of the current controller.
	CurrentController() (string, error)
	// ControllerByName returns the controller with the given name.
	ControllerByName(string) (*juju.Controller, error)
	// CurrentModel returns the name of the current model for the given controller.
	CurrentModel(controllerName string) (string, error)
	// ModelByName returns the model with the given name from the specified controller.
	ModelByName(controllerName, modelName string) (*juju.Model, error)
	// AccountDetails returns the account details for the given controller.
	AccountDetails(controllerName string) (*juju.Account, error)
}

// CurrentModel retrieves the current model's basic information.
// TODO: Implement full model status retrieval once the Juju API integration is complete.
func (c *JujuClient) CurrentModel(ctx context.Context) (juju.Model, error) {
	emptyModel := juju.Model{}

	// Get current controller
	controllerName, err := c.store.CurrentController()
	if err != nil {
		return emptyModel, errors.Annotate(err, "getting current controller")
	}

	// Get current model name
	modelName, err := c.store.CurrentModel(controllerName)
	if err != nil {
		return emptyModel, errors.Annotate(err, "getting current model")
	}

	// Get model details from store
	model, err := c.store.ModelByName(controllerName, modelName)
	if err != nil {
		return emptyModel, errors.Annotatef(err, "getting model %q", modelName)
	}

	// Return basic model information for now
	return *model, nil
}
