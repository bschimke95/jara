package startup

import (
	"github.com/bschimke95/jara/pkg/app"
	"github.com/bschimke95/jara/pkg/pages/model"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	spinner  spinner.Model
	provider app.Provider
}

func New() Model {
	return Model{
		spinner: spinner.New(),
	}
}

func (m Model) Init() tea.Cmd {
	return startup()
}

func (m Model) View() string {
	return m.spinner.View()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		// The keymap is not yet loaded but we still want to let the user quit
		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))):
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.spinner, _ = m.spinner.Update(msg)

	case spinner.TickMsg:
		m.spinner, _ = m.spinner.Update(msg)

	case setupMsg:
		// Setup completed - return the new model
		return model.New(m.provider), nil
	}

	return m, nil
}

// setupMsg is a custom message type to indicate setup completion
type setupMsg struct{}

// startup is a command to refresh Juju model data
// TODO(ben): Setup should load the last used model or default model and startup configuration.
func startup() tea.Cmd {
	return func() tea.Msg {
		return setupMsg{}
	}
}
