// Package juju provides a client for interacting with Juju models and controllers.
package juju

import (
	"context"

	"github.com/bschimke95/jara/pkg/types/juju"
)

// Client provides methods to interact with Juju models and controllers.
// It implements the JujuClient interface.
type Client struct {
}

// NewClient creates a new Client.
func NewClient() JujuClient {
	return &Client{}
}

// TODO: Implement me
func (c *Client) CurrentController(ctx context.Context) (juju.Controller, error) {
	return juju.Controller{}, nil
}

func (c *Client) Models(ctx context.Context, controllerName string) ([]juju.Model, error) {
	return []juju.Model{}, nil
}

func (c *Client) CurrentModel(ctx context.Context, controllerName string) (juju.Model, error) {
	return juju.Model{}, nil
}

// Ensure Client implements JujuClient interface
var _ JujuClient = &Client{}
