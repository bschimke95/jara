package juju

import (
	"context"
	"fmt"
	"log"

	"github.com/juju/errors"
	"github.com/juju/juju/api"
	"github.com/juju/juju/api/base"
	"github.com/juju/juju/api/client/modelmanager"
	"github.com/juju/juju/jujuclient"
	"github.com/juju/names/v5"
)

// JujuClient wraps the jujuclient.Client and adds functionality for our application.
type JujuClient struct {
	store jujuclient.ClientStore
}

// NewClient creates and returns a new Juju client.
func NewJujuClient() *JujuClient {
	return &JujuClient{
		store: jujuclient.NewFileClientStore(),
	}
}

func (c *JujuClient) CurrentController() (*jujuclient.ControllerDetails, error) {
	controllerName, err := c.store.CurrentController()
	if err != nil {
		return nil, errors.Annotate(err, "failed to get current controller from local store")
	}
	controller, err := c.store.ControllerByName(controllerName)
	if err != nil {
		return nil, errors.Annotate(err, fmt.Sprintf("failed to get controller by name %s from local store", controllerName))
	}

	return controller, nil
}

func (c *JujuClient) CurrentModel() (base.ModelStatus, error) {
	controllerName, err := c.store.CurrentController()
	if err != nil {
		return base.ModelStatus{}, errors.Annotate(err, "failed to get current controller from local store")
	}

	controller, err := c.store.ControllerByName(controllerName)
	if err != nil {
		return base.ModelStatus{}, errors.Annotate(err, fmt.Sprintf("failed to get controller by name %s from local store", controllerName))
	}

	modelName, err := c.store.CurrentModel(controllerName)
	if err != nil {
		return base.ModelStatus{}, errors.Annotate(err, "failed to get current model from local store")
	}

	// Retrieve account credentials for controller (from ~/.local/share/juju/accounts.yml)
	account, err := c.store.AccountDetails(controllerName)
	if err != nil {
		panic(err)
	}
	log.Printf("Using default user: %s\n", account.User)

	model, err := c.store.ModelByName(controllerName, modelName)
	if err != nil {
		return base.ModelStatus{}, errors.Annotate(err, "failed to get model by name from local store")
	}

	conn, err := api.Open(context.Background(), &api.Info{
		Addrs:       controller.APIEndpoints,
		CACert:      controller.CACert,
		SNIHostName: controller.PublicDNSName, // optional
		Tag:         names.NewUserTag(account.User),
		Password:    account.Password,
	}, api.DefaultDialOpts())
	if err != nil {
		return base.ModelStatus{}, errors.Annotate(err, "failed to create API connection")
	}

	// Access the ModelManager API
	mgr := modelmanager.NewClient(conn)
	defer mgr.Close()

	// Retrieve list of models
	ctx := context.Background()
	modelStatus, err := mgr.ModelStatus(ctx, names.NewModelTag(model.ModelUUID))
	if err != nil {
		log.Fatalf("failed to fetch models: %v", err)
	}

	return modelStatus[0], nil
}
