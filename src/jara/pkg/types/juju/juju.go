// Package juju provides types for interacting with Juju models and controllers.
package juju

// Unit represents a Juju unit.
type Unit struct {
	Name string `json:"name"`
}

// Application represents a Juju application.
type Application struct {
	Name  string `json:"name"`
	Units []Unit `json:"units"`
}

// Model represents a Juju model.
type Model struct {
	Name         string        `json:"name"`
	ModelUUID    string        `json:"model_uuid"`
	Status       string        `json:"status"`
	Applications []Application `json:"applications"`
}

// Controller represents a Juju controller.
type Controller struct {
	Name          string   `json:"name"`
	APIEndpoints  []string `json:"api_endpoints"`
	CACert        string   `json:"ca_cert"`
	PublicDNSName string   `json:"public_dns_name"`
}
