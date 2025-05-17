// An abstraction around the Juju API model
package juju

type Application struct {
	Name string
}

type Model struct {
	Name         string
	Status       string
	Applications []Application
}
