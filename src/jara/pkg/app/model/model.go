package model

import (
	"fmt"

	"github.com/bschimke95/jara/pkg/env"
	"github.com/bschimke95/jara/pkg/types/juju"
	"github.com/bschimke95/jara/pkg/ui"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	provider   env.Provider
	controller juju.Controller
	model      juju.Model
	modelUUID  string // Store the model UUID for refreshing
}

func New(provider env.Provider) *Model {
	return &Model{
		provider: provider,
	}
}

func (m Model) Init() tea.Cmd {
	return m.refresh()
}

func (m *Model) View() string {
	layout := ui.NewLayout(ui.WithHeader(ui.HeaderInfo{
		Controller: m.controller,
		KeyHints:   env.DefaultKeyMap,
	}))

	// Model name and status
	var content string
	content += fmt.Sprintf("Model: %s\n", m.model.Name)
	content += fmt.Sprintf("Status: %s\n", m.model.Status)

	// Applications list
	content += "\nApplications:\n"
	for _, app := range m.model.Applications {
		content += fmt.Sprintf("- %s\n", app.Name)
	}

	return layout.Render(content)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case jujuModelMsg:
		m.model = msg.model
		m.controller = msg.controller
		return m, nil
	}

	return m, nil
}

type jujuModelMsg struct {
	model      juju.Model
	controller juju.Controller
}

func (m *Model) refresh() tea.Cmd {
	return func() tea.Msg {
		// Get the Juju client from the provider
		jujuClient := m.provider.JujuClient()
		if jujuClient == nil {
			// Return an empty model if we can't get the client
			return jujuModelMsg{model: juju.Model{}}
		}

		// First, get the current controller
		controller, err := jujuClient.CurrentController(m.provider.Context())
		if err != nil {
			// Return an empty model if there's an error
			// TODO(ben): Handle error
			return jujuModelMsg{model: juju.Model{}}
		}

		// If no specific model UUID is requested, use the current model
		if m.modelUUID == "" {
			model, err := jujuClient.CurrentModel(m.provider.Context())
			if err != nil {
				return jujuModelMsg{model: juju.Model{}}
			}
			return jujuModelMsg{
				model:      model,
				controller: controller,
			}
		}

		// Get all models and find the one with matching UUID
		models, err := jujuClient.Models(m.provider.Context())
		if err != nil {
			return jujuModelMsg{model: juju.Model{}}
		}

		for _, model := range models {
			if model.ModelUUID == m.modelUUID {
				return jujuModelMsg{
					model:      model,
					controller: controller,
				}
			}
		}

		// If we get here, the model wasn't found
		return jujuModelMsg{model: juju.Model{}}
	}
}

// SetModelUUID sets the model UUID to be loaded
func (m *Model) SetModelUUID(uuid string) {
	m.modelUUID = uuid
}
