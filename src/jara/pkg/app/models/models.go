package model

import (
	"fmt"

	"github.com/bschimke95/jara/pkg/env"
	"github.com/bschimke95/jara/pkg/types/juju"
	"github.com/bschimke95/jara/pkg/ui"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	provider env.Provider
	model    juju.Model
}

func New(provider env.Provider) *Model {
	return &Model{
		provider: provider,
	}
}

func (m Model) Init() tea.Cmd {
	// TODO(ben): Verify model name exists otherwise navigate to model list?
	return m.refresh()
}

func (m *Model) View() string {
	layout := ui.NewLayout()

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
	case tea.KeyMsg:
		// TODO(ben): Create a "handleKeyInput" function that handles default + model specific actions.
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		}
	case jujuModelMsg:
		m.model = msg.model
		return m, nil
	}

	return m, nil
}

type jujuModelMsg struct {
	model juju.Model
}

func (m *Model) refresh() tea.Cmd {
	return func() tea.Msg {
		// Get the Juju client from the provider
		jujuClient := m.provider.JujuClient()
		if jujuClient == nil {
			// Return an empty model if we can't get the client
			return jujuModelMsg{model: juju.Model{}}
		}

		// Get the current model
		model, err := jujuClient.CurrentModel(m.provider.Context(), "")
		if err != nil {
			// Return an empty model if there's an error
			// TODO(ben): Handle error
			return jujuModelMsg{model: juju.Model{}}
		}

		return jujuModelMsg{
			model: model,
		}
	}
}
