package app

import (
	"github.com/bschimke95/jara/pkg/client/juju"
	"github.com/juju/juju/jujuclient"
)

// ProviderImpl implements the Provider interface
type ProviderImpl struct {
	config     Config
	jujuClient *juju.JujuClient
}

// Config implements the Provider interface
func (p *ProviderImpl) Config() Config {
	return p.config
}

// JujuClient implements the Provider interface
func (p *ProviderImpl) JujuClient() *juju.JujuClient {
	return p.jujuClient
}

// NewProvider creates a new Provider implementation
func DefaultProvider() Provider {
	clientStore := jujuclient.NewFileClientStore()
	return &ProviderImpl{
		config:     DefaultConfig,
		jujuClient: juju.NewJujuClient(juju.NewJujuClientStoreAdapter(clientStore)),
	}
}
