package views

import (
	"fmt"
	"strings"

	"github.com/canonical/k8s/pkg/ui"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AppModel represents the main application state
type AppModel struct {
	spinner    spinner.Model
	status     string
	keys       ui.KeyMap
	frontpage  FrontpageModel
	loading    bool
}

// Init implements tea.Model interface
func (m AppModel) Init() tea.Cmd {
	return refreshJujuModels()
}

type JujuModel struct {
	Name         string
	Status       string
	Applications []JujuApplication
}

type JujuApplication struct {
	Name   string
	Units  int
	Status string
}

// InitialModel creates the initial application model
func InitialModel() AppModel {
	return AppModel{
		spinner: spinner.New(),
		status:  "Loading Juju models...",
		keys:    ui.DefaultKeyMap,
		frontpage: NewFrontpageModel(),
		loading: true,
	}
}

// View renders the application UI
func (m AppModel) View() string {
	if m.loading {
		var sb strings.Builder

		header := lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Render("JARA - Juju Application Runner and Analyzer")

		status := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render(fmt.Sprintf("Status: %s", m.status))

		help := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render(fmt.Sprintf("%s, %s", ui.DefaultKeyMap.Quit.Help(), ui.DefaultKeyMap.Refresh.Help()))

		loading := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render(m.spinner.View())

		sb.WriteString(fmt.Sprintf("%s\n%s\n%s\n%s", header, status, loading, help))
		return sb.String()
	}

	return m.frontpage.View()
}

// Update handles application events
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Refresh):
			return m, refreshJujuModels()
		}

	case tea.WindowSizeMsg:
		m.spinner, _ = m.spinner.Update(msg)

	case spinner.TickMsg:
		m.spinner, _ = m.spinner.Update(msg)

	case jujuModelsMsg:
		m.frontpage.models = msg.Models
		m.status = "Ready"
		m.loading = false
	}

	return m, nil
}

// jujuModelsMsg is a custom message type for Juju model updates
type jujuModelsMsg struct {
	Models []JujuModel
}

// refreshJujuModels is a command to refresh Juju model data
func refreshJujuModels() tea.Cmd {
	// TODO: Implement actual Juju model fetching
	return func() tea.Msg {
		return jujuModelsMsg{
			Models: []JujuModel{
				{
					Name:   "production",
					Status: "active",
					Applications: []JujuApplication{
						{Name: "webapp", Units: 3, Status: "active"},
						{Name: "database", Units: 1, Status: "active"},
					},
				},
			},
		}
	}
}
