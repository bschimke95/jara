package env

import (
	"context"

	"github.com/bschimke95/jara/pkg/client/juju"
)

type Provider interface {
	Config() Config
	JujuClient() juju.JujuClient
	Context() context.Context
}

// NewProvider creates a new Provider implementation
func DefaultProvider() Provider {
	return &ProviderImpl{
		config:     DefaultConfig,
		jujuClient: juju.NewMockClient(),
		ctx:        context.Background(),
	}
}

// ProviderImpl implements the Provider interface
type ProviderImpl struct {
	config     Config
	jujuClient juju.JujuClient
	ctx        context.Context
}

func (p *ProviderImpl) Config() Config {
	return p.config
}

func (p *ProviderImpl) JujuClient() juju.JujuClient {
	return p.jujuClient
}

func (p *ProviderImpl) Context() context.Context {
	return p.ctx
}

var _ Provider = &ProviderImpl{}
