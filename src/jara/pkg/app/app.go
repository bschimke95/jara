package app

import (
	"github.com/bschimke95/jara/pkg/client/juju"
)

type Provider interface {
	Config() Config
	JujuClient() *juju.JujuClient
}
