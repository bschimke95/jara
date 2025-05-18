package juju

import (
	"github.com/bschimke95/jara/pkg/types/juju"
	jujuclient "github.com/juju/juju/jujuclient"
)

// jujuClientStoreAdapter adapts jujuclient.ClientStore to implement the JujuStore interface
type jujuClientStoreAdapter struct {
	store jujuclient.ClientStore
}

// NewJujuClientStoreAdapter creates a new adapter for jujuclient.ClientStore
func NewJujuClientStoreAdapter(store jujuclient.ClientStore) JujuStore {
	return &jujuClientStoreAdapter{store: store}
}

// CurrentController implements JujuStore
func (a *jujuClientStoreAdapter) CurrentController() (string, error) {
	return a.store.CurrentController()
}

// ControllerByName implements JujuStore
func (a *jujuClientStoreAdapter) ControllerByName(name string) (*juju.Controller, error) {
	details, err := a.store.ControllerByName(name)
	if err != nil {
		return nil, err
	}
	return &juju.Controller{
		APIEndpoints: details.APIEndpoints,
		CACert:       details.CACert,
		PublicDNSName: details.APIEndpoints[0], // Use first endpoint as public DNS name
	}, nil
}

// CurrentModel implements JujuStore
func (a *jujuClientStoreAdapter) CurrentModel(controllerName string) (string, error) {
	return a.store.CurrentModel(controllerName)
}

// ModelByName implements JujuStore
func (a *jujuClientStoreAdapter) ModelByName(controllerName, modelName string) (*juju.Model, error) {
	details, err := a.store.ModelByName(controllerName, modelName)
	if err != nil {
		return nil, err
	}
	// Convert jujuclient.ModelDetails to juju.Model
	return &juju.Model{
		Name:      modelName,
		ModelUUID: details.ModelUUID,
		// Initialize empty applications slice - this can be populated later if needed
		Applications: []juju.Application{},
	}, nil
}

// AccountDetails implements JujuStore
func (a *jujuClientStoreAdapter) AccountDetails(controllerName string) (*juju.Account, error) {
	details, err := a.store.AccountDetails(controllerName)
	if err != nil {
		return nil, err
	}
	// Convert jujuclient.AccountDetails to juju.Account
	return &juju.Account{
		User:     details.User,
		// Password might not be available in jujuclient.AccountDetails
		// You might need to handle this differently based on your requirements
		Password: "",
	}, nil
}
