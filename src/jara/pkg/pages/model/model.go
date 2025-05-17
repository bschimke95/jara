package model

import (
	"fmt"

	"github.com/bschimke95/jara/pkg/app"
	"github.com/bschimke95/jara/pkg/types/juju"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	provider app.Provider
	model    juju.Model
}

func New(provider app.Provider) *Model {
	return &Model{
		provider: provider,
	}
}

func (m Model) Init() tea.Cmd {
	// TODO(ben): Verify model name exists otherwise navigate to model list?
	return refreshModel()
}

func (m *Model) View() string {
	var content string

	// Model name and status
	content += fmt.Sprintf("Model: %s\n", m.model.Name)
	content += fmt.Sprintf("Status: %s\n", m.model.Status)

	// Applications list
	content += "\nApplications:\n"
	for _, app := range m.model.Applications {
		content += fmt.Sprintf("- %s\n", app.Name)
	}

	return content
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
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

func refreshModel() tea.Cmd {
	return func() tea.Msg {
		return jujuModelMsg{
			model: juju.Model{
				Name: "test",
				Applications: []juju.Application{
					{
						Name: "test-app",
					},
				},
			},
		}
	}
}
