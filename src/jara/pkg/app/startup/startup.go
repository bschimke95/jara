package startup

import (
	"time"

	"github.com/bschimke95/jara/pkg/app/models"
	"github.com/bschimke95/jara/pkg/app/navigation"
	"github.com/bschimke95/jara/pkg/env"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	spinner spinner.Model
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
	case spinner.TickMsg:
		m.spinner, _ = m.spinner.Update(msg)
		return m, tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
			return spinner.TickMsg{Time: t}
		})

	case setupMsg:
		// Setup completed - navigate to models view without adding startup to history
		models := models.New(msg.App)
		return nil, navigation.GoTo(models, navigation.GoToOpts{
			SkipHistory: true,
		})
	}

	return m, nil
}

// setupMsg is a custom message type to indicate setup completion
type setupMsg struct {
	App env.Provider
}

// startup is a command to refresh Juju model data
// TODO(ben): Setup should load the last used model or default model and startup configuration.
func startup() tea.Cmd {
	return func() tea.Msg {
		// TODO(ben): Should return spinner.TickMsg while loading
		return setupMsg{
			App: env.DefaultProvider(),
		}
	}
}
