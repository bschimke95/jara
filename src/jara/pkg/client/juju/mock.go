package juju

import (
	"context"

	"github.com/bschimke95/jara/pkg/types/juju"
)

// Mock contains predefined responses for the mock client
type Mock struct {
	// Return values
	Controller   juju.Controller
	Models       []juju.Model
	CurrentModel juju.Model

	// Error values
	ControllerErr   error
	ModelsErr       error
	CurrentModelErr error
}

// MockClient implements the JujuClient interface for testing
type MockClient struct {
	// Store method calls for verification
	CurrentControllerCalls int
	ModelsCalls            int
	CurrentModelCalls      int

	// Mock contains the responses to return
	Mock Mock
}

// NewMockClient creates a new mock client with default responses
func NewMockClient() JujuClient {
	return &MockClient{
		Mock: Mock{
			Controller: juju.Controller{
				Name:          "mock-controller",
				APIEndpoints:  []string{"10.0.0.1:17070"},
				CACert:        "mock-ca-cert",
				PublicDNSName: "controller.example.com",
			},
			Models: []juju.Model{
				{
					Name:      "default",
					ModelUUID: "11111111-1111-1111-1111-111111111111",
					Status:    "available",
					Applications: []juju.Application{
						{
							Name: "ubuntu",
							Units: []juju.Unit{
								{Name: "ubuntu/0"},
								{Name: "ubuntu/1"},
							},
						},
					},
				},
				{
					Name:      "test",
					ModelUUID: "22222222-2222-2222-2222-222222222222",
					Status:    "available",
					Applications: []juju.Application{
						{
							Name: "mysql",
							Units: []juju.Unit{
								{Name: "mysql/0"},
							},
						},
					},
				},
			},
			CurrentModel: juju.Model{
				Name:      "default",
				ModelUUID: "11111111-1111-1111-1111-111111111111",
				Status:    "available",
				Applications: []juju.Application{
					{
						Name: "ubuntu",
						Units: []juju.Unit{
							{Name: "ubuntu/0"},
							{Name: "ubuntu/1"},
						},
					},
					{
						Name: "mysql",
						Units: []juju.Unit{
							{Name: "mysql/0"},
						},
					},
				},
			},
		},
	}
}

// CurrentController implements the JujuClient interface
func (m *MockClient) CurrentController(ctx context.Context) (juju.Controller, error) {
	m.CurrentControllerCalls++
	return m.Mock.Controller, m.Mock.ControllerErr
}

// Models implements the JujuClient interface
func (m *MockClient) Models(ctx context.Context) ([]juju.Model, error) {
	m.ModelsCalls++
	return m.Mock.Models, m.Mock.ModelsErr
}

// CurrentModel implements the JujuClient interface
func (m *MockClient) CurrentModel(ctx context.Context) (juju.Model, error) {
	m.CurrentModelCalls++
	return m.Mock.CurrentModel, m.Mock.CurrentModelErr
}

// Ensure MockClient implements JujuClient interface
var _ JujuClient = &MockClient{}
